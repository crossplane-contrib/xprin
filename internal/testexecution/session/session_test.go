/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package session

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testexecution/runner"
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const testSuiteFile = "/suite_xprin.yaml"

// mockTestCaseRunner is the TestCaseRunner of the tests: a test case passes, or has the result that runTestCaseFunc gives. Like
// the real one, it skips a test case that needs one that failed or was skipped, and it records what the session gave it.
type mockTestCaseRunner struct {
	runTestCaseFunc func(testCase api.TestCase) *engine.TestCaseResult

	mu        sync.Mutex
	contexts  map[string]runner.TestSuiteContext // by the ID of the test case
	completed map[string][]string                // the IDs of the results that the test case was given
}

func newMockTestCaseRunner() *mockTestCaseRunner {
	return &mockTestCaseRunner{contexts: map[string]runner.TestSuiteContext{}, completed: map[string][]string{}}
}

func (m *mockTestCaseRunner) RunTestCase(testSuiteContext runner.TestSuiteContext, testCase api.TestCase, completed map[string]*engine.TestCaseResult) *engine.TestCaseResult {
	m.mu.Lock()

	m.contexts[testCase.ID] = testSuiteContext

	for id := range completed {
		m.completed[testCase.ID] = append(m.completed[testCase.ID], id)
	}

	m.mu.Unlock()

	if _, skipReasons := engine.GetNeedsResults(testCase.Needs, completed); len(skipReasons) > 0 {
		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false).Skip(skipReasons...)
	}

	if m.runTestCaseFunc != nil {
		return m.runTestCaseFunc(testCase)
	}

	return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false).Complete()
}

// runTestsSerial sets the session up and runs its test cases with one worker, which is the order of a serial run. It
// fails the test if the set-up or the scheduler fail, and returns what Finish returned: nil, or ErrTestsFailed.
func runTestsSerial(t *testing.T, s *Session) error {
	t.Helper()

	require.NoError(t, s.Setup())

	// Finish removes it too: this is for a run that ends without it
	defer s.removeSavedOutputsDir()

	finishErrs, err := scheduler.Run([]scheduler.Job{s}, 1)
	require.NoError(t, err)

	return finishErrs[0]
}

// newTestSession returns the session of testSuite, and the mock test case runner that it runs its test cases with.
func newTestSession(options *testexecutionOptions.Options, testSuite *testsuite.TestSuite) (*Session, *mockTestCaseRunner) {
	mock := newMockTestCaseRunner()

	return New(options, testSuite, mock), mock
}

func TestJob_APanicFailsOnlyThatTestCase(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a"},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "d"},
	}}

	r, mock := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var out bytes.Buffer

	r.output = &out
	mock.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		if testCase.ID == "b" {
			panic("boom")
		}

		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
	}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r}, 4)
	require.NoError(t, err)

	require.Len(t, finishErrs, 1)
	require.ErrorIs(t, finishErrs[0], ErrTestsFailed)
	assert.Contains(t, out.String(), "--- FAIL: B")
	assert.Contains(t, out.String(), "panic: boom")
	assert.Contains(t, out.String(), "--- SKIP: C", "what needs it is skipped, as after any failure")
	assert.NotContains(t, out.String(), "--- FAIL: A")
	assert.NotContains(t, out.String(), "--- FAIL: D", "the others are not affected")
	assert.NotContains(t, out.String(), "goroutine", "the stack is for --debug")
}

func TestSession_GivesTheRunnerTheTestSuiteContextAndTheResultsThatAreDone(t *testing.T) {
	// B is first in the file, and runs second: it needs A
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "A", ID: "a"},
	}}
	suite := testsuite.New("/tests/a_xprin.yaml", spec)
	s, mock := newTestSession(&testexecutionOptions.Options{}, suite)
	s.output = io.Discard
	s.fs = afero.NewMemMapFs()

	require.NoError(t, s.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{s}, 1)
	require.NoError(t, err)
	require.NoError(t, finishErrs[0])

	require.Len(t, mock.contexts, 2)

	for id, testSuiteContext := range mock.contexts {
		assert.Same(t, suite, testSuiteContext.TestSuite, "%s gets the file", id)
		assert.NotEmpty(t, testSuiteContext.SavedOutputsDir, "%s gets the directory of the saved outputs", id)
	}

	assert.Equal(t, 1, mock.contexts["a"].Index, "the numbers are the ones of the order that they run in, which starts at 1")
	assert.Equal(t, 2, mock.contexts["b"].Index)
	assert.Empty(t, mock.completed["a"], "nothing is done when A runs")
	assert.Equal(t, []string{"a"}, mock.completed["b"], "A is done when B runs")
}

