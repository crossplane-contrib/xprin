/*
Copyright 2025 The Crossplane Authors.

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

package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const testSuiteFile = "/suite_xprin.yaml"

// createTestCaseResult is a helper function to create TestCaseResult with common defaults.
func createTestCaseResult(name string, verbose bool, err error) *engine.TestCaseResult {
	result := engine.NewTestCaseResult(name, "", verbose, false, false, false, false)
	if err != nil {
		return result.Fail(err)
	}

	return result.Complete()
}

// newMockRunner creates a test-specific Runner with mocked functions.
func newMockRunner(options *testexecutionOptions.Options, mocks ...func(*Runner)) *Runner {
	// Use default test suite file and spec for tests that don't need specific values
	testSuiteSpec := &api.TestSuiteSpec{Tests: []api.TestCase{}}
	r := NewRunner(options, testsuite.New(testSuiteFile, testSuiteSpec))

	// Set up default mock functions for testing
	r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) { return path, nil }
	r.verifyPathExists = func(_ string) error { return nil }
	r.copy = func(_, _ string, _ ...cp.Options) error { return nil }

	// Apply any custom mocks
	for _, mock := range mocks {
		mock(r)
	}

	return r
}

// makeOptions is a helper to create Options from a config and allow overrides for tests.
func makeOptions(cfg *config.Config, render, validate []string, overrides ...func(*testexecutionOptions.Options)) *testexecutionOptions.Options {
	// If render/validate are nil, extract from cfg.Subcommands (if present)
	if render == nil && cfg.Subcommands != nil && cfg.Subcommands.Render != "" {
		render = strings.Fields(cfg.Subcommands.Render)
	}

	if validate == nil && cfg.Subcommands != nil && cfg.Subcommands.Validate != "" {
		validate = strings.Fields(cfg.Subcommands.Validate)
	}
	// If still nil or empty, set to known defaults from config
	if len(render) == 0 {
		render = strings.Fields(config.DefaultRenderCmd)
	}

	if len(validate) == 0 {
		validate = strings.Fields(config.ValidateSubcommand + " " + config.ValidateFlags)
	}

	opt := &testexecutionOptions.Options{
		Dependencies: cfg.Dependencies,
		Render:       render,
		Validate:     validate,
		XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
		ShowRender:   true,
		ShowValidate: true,
		ShowHooks:    true,
		Verbose:      true,
		Debug:        false,
	}
	for _, fn := range overrides {
		fn(opt)
	}

	return opt
}

// timings are what differs between two runs of the same test cases.
var timings = regexp.MustCompile(`\d+\.\d+s`)

// runWithFakeResults runs the test cases of the spec with fake results: the ones in failing fail, and the others pass.
// With 0 workers it runs them with RunTests. With more, the steps of the runner are run by the scheduler, with that
// many workers. The test cases take a little time, different for each, so that they don't finish in the order they
// started.
// It returns what was printed and the IDs of the ones that ran.
func runWithFakeResults(t *testing.T, spec *api.TestSuiteSpec, workers int, failing ...string) (output string, ran []string) {
	t.Helper()

	r := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var (
		out bytes.Buffer
		mu  sync.Mutex
	)

	r.output = &out
	r.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
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
		_ = r.RunTests() // it fails when a test case does: what matters is what ran, and what was printed

		return out.String(), ran
	}

	require.NoError(t, r.setupTestSuite())

	_, err := scheduler.Run([]scheduler.Job{r.Job()}, workers)
	require.NoError(t, err)

	return out.String(), ran
}

func TestNewRunner(t *testing.T) {
	// Integration-style: take Render/Validate from config
	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
		Repositories: map[string]string{
			"my-claims-repo": "path/to/my/claims/repo",
		},
		Subcommands: &config.Subcommands{
			Render:   "render --foo bar",
			Validate: "validate --baz qux",
		},
	}
	// Simulate config-to-options conversion (string to []string)
	options := makeOptions(cfg, nil, nil, func(o *testexecutionOptions.Options) {
		o.Debug = true
	})
	runner := NewRunner(options, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}}))

	// Check the runner was created with the right settings
	assert.Equal(t, options, runner.Options)
	assert.Equal(t, testSuiteFile, runner.testSuite.File)
	assert.True(t, runner.ShowRender)
	assert.True(t, runner.ShowValidate)
	assert.True(t, runner.ShowHooks)
	assert.True(t, runner.Verbose)
	assert.True(t, runner.Debug)
	// runTestsFunc is nil by default and uses the default runTests implementation
	assert.Equal(t, cfg.Dependencies, runner.Dependencies)
	assert.Equal(t, strings.Fields(cfg.Subcommands.Render), runner.Render)
	assert.Equal(t, strings.Fields(cfg.Subcommands.Validate), runner.Validate)
}

func TestNewRunner_DefaultsIfNotSpecified(t *testing.T) {
	// If Render/Validate are not set in config, runner should get the defaults
	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
		Repositories: map[string]string{
			"my-claims-repo": "path/to/my/claims/repo",
		},
		Subcommands: &config.Subcommands{}, // No Render/Validate set
	}
	// Simulate config-to-options conversion (empty means use defaults)
	options := makeOptions(cfg, nil, nil)
	runner := NewRunner(options, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}}))

	// The runner should have the default Render/Validate commands (from runner or config loader)
	assert.NotEmpty(t, runner.Render)
	assert.NotEmpty(t, runner.Validate)
	// Optionally, check for specific defaults if known, e.g.:
	assert.Equal(t, strings.Fields(config.DefaultRenderCmd), runner.Render)
	assert.Equal(t, strings.Fields(config.ValidateSubcommand+" "+config.ValidateFlags), runner.Validate)
}

func TestNewRunner_SetsWorkingDirectory(t *testing.T) {
	// Test that the runner takes the directory of its test suite, and that runCommand uses it
	testSuiteFile := "/path/to/test/suite_xprin.yaml"
	expectedDir := "/path/to/test"

	options := &testexecutionOptions.Options{}
	runner := NewRunner(options, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}}))

	// Verify the directory of the test suite is right
	assert.Equal(t, expectedDir, runner.testSuite.GetDir(), "the directory should be computed from the file of the test suite")

	// Verify that runCommand sets cmd.Dir correctly by capturing it
	var capturedDir string

	runner.runCommand = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		// The original runCommand sets cmd.Dir to the directory of the test suite (captured in closure)
		// We verify this by checking that the closure has access to the correct value
		cmd.Dir = runner.testSuite.GetDir()
		capturedDir = cmd.Dir

		return []byte{}, nil
	}

	// Execute a command through runCommand
	_, _ = runner.runCommand("echo", "test")

	assert.Equal(t, expectedDir, capturedDir, "runCommand should set cmd.Dir to testsuite file directory")

	// Test with a different path to ensure it's computed correctly
	testSuiteFile2 := "/another/path/test_xprin.yaml"
	expectedDir2 := "/another/path"
	runner2 := NewRunner(options, testsuite.New(testSuiteFile2, &api.TestSuiteSpec{Tests: []api.TestCase{}}))
	assert.Equal(t, expectedDir2, runner2.testSuite.GetDir(), "the directory should be computed from the file of the test suite")
}

func TestRunTests(t *testing.T) {
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

			runner := NewRunner(options, testsuite.New(testSuiteFile, testSuiteSpec))
			runner.output = &buf

			// Mock the runTestCaseFunc to return controlled results
			call := 0
			runner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
				err := tc.mockErrs[call]
				call++

				return createTestCaseResult(testCase.Name, false, err)
			}

			err := runner.RunTests()
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

func TestRunTestsIntegration(t *testing.T) {
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
			name:      "nil TestSuiteSpec",
			wantError: true,
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

			runner := NewRunner(options, testsuite.New(testSuiteFile, tc.suite))
			runner.output = &buf
			runner.runTestCaseFunc = tc.runFunc
			runner.Verbose = tc.verbose // set per-test verbosity

			err := runner.RunTests()
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
		runner := NewRunner(options, testsuite.New(testSuiteFile, suite))
		runner.output = &buf

		// test1: pass, test2: fail, test3: pass
		call := 0
		results := []error{nil, errors.New("fail in test2"), nil}
		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, runner.Verbose, err)
		}
		err := runner.RunTests()
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
		runner := NewRunner(options, testsuite.New(testSuiteFile, suite))
		runner.output = &buf

		// group1: pass, group2: fail, group3: pass
		call := 0
		results := []error{nil, errors.New("fail in group2"), nil}
		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, false, err)
		}
		err := runner.RunTests()
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

// TestRunTestCase tests the core validation and execution logic.
func TestRunTestCase(t *testing.T) {
	validRenderYAML := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	fs := afero.NewMemMapFs()
	localCRDPath := "/crd.yaml"
	require.NoError(t, afero.WriteFile(fs, localCRDPath, []byte("dummy crd content"), 0o644))

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionOptions.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}

	cases := []struct {
		name      string
		testCase  api.TestCase
		setup     func(*Runner)
		wantError string
	}{
		// Validation Tests
		{
			name: "missing claim or xr",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			wantError: "missing mandatory field: either 'claim' or 'xr' must be specified",
		},
		{
			name: "xr field with missing xr file",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "missing.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "missing.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "XR file not found: not found",
		},
		// Execution Failure Tests
		{
			name: "render fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane render
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return []byte("render fail"), fmt.Errorf("fail")
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Render failure: Error is nil, failure shown in section
		},
		{
			name: "validate fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane validate
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Validate failure: Error is nil, failure shown in section
		},
		{
			name: "validate fails but post-test hooks run",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'cleanup executed'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane validate
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}
					// Mock successful hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup executed"), nil
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Validate failure: Error is nil, failure shown in section
		},
		{
			name: "both validate and post-test hook fail",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "failing-cleanup", Run: "echo 'cleanup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}
					// Mock failing hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup\nhook failed"), errors.New("exit status 1")
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Both validate and post-test fail: Error is nil, failures in sections
		},
		{
			name: "post-test hook template failure shows root cause",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{
							Name: "templated-cleanup",
							Run:  fmt.Sprintf("echo 'value: %s.Outputs.UnknownField%s'", placeholder.Open, placeholder.Close),
						},
					},
				},
			},
			setup: func(r *Runner) {
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Pipeline failure only: failure is shown in post-test hooks section
		},
		// Happy Path Tests
		{
			name: "happy path with xr field",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return success
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "happy path with claim field",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock convertClaimToXRFunc to return a fake XR file path
				r.convertClaimToXRFunc = func(_ *Runner, _, _, _, outputPath string) (string, error) {
					return filepath.Join(outputPath, "xr.yaml"), nil
				}
				// Mock the runCommand function to return success for all operations
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		// Connection Secret
		{
			name: "xr with pre-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "setup", Run: "echo 'pre-test setup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("pre-test setup"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with post-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'post-test cleanup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("post-test cleanup"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with both pre-test and post-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "setup", Run: "echo 'pre-test setup'"},
					},
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'post-test cleanup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("hook output"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with failing pre-test hook",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "failing-setup", Run: "echo 'failing setup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results for crossplane commands
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock failing hook execution (use real *exec.ExitError so output shows [x])
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						err := exec.Command("sh", "-c", "exit 1").Run()
						return []byte("failing setup\nhook failed"), err
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Pre-test failure: Error is nil, failure shown in section
		},
		{
			name: "xr with failing post-test hook",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "failing-cleanup", Run: "echo 'failing cleanup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results for crossplane commands
				r.runCommand = func(name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock failing hook execution (use real *exec.ExitError so output shows [x])
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						err := exec.Command("sh", "-c", "exit 1").Run()
						return []byte("failing cleanup\nhook failed"), err
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Post-test failure: Error is nil, failure shown in section
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new test runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem
			testRunner.testSuite = testsuite.New(testRunner.testSuite.File, &api.TestSuiteSpec{})

			// Apply any custom mocks from the test case setup
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			result := testRunner.runTestCase(tc.testCase, 1, testSuiteResult)

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail(), result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else if result.Status != engine.StatusFail() || result.Error != nil {
				assert.Equal(t, engine.StatusPass(), result.Status)
				require.NoError(t, result.Error)
			}
			// else: pipeline failure only (StatusFail, Error == nil); failure is shown in sections

			// For tests with post-test hooks, verify they ran
			if tc.name == "validate fails but post-test hooks run" {
				assert.Len(t, result.PostTestHooksResults, 1)
				assert.Equal(t, "cleanup", result.PostTestHooksResults[0].Name)
				require.NoError(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")
			}

			// Pipeline-only failures: verify failure is in formatted sections
			if tc.name == "render fails" {
				assert.Contains(t, result.FormattedRenderOutput, "render fail")
			}

			if tc.name == "validate fails" {
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")
			}

			if tc.name == "xr with failing pre-test hook" {
				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "[x] failing-setup")
			}

			if tc.name == "xr with failing post-test hook" {
				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "[x] failing-cleanup")
			}

			if tc.name == "both validate and post-test hook fail" {
				assert.Len(t, result.PostTestHooksResults, 1)
				require.Error(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")

				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "Post-test Hooks")
			}

			if tc.name == "post-test hook template failure shows root cause" {
				assert.Len(t, result.PostTestHooksResults, 1)
				require.Error(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.PostTestHooksResults[0].Error.Error(), "failed to render hook template")
				assert.Contains(t, result.PostTestHooksResults[0].Error.Error(), "can't evaluate field UnknownField")

				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "Post-test Hooks:")
				assert.Contains(t, buf.String(), "[!] templated-cleanup")
				assert.Contains(t, buf.String(), "error: failed to render hook template")
				assert.Contains(t, buf.String(), "can't evaluate field UnknownField")
				assert.NotContains(t, buf.String(), "exit code")
			}
		})
	}
}

// The common inputs of the testsuite file are merged into the test case before it runs: the inputs it does not set
// come from them, and the ones it sets are kept. How each input is merged is tested in api.
func TestRunTestCase_MergesTheCommonInputs(t *testing.T) {
	common := api.Common{
		Inputs: api.Inputs{
			Composition:         "common_comp.yaml",
			Functions:           "common_functions.yaml",
			ContextFiles:        map[string]string{"ctx": "common_context.yaml"},
			ObservedResources:   "common_observed.yaml",
			ExtraResources:      "common_extra.yaml",
			FunctionCredentials: "common_credentials.yaml",
		},
		Patches: api.Patches{XRD: "common_xrd.yaml"},
	}

	// What render gets after the functions, from the common inputs
	wantCommon := []string{
		"common_xrd.yaml",
		"--context-files", "common_context.yaml",
		"--observed-resources", "common_observed.yaml",
		"--extra-resources", "common_extra.yaml",
		"--function-credentials", "common_credentials.yaml",
	}

	for name, tt := range map[string]struct {
		testCase api.TestCase
		want     []string // the XR, the composition and the functions that render gets
	}{
		"the inputs that the test case does not set come from the common inputs": {
			testCase: api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml"}},
			want:     []string{"xr.yaml", "common_comp.yaml", "common_functions.yaml"},
		},
		"the inputs that the test case sets are kept": {
			testCase: api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "own_comp.yaml"}},
			want:     []string{"xr.yaml", "own_comp.yaml", "common_functions.yaml"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			runner := newMockRunner(&testexecutionOptions.Options{
				Dependencies: map[string]string{"crossplane": config.CrossplaneCmd},
				Render:       []string{config.RenderSubcommand, config.RenderFlags},
				XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
			})
			runner.fs = afero.NewMemMapFs()
			runner.testSuite = testsuite.New(testSuiteFile, &api.TestSuiteSpec{Common: common})

			var renderArgs []string

			runner.runCommand = func(_ string, args ...string) ([]byte, error) {
				if len(args) > 0 && args[0] == config.RenderSubcommand {
					renderArgs = args

					return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), nil
				}

				return []byte{}, nil
			}

			result := runner.runTestCase(tt.testCase, 1, engine.NewTestSuiteResult("test-suite.yaml", false))

			require.Equal(t, engine.StatusPass(), result.Status)
			require.GreaterOrEqual(t, len(renderArgs), 3)

			baseNames := make([]string, 0, len(renderArgs))
			for _, arg := range renderArgs {
				baseNames = append(baseNames, filepath.Base(arg))
			}

			// After the render command and its flags, and with only the file names of the copies in the temporary directory
			assert.Equal(t, append(tt.want, wantCommon...), baseNames[2:])
		})
	}
}

// The internal artifacts directory, where the test cases of a run keep their outputs for the ones that need them, is
// created by setupTestSuite and removed by completeTestSuite.
func TestSetupTestSuite_CreatesTheInternalArtifactsDirectory(t *testing.T) {
	r := NewRunner(&testexecutionOptions.Options{}, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "test1"}}}))
	r.fs = afero.NewMemMapFs()
	r.output = io.Discard

	assert.Nil(t, r.env, "there is no env before the run is set up")

	require.NoError(t, r.setupTestSuite())

	exists, err := afero.DirExists(r.fs, r.env.savedOutputsDir)
	require.NoError(t, err)
	assert.True(t, exists, "setupTestSuite should create the directory")

	dir := r.env.savedOutputsDir

	require.NoError(t, r.completeTestSuite())

	exists, err = afero.DirExists(r.fs, dir)
	require.NoError(t, err)
	assert.False(t, exists, "completeTestSuite should remove the directory")
}

func TestRunTestCase_UsesTestResultWithStartTime(t *testing.T) {
	options := &testexecutionOptions.Options{
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}
	runner := newMockRunner(options)
	runner.testSuite = testsuite.New(runner.testSuite.File, &api.TestSuiteSpec{})

	// Mock runCommand to return successful results
	runner.runCommand = func(name string, args ...string) ([]byte, error) {
		if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
			return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), nil
		}

		if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
			return []byte("validate ok"), nil
		}

		return []byte{}, nil
	}

	// Provide a minimal valid testCase
	testCase := api.TestCase{
		Name: "test",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
			Functions:   "functions.yaml",
		},
	}
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	result := runner.runTestCase(testCase, 1, testSuiteResult)
	assert.Equal(t, engine.StatusPass(), result.Status)
	require.NoError(t, result.Error)
	assert.False(t, result.StartTime.IsZero(), "TestCaseResult should have StartTime set")
}

func TestRunTestsFileResults(t *testing.T) {
	options := &testexecutionOptions.Options{
		ShowRender:   false,
		ShowValidate: false,
		Debug:        false,
	}

	cases := []struct {
		name       string
		verbose    bool
		withError  bool
		wantStatus string
	}{
		{
			name:       "passing result",
			verbose:    false,
			withError:  false,
			wantStatus: "ok",
		},
		{
			name:       "passing result, verbose",
			verbose:    true,
			withError:  false,
			wantStatus: "PASS",
		},
		{
			name:       "failing result",
			verbose:    false,
			withError:  true,
			wantStatus: "FAIL",
		},
		{
			name:       "failing result, verbose",
			verbose:    true,
			withError:  true,
			wantStatus: "FAIL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			runner := NewRunner(options, testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}}))
			runner.output = &buf
			runner.Verbose = tc.verbose

			// Create a test suite result to test file-level reporting
			testSuiteFile := "test-suite.yaml"
			testSuiteResult := engine.NewTestSuiteResult(testSuiteFile, tc.verbose)

			// Create a test case result
			testCaseResult := engine.NewTestCaseResult("test1", "", tc.verbose, false, false, false, false)

			if tc.withError {
				testCaseResult.Fail(errors.New("test failure"))
			} else {
				testCaseResult.Complete()
			}

			// Add the test case result to the suite
			testSuiteResult.AddResult(testCaseResult)
			testSuiteResult.Complete()

			// Print the test suite result (file-level reporting)
			testSuiteResult.Print(&buf)

			// Verify results
			output := buf.String()

			// Check if the expected status is in the output
			assert.Contains(t, output, tc.wantStatus, "output should contain expected status")

			// Check that the file path appears in the output
			assert.Contains(t, output, testSuiteFile, "Output should contain the test file path")
		})
	}
}

func TestRunTests_Needs(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a"},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "d", Needs: []string{"a", "b"}},
		{Name: "E", ID: "e"},
	}}

	t.Run("nothing fails: everything runs", func(t *testing.T) {
		out, ran := runWithFakeResults(t, spec, 0)

		assert.Equal(t, []string{"a", "b", "c", "d", "e"}, ran)
		assert.NotContains(t, out, "SKIP")
		assert.NotContains(t, out, "FAIL")
	})

	t.Run("the dependents of a failed testcase are skipped, with the reason on follow-up lines, one per need", func(t *testing.T) {
		out, ran := runWithFakeResults(t, spec, 0, "a")

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

func TestRunTests_Order(t *testing.T) {
	// "a" is listed first, but needs "c" and "b", which are listed after it
	tests := []api.TestCase{
		{Name: "A", ID: "a", Needs: []string{"c", "b"}},
		{Name: "B", ID: "b"},
		{Name: "C", ID: "c"},
		{Name: "D", ID: "d"},
	}

	t.Run("the test cases run after the ones they need, the rest in the order of the file", func(t *testing.T) {
		_, ran := runWithFakeResults(t, &api.TestSuiteSpec{Tests: tests}, 0)

		assert.Equal(t, []string{"b", "c", "a", "d"}, ran)
	})

	t.Run("a test case that needs one listed after it is skipped when that one fails", func(t *testing.T) {
		out, ran := runWithFakeResults(t, &api.TestSuiteSpec{Tests: tests}, 0, "b")

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

			serial, serialRan := runWithFakeResults(t, spec, 0, tt.failing...)
			require.NotEmpty(t, serial)

			for _, workers := range []int{1, 2, 8, 16} {
				got, ran := runWithFakeResults(t, spec, workers, tt.failing...)

				assert.Equal(t, timings.ReplaceAllString(serial, "N"), timings.ReplaceAllString(got, "N"), "workers: %d", workers)
				assert.ElementsMatch(t, serialRan, ran, "workers: %d", workers)
			}
		})
	}
}