func TestJob_FinishReturnsNothingWhenNothingFailed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a"}}}

	r, _ := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))
	r.output = &bytes.Buffer{}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r}, 2)
	require.NoError(t, err)

	require.Len(t, finishErrs, 1)
	require.NoError(t, finishErrs[0], "nothing failed")
}

// timings are what differs between two runs of the same test cases.
var timings = regexp.MustCompile(`\d+\.\d+s`)

// runWithMockResults runs the test cases of the spec with mock results: the ones in failing fail, and the others pass.
// With 0 workers it runs them with runTestsSerial. With more, the steps of the runner are run by the scheduler, with that
// many workers. The test cases take a little time, different for each, so that they don't finish in the order they
// started.
// It returns what was printed and the IDs of the ones that ran.
func runWithMockResults(t *testing.T, spec *api.TestSuiteSpec, workers int, failing ...string) (output string, ran []string) {
	t.Helper()

	r, tcRunner := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var (
		out bytes.Buffer
		mu  sync.Mutex
	)

	r.output = &out
	tcRunner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		time.Sleep(time.Duration(len(testCase.ID)%4) * time.Millisecond)

		mu.Lock()

		ran = append(ran, testCase.ID)
		mu.Unlock()

		result := engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
		if slices.Contains(failing, testCase.ID) {
			return result.Fail(errors.New("boom"))
		}

		return result
	}

	if workers == 0 {
		_ = runTestsSerial(t, r) // it fails when a test case does: what matters is what ran, and what was printed

		return out.String(), ran
	}

	require.NoError(t, r.Setup())

	_, err := scheduler.Run([]scheduler.Job{r}, workers)
	require.NoError(t, err)

	return out.String(), ran
}

func TestFileRun_Result(t *testing.T) {
	options := &testexecutionOptions.Options{
		ShowRender:   false,
		ShowValidate: false,
		Verbose:      false,
		Debug:        false,
	}
	cases := []struct {
		name      string
		testCases []api.TestCase
		mockErrs  []error
		wantFails int
		wantPass  int
	}{
		{
			name:      "no tests",
			testCases: nil,
			mockErrs:  nil,
			wantFails: 0,
			wantPass:  0,
		},
		{
			name:      "one passing test",
			testCases: []api.TestCase{{Name: "test1"}},
			mockErrs:  []error{nil},
			wantFails: 0,
			wantPass:  1,
		},
		{
			name:      "one failing test",
			testCases: []api.TestCase{{Name: "test1"}},
			mockErrs:  []error{errors.New("fail")},
			wantFails: 1,
			wantPass:  0,
		},
		{
			name:      "multiple tests",
			testCases: []api.TestCase{{Name: "test1"}, {Name: "test2"}},
			mockErrs:  []error{nil, errors.New("fail")},
			wantFails: 1,
			wantPass:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			var tests []api.TestCase
			for _, test := range tc.testCases {
				tests = append(tests, api.TestCase{
					Name: test.Name,
				})
			}

			testSuiteSpec := &api.TestSuiteSpec{
				Tests: tests,
			}

			runner, tcRunner := newTestSession(options, testsuite.New(testSuiteFile, testSuiteSpec))
			runner.output = &buf

			// Mock the runTestCaseFunc to return controlled results
			call := 0
			tcRunner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
				err := tc.mockErrs[call]
				call++

				return createTestCaseResult(testCase.Name, false, err)
			}

			err := runTestsSerial(t, runner)
			if tc.wantFails > 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output was generated
			output := buf.String()
			if tc.wantFails > 0 {
				assert.Contains(t, output, "FAIL", "output should contain FAIL for failed tests")
			} else {
				assert.NotContains(t, output, "FAIL", "output should not contain FAIL for passing tests")
			}
		})
	}
}

func TestFileRun_Output(t *testing.T) {
	options := &testexecutionOptions.Options{
		ShowRender:   false,
		ShowValidate: false,
		Verbose:      false,
		Debug:        false,
	}

	testCases := []struct {
		name       string
		runFunc    func(tc api.TestCase) *engine.TestCaseResult
		suite      *api.TestSuiteSpec
		wantError  bool
		wantStatus string
		wantOutput []string
		verbose    bool // add verbose field
	}{
		{
			name: "all pass",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, nil)
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "xr.yaml",
							Composition: "comp.yaml",
							Functions:   "functions.yaml",
							CRDs:        []string{"crd.yaml"},
						},
					},
				},
			},
			wantError:  false,
			wantStatus: "PASS",
			verbose:    false,
		},
		{
			name: "one fails",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, errors.New("fail"))
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "xr.yaml",
							Composition: "comp.yaml",
							Functions:   "functions.yaml",
							CRDs:        []string{"crd.yaml"},
						},
					},
				},
			},
			wantError:  true,
			wantStatus: "FAIL",
			verbose:    false,
		},
		{
			name:      "no tests",
			suite:     &api.TestSuiteSpec{Tests: []api.TestCase{}},
			wantError: false, // Empty test suite should not error
			verbose:   false,
		},
		{
			name: "verbose output includes RUN and hierarchy",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, true, nil)
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "foo.yaml",
							Composition: "bar.yaml",
							Functions:   "baz/",
						},
					},
				},
				Common: api.Common{},
			},
			wantOutput: []string{
				"=== RUN   test1",
				"--- PASS: test1 (0.00s)",
			},
			verbose: true,
		},
		{
			name: "non-verbose output prints group and subtest failures with durations",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, errors.New("fail: something went wrong"))
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "foo.yaml",
							Composition: "bar.yaml",
							Functions:   "baz/",
						},
					},
				},
				Common: api.Common{},
			},
			wantError:  true,
			wantStatus: "FAIL",
			wantOutput: []string{
				"--- FAIL: test1 (0.00s)",
				"fail: something went wrong",
			},
			verbose: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			runner, tcRunner := newTestSession(options, testsuite.New(testSuiteFile, tc.suite))
			runner.output = &buf
			tcRunner.runTestCaseFunc = tc.runFunc
			options.Verbose = tc.verbose // set per-test verbosity

			err := runTestsSerial(t, runner)
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantStatus != "" {
				out := buf.String()

				switch tc.wantStatus {
				case "FAIL":
					assert.True(t, err != nil || strings.Contains(out, "FAIL"), "Expected test failure or FAIL in output")
				case "PASS":
					assert.True(t, err == nil && !strings.Contains(out, "FAIL"), "Expected no test failures")
				}
			}

			if len(tc.wantOutput) > 0 {
				out := buf.String()
				for _, want := range tc.wantOutput {
					assert.Contains(t, out, want)
				}
			}
		})
	}

	t.Run("multiple tests mixed pass/fail, verbose output check", func(t *testing.T) {
		var buf bytes.Buffer

		options := &testexecutionOptions.Options{
			ShowRender:   false,
			ShowValidate: false,
			Verbose:      true,
			Debug:        false,
		}
		suite := &api.TestSuiteSpec{
			Tests: []api.TestCase{
				{Name: "test1", Inputs: api.Inputs{
					XR: "a",
				}},
				{Name: "test2", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
				{Name: "test3", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
			},
		}
		runner, tcRunner := newTestSession(options, testsuite.New(testSuiteFile, suite))
		runner.output = &buf

		// test1: pass, test2: fail, test3: pass
		call := 0
		results := []error{nil, errors.New("fail in test2"), nil}
		tcRunner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, options.Verbose, err)
		}
		err := runTestsSerial(t, runner)
		require.Error(t, err)

		out := buf.String()
		// Check individual test results
		assert.Contains(t, out, "test1", "output should contain test1")
		assert.Contains(t, out, "test2", "output should contain test2")
		assert.Contains(t, out, "test3", "output should contain test3")
		// Should have failed test indicated by error and output
		assert.Contains(t, out, "FAIL", "output should contain FAIL for failed test")
	})

	t.Run("multiple groups mixed pass/fail, non-verbose output check", func(t *testing.T) {
		var buf bytes.Buffer

		options := &testexecutionOptions.Options{
			ShowRender:   false,
			ShowValidate: false,
			Verbose:      false,
			Debug:        false,
		}
		suite := &api.TestSuiteSpec{
			Tests: []api.TestCase{
				{Name: "test1", Inputs: api.Inputs{
					XR: "a",
				}},
				{Name: "test2", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
				{Name: "test3", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
			},
		}
		runner, tcRunner := newTestSession(options, testsuite.New(testSuiteFile, suite))
		runner.output = &buf

		// group1: pass, group2: fail, group3: pass
		call := 0
		results := []error{nil, errors.New("fail in group2"), nil}
		tcRunner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, false, err)
		}
		err := runTestsSerial(t, runner)
		require.Error(t, err)

		out := buf.String()
		// Only failed test should be shown in non-verbose mode
		assert.Contains(t, out, "--- FAIL: test2 (0.00s)")
		// Passing tests should not be shown in non-verbose mode
		assert.NotContains(t, out, "--- PASS: test1")
		assert.NotContains(t, out, "--- PASS: test3")
		// Error message should be present
		assert.Contains(t, out, "fail in group2")
	})
}

// The internal artifacts directory, where the test cases of a run keep their outputs for the ones that need them, is
// created by Setup and removed by Finish.
func TestSessionSetup_CreatesTheInternalArtifactsDirectory(t *testing.T) {
	r, _ := newTestSession(&testexecutionOptions.Options{}, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "test1"}}}))
	r.fs = afero.NewMemMapFs()
	r.output = io.Discard

	assert.Nil(t, r.env, "there is no env before the run is set up")

	require.NoError(t, r.Setup())

	exists, err := afero.DirExists(r.fs, r.env.savedOutputsDir)
	require.NoError(t, err)
	assert.True(t, exists, "Setup should create the directory")

	dir := r.env.savedOutputsDir

	require.NoError(t, r.Finish())

	exists, err = afero.DirExists(r.fs, dir)
	require.NoError(t, err)
	assert.False(t, exists, "Finish should remove the directory")
}

func TestSessionSetup_RequiresTheFileAndTheSpec(t *testing.T) {
	for name, tt := range map[string]struct {
		file string
		spec *api.TestSuiteSpec
		want string
	}{
		"no path":          {file: "", spec: &api.TestSuiteSpec{}, want: "testsuite file path is required"},
		"no specification": {file: testSuiteFile, spec: nil, want: "testsuite specification is required"},
	} {
		t.Run(name, func(t *testing.T) {
			s, _ := newTestSession(&testexecutionOptions.Options{}, testsuite.New(tt.file, tt.spec))

			require.EqualError(t, s.Setup(), tt.want)
			assert.Nil(t, s.env, "nothing is made for a file that is not set up")
		})
	}
}

func TestFileRun_Needs(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a"},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "d", Needs: []string{"a", "b"}},
		{Name: "E", ID: "e"},
	}}

	t.Run("nothing fails: everything runs", func(t *testing.T) {
		out, ran := runWithMockResults(t, spec, 0)

		assert.Equal(t, []string{"a", "b", "c", "d", "e"}, ran)
		assert.NotContains(t, out, "SKIP")
		assert.NotContains(t, out, "FAIL")
	})

	t.Run("the dependents of a failed testcase are skipped, with the reason on follow-up lines, one per need", func(t *testing.T) {
		out, ran := runWithMockResults(t, spec, 0, "a")

		assert.Equal(t, []string{"a", "e"}, ran)
		assert.Contains(t, out, "--- FAIL: A")
		assert.Contains(t, out, "--- SKIP: B")
		assert.Contains(t, out, "    [!] needs failed test case \"a\"\n")
		assert.Contains(t, out, "--- SKIP: C")
		assert.Contains(t, out, "    [!] needs skipped test case \"b\"\n")
		assert.Contains(t, out, "--- SKIP: D")
		assert.Contains(t, out, "    [!] needs failed test case \"a\"\n    [!] needs skipped test case \"b\"\n")
		assert.NotContains(t, out, "--- SKIP: E")
	})
}

func TestFileRun_Order(t *testing.T) {
	// "a" is listed first, but needs "c" and "b", which are listed after it
	tests := []api.TestCase{
		{Name: "A", ID: "a", Needs: []string{"c", "b"}},
		{Name: "B", ID: "b"},
		{Name: "C", ID: "c"},
		{Name: "D", ID: "d"},
	}

	t.Run("the test cases run after the ones they need, the rest in the order of the file", func(t *testing.T) {
		_, ran := runWithMockResults(t, &api.TestSuiteSpec{Tests: tests}, 0)

		assert.Equal(t, []string{"b", "c", "a", "d"}, ran)
	})

	t.Run("a test case that needs one listed after it is skipped when that one fails", func(t *testing.T) {
		out, ran := runWithMockResults(t, &api.TestSuiteSpec{Tests: tests}, 0, "b")

		assert.Equal(t, []string{"b", "c", "d"}, ran)
		assert.Contains(t, out, "--- SKIP: A")
		assert.Contains(t, out, `[!] needs failed test case "b"`)
	})
}

// With the scheduler the output is the one of a serial run, whatever the number of workers and the order the test
// cases finish in. Run with -race.
func TestRunner_StepsRunByTheSchedulerGiveTheOutputOfASerialRun(t *testing.T) {
	// One of the test cases is listed after the ones that need it, and some are independent
	chain := []api.TestCase{
		{Name: "A", ID: "a", Needs: []string{"c", "b"}},
		{Name: "B", ID: "b"},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "dd", Needs: []string{"a"}},
		{Name: "E", ID: "eee"},
		{Name: "F", ID: "ffff", Needs: []string{"b", "eee"}},
	}

	many := make([]api.TestCase, 0, 40)

	for i := range 40 {
		testCase := api.TestCase{Name: fmt.Sprintf("test %d", i), ID: fmt.Sprintf("t%d", i)}
		if i%5 == 4 {
			testCase.Needs = []string{fmt.Sprintf("t%d", i-3), fmt.Sprintf("t%d", i-1)}
		}

		many = append(many, testCase)
	}

	for name, tt := range map[string]struct {
		tests   []api.TestCase
		failing []string
	}{
		"everything passes":                          {tests: chain},
		"a failure skips the ones that need it":      {tests: chain, failing: []string{"b"}},
		"a failure in the middle of a chain":         {tests: chain, failing: []string{"a"}},
		"many test cases, some of them independent":  {tests: many, failing: []string{"t1", "t10"}},
		"many test cases, and none of them fail":     {tests: many},
		"every test case fails, so every need fails": {tests: chain, failing: []string{"a", "b", "c", "dd", "eee", "ffff"}},
	} {
		t.Run(name, func(t *testing.T) {
			spec := &api.TestSuiteSpec{Tests: tt.tests}

			serial, serialRan := runWithMockResults(t, spec, 0, tt.failing...)
			require.NotEmpty(t, serial)

			for _, workers := range []int{1, 2, 8, 16} {
				got, ran := runWithMockResults(t, spec, workers, tt.failing...)

				assert.Equal(t, timings.ReplaceAllString(serial, "N"), timings.ReplaceAllString(got, "N"), "workers: %d", workers)
				assert.ElementsMatch(t, serialRan, ran, "workers: %d", workers)
			}
		})
	}
}

// createTestCaseResult is a helper function to create TestCaseResult with common defaults.
func createTestCaseResult(name string, verbose bool, err error) *engine.TestCaseResult {
	result := engine.NewTestCaseResult(name, "", verbose, false, false, false, false)
	if err != nil {
		return result.Fail(err)
	}

	return result.Complete()
}

// interruptedRunOutput runs a file of three test cases, where the first ends as given and the other two did not run
// because the run was stopped, and returns what was printed.
func interruptedRunOutput(t *testing.T, first func(*engine.TestCaseResult) *engine.TestCaseResult) string {
	t.Helper()

	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a"}, {Name: "B", ID: "b"}, {Name: "C", ID: "c"}}}
	r, tcRunner := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var out bytes.Buffer

	r.output = &out
	tcRunner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		result := engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
		if testCase.ID == "a" {
			return first(result)
		}

		return result.Skip(runner.ErrInterrupted)
	}

	_ = runTestsSerial(t, r)

	return timings.ReplaceAllString(out.String(), "Xs")
}

func TestFileRun_Interrupted(t *testing.T) {
	t.Run("the test cases that did not run are not printed one by one, and the file is not a pass", func(t *testing.T) {
		out := interruptedRunOutput(t, func(r *engine.TestCaseResult) *engine.TestCaseResult { return r })

		assert.NotContains(t, out, "interrupted\n", "the test cases that did not run are not printed one by one")
		assert.Contains(t, out, "FAIL\t/tests/a_xprin.yaml\t[interrupted]\n")
		assert.NotContains(t, out, "ok\t")
	})

	t.Run("a file with a failure is printed as it is without the interrupt", func(t *testing.T) {
		out := interruptedRunOutput(t, func(r *engine.TestCaseResult) *engine.TestCaseResult { return r.Fail(errors.New("boom")) })

		assert.Contains(t, out, "FAIL\t/tests/a_xprin.yaml\tXs\n")
		assert.NotContains(t, out, "[interrupted]")
	})
}
