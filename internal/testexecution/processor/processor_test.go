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

package processor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	testexecutionRunner "github.com/crossplane-contrib/xprin/internal/testexecution/runner"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

const (
	testSuiteYAML = "/suite.yaml"
	waitFor       = 5 * time.Second
)

// emptySession is a sessionInterface for testing a file without test cases: what finishFunc returns is what the scheduler
// gets from the end of the file, which is where the test cases of the real runner report their failures.
// A file with test cases is a timelineSession, further down.
type emptySession struct {
	finishFunc func() error
	setupErr   error
	file       string
	output     io.Writer
	options    *testexecutionOptions.Options
}

func (m *emptySession) GetTestSuiteFilePath() string { return m.file }

func (m *emptySession) Setup() error { return m.setupErr }

func (*emptySession) Graph() *graph.Graph { return graph.Build(&api.TestSuiteSpec{}) }
func (*emptySession) Run(int)             {}
func (*emptySession) Report(int)          {}

func (m *emptySession) Finish() error {
	if m.finishFunc != nil {
		return m.finishFunc()
	}

	return nil
}

// timeline records what the runners do, in the order it happens.
type timeline struct {
	mu     sync.Mutex
	events []string
}

func (tl *timeline) addf(format string, args ...any) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.events = append(tl.events, fmt.Sprintf(format, args...))
}

func (tl *timeline) all() []string {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	return slices.Clone(tl.events)
}

// with returns the events that start with prefix, in order.
func (tl *timeline) with(prefix string) []string {
	return slices.DeleteFunc(tl.all(), func(e string) bool { return !strings.HasPrefix(e, prefix) })
}

func (tl *timeline) indexOf(t *testing.T, event string) int {
	t.Helper()

	i := slices.Index(tl.all(), event)
	require.GreaterOrEqual(t, i, 0, "%q did not happen: %v", event, tl.all())

	return i
}

// timelineJob is a testsuite file as the scheduler sees it, without any need: what is done to it is recorded.
type timelineJob struct {
	name      string
	n         int
	tl        *timeline
	run       func(i int) // called between the start and the end of a test case
	finishErr error       // what Finish returns
}

// Graph returns the graph of n test cases that don't need each other.
func (s *timelineJob) Graph() *graph.Graph {
	return graph.Build(&api.TestSuiteSpec{Tests: make([]api.TestCase, s.n)})
}

func (s *timelineJob) Run(i int) {
	s.tl.addf("start %s:%d", s.name, i)

	if s.run != nil {
		s.run(i)
	}

	s.tl.addf("end %s:%d", s.name, i)
}

func (s *timelineJob) Report(i int) { s.tl.addf("report %s:%d", s.name, i) }

func (s *timelineJob) Finish() error {
	s.tl.addf("finish %s", s.name)
	fmt.Fprintf(os.Stdout, "ok\t%s\n", s.name) // the summary of the file, as the runner prints it

	return s.finishErr
}

// timelineSession is a session whose test cases are the ones of a timelineJob: what the processor does to it is recorded.
type timelineSession struct {
	*timelineJob

	file     string
	setupErr error
}

func (r *timelineSession) GetTestSuiteFilePath() string { return r.file }

func (r *timelineSession) Setup() error {
	r.tl.addf("setup %s", r.name)

	return r.setupErr
}

// suiteYAML is a testsuite file with n test cases.
func suiteYAML(n int) string {
	var b strings.Builder

	b.WriteString("tests:\n")

	for i := range n {
		fmt.Fprintf(&b, "- name: t%d\n", i)
	}

	return b.String()
}

// setupTimelineSessions writes the testsuite files (the name of each with its content), and makes newSessionFunc return
// timeline runners, which configure can change.
func setupTimelineSessions(t *testing.T, files map[string]string, configure func(r *timelineSession)) (fs afero.Fs, tl *timeline) {
	t.Helper()

	fs = afero.NewMemMapFs()
	tl = &timeline{}

	for name, content := range files {
		require.NoError(t, afero.WriteFile(fs, "/tests/"+name, []byte(content), 0o644))
	}

	original := newSessionFunc

	t.Cleanup(func() { newSessionFunc = original })

	newSessionFunc = func(_ *testexecutionRunner.Runner, testSuite *testsuite.TestSuite) sessionInterface {
		name := filepath.Base(testSuite.File)
		tl.addf("new %s", name)

		r := &timelineSession{file: testSuite.File, timelineJob: &timelineJob{name: name, n: len(testSuite.Spec.Tests), tl: tl}}
		if configure != nil {
			configure(r)
		}

		return r
	}

	return fs, tl
}

// optionsWithWorkers are the options of a run with that number of workers, without the reuse of the containers.
func optionsWithWorkers(workers int) *testexecutionOptions.Options {
	return &testexecutionOptions.Options{Parallel: workers, NoContainerReuse: true}
}

func TestProcessTargets_GivesTheRunnerTheTestSuite(t *testing.T) {
	originalNewSessionFunc := newSessionFunc

	defer func() {
		newSessionFunc = originalNewSessionFunc
	}()

	fs := afero.NewMemMapFs()
	// "one" needs "two", which is listed after it
	require.NoError(t, afero.WriteFile(fs, "/testdir/a_xprin.yaml", []byte("tests:\n- name: one\n  needs:\n  - two\n- name: two\n  id: two\n"), 0o644))

	var given *testsuite.TestSuite

	newSessionFunc = func(_ *testexecutionRunner.Runner, testSuite *testsuite.TestSuite) sessionInterface {
		given = testSuite

		return &emptySession{}
	}

	require.NoError(t, ProcessTargets(fs, []string{"/testdir/a_xprin.yaml"}, &testexecutionOptions.Options{NoContainerReuse: true}))

	require.NotNil(t, given, "the runner gets the test suite that was loaded")
	assert.Equal(t, "/testdir/a_xprin.yaml", given.File)
	assert.Len(t, given.Spec.Tests, 2)
	assert.Equal(t, []int{1, 0}, given.Graph.GetOrder(), "with the graph that the loader built")
}

func TestProcessTargets(t *testing.T) {
	originalNewSessionFunc := newSessionFunc

	defer func() {
		newSessionFunc = originalNewSessionFunc
	}()

	t.Run("non-existent path", func(t *testing.T) {
		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{options: options}
		}

		err := ProcessTargets(afero.NewMemMapFs(), []string{"non-existent-path"}, &testexecutionOptions.Options{Debug: false})

		assert.NoError(t, err, "expected no error for non-existent path")
	})

	t.Run("directory with no test files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/testdir", 0o755))

		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{options: options}
		}

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{"/testdir"}, &testexecutionOptions.Options{})
		})

		require.NoError(t, err, "expected no error for directory path")
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})

	t.Run("recursive path with ...", func(t *testing.T) {
		// Create directory structure
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/base/subdir", 0o755))

		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{options: options}
		}

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{"/base..."}, &testexecutionOptions.Options{})
		})

		// No errors expected
		require.NoError(t, err)
		// Should contain output for both directories
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})

	t.Run("recursive path with error", func(t *testing.T) {
		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{options: options}
		}

		// Invalid glob pattern will cause recursiveDirs to fail
		var err error

		_ = unittestsUtils.CaptureOutput(func() {
			err = ProcessTargets(afero.NewMemMapFs(), []string{"[]..."}, &testexecutionOptions.Options{})
		})

		// Should have an error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "processing completed with errors")
	})

	t.Run("invalid YAML test file", func(t *testing.T) {
		// Create a test file with invalid YAML
		fs := afero.NewMemMapFs()
		testFile := "/test_xprin.yaml"
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("invalid: yaml: : content}"), 0o644))

		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{options: options}
		}

		var err error

		output := unittestsUtils.CaptureOutput(func() {
			err = ProcessTargets(fs, []string{testFile}, &testexecutionOptions.Options{})
		})

		// Should have an error from loading the invalid YAML
		require.Error(t, err)
		assert.Contains(t, err.Error(), "processing completed with errors")
		assert.Contains(t, output.Stderr, "failed to parse testsuite file")
		assert.Contains(t, output.Stderr, "FAIL")
		assert.Contains(t, output.Stdout, "FAIL")
	})

	t.Run("valid test file", func(t *testing.T) {
		// Create a valid test file
		fs := afero.NewMemMapFs()
		testFile := "/test_xprin.yaml"

		// Create a valid test file
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(suiteYAML(1)), 0o644))

		// Mock runner with custom runTests implementation
		runner := &emptySession{
			options: &testexecutionOptions.Options{},
			finishFunc: func() error {
				return nil // Success
			},
		}

		newSessionFunc = func(_ *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			return runner
		}

		err := ProcessTargets(fs, []string{testFile}, &testexecutionOptions.Options{})

		// Should be no errors
		assert.NoError(t, err)
	})

	// Test with recursive path that has a trailing slash
	t.Run("recursive path with trailing slash", func(t *testing.T) {
		// Create directory structure
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/base/subdir", 0o755))

		// Create mock runner
		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{
				options: options,
			}
		}

		// Use a path with a trailing slash and ...
		path := "/base/..."

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{path}, &testexecutionOptions.Options{})
		})

		// No errors expected
		require.NoError(t, err)
		// Should contain output for both directories (the root and the subdir)
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})

	// Test with recursive path that doesn't have a trailing slash
	t.Run("recursive path without trailing slash", func(t *testing.T) {
		// Create directory structure
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/base/subdir", 0o755))

		// Create mock runner
		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			options := r.Options

			return &emptySession{
				options: options,
			}
		}

		// Use a path without a trailing slash and ...
		path := "/base..."

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{path}, &testexecutionOptions.Options{})
		})

		// No errors expected
		require.NoError(t, err)
		// Should contain output for both directories (the root and the subdir)
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})
	// Test that files with invalid names are silently ignored with no errors added
	t.Run("invalid test file names are ignored", func(t *testing.T) {
		var buf bytes.Buffer

		// Create a temp directory with valid and invalid files
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/testdir", 0o755))
		// Invalid files - should not be processed
		invalidFile1 := "/testdir/invalid.yaml"
		invalidFile2 := "/testdir/test.yaml"
		invalidFile3 := "/testdir/xprin.yaml1"          // Extra suffix makes it invalid
		invalidFile4 := "/testdir/whatever_xprin.yaml1" // Extra suffix makes it invalid

		// Valid files - should be processed
		validFile1 := "/testdir/test_xprin.yaml"
		validFile2 := "/testdir/xprin.yaml"
		validFile3 := "/testdir/test_not_xprin.yaml" // Valid per documentation

		// Create the files with content
		require.NoError(t, afero.WriteFile(fs, invalidFile1, []byte("invalid content"), 0o644))
		require.NoError(t, afero.WriteFile(fs, invalidFile2, []byte("also invalid content"), 0o644))
		require.NoError(t, afero.WriteFile(fs, invalidFile3, []byte("invalid with extra suffix"), 0o644))
		require.NoError(t, afero.WriteFile(fs, invalidFile4, []byte("invalid with extra suffix"), 0o644))
		require.NoError(t, afero.WriteFile(fs, validFile1, []byte("valid: content"), 0o644))
		require.NoError(t, afero.WriteFile(fs, validFile2, []byte("valid: content"), 0o644))
		require.NoError(t, afero.WriteFile(fs, validFile3, []byte("valid: content"), 0o644))

		// Create valid test files
		for _, file := range []string{validFile1, validFile2, validFile3} {
			require.NoError(t, afero.WriteFile(fs, file, []byte(suiteYAML(1)), 0o644))
		}

		// Replace the runner factory
		origNewSessionFunc := newSessionFunc

		defer func() { newSessionFunc = origNewSessionFunc }()

		newSessionFunc = func(_ *testexecutionRunner.Runner, testSuite *testsuite.TestSuite) sessionInterface {
			// Create runner with output capture, capturing the test suite in closure
			runner := &emptySession{
				output:  &buf,
				options: &testexecutionOptions.Options{},
				finishFunc: func() error {
					// Simulate what the real runTests would do by printing file results
					// Write directly to the buffer we already have
					fmt.Fprintf(&buf, "ok  \t%s\t%.3fs\n", testSuite.File, 0.123)
					return nil
				},
			}

			return runner
		}

		// Process individual files first to confirm direct file handling works as expected
		err := ProcessTargets(fs, []string{invalidFile1, invalidFile2, invalidFile3, invalidFile4, validFile1, validFile2, validFile3}, &testexecutionOptions.Options{})

		// Should not error since invalid files are just ignored
		require.NoError(t, err)
		// Note: We can't easily track which files were processed without loadFunc,
		// but the test verifies that invalid files don't cause errors
		assert.Contains(t, buf.String(), "test_xprin.yaml", "Output should include the valid files")

		// Reset for directory test
		buf.Reset()

		// Process the directory containing both invalid and valid files
		err = ProcessTargets(fs, []string{"/testdir"}, &testexecutionOptions.Options{})

		// After directory processing, should not error
		require.NoError(t, err, "Processing directory with mixed valid/invalid files should not error")
		assert.Contains(t, buf.String(), "test_xprin.yaml", "Output should include the valid file names")
	})
}

func TestCreateSession(t *testing.T) {
	t.Run("no test cases found error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		// Create a file with no tests
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("common:\n  inputs:\n    composition: comp.yaml\n"), 0o644))

		var err error

		out := unittestsUtils.CaptureStderr(func() {
			_, err = createSession(fs, testFile, testexecutionRunner.NewRunner(&testexecutionOptions.Options{}))
		})
		if !strings.Contains(out, "?   \t/suite.yaml\t[no test cases found]") {
			t.Errorf("expected no test cases found output, got: %q", out)
		}

		if err != nil {
			t.Errorf("did not expect error, got: %v", err)
		}
	})

	t.Run("quiet suppresses [no test cases found]", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("common:\n  inputs:\n    composition: comp.yaml\n"), 0o644))

		var err error

		out := unittestsUtils.CaptureStderr(func() {
			_, err = createSession(fs, testFile, testexecutionRunner.NewRunner(&testexecutionOptions.Options{Quiet: true}))
		})

		require.NoError(t, err)
		assert.NotContains(t, out, "[no test cases found]", "should not emit [no test cases found] when quiet is set")
	})

	t.Run("invalid config error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		// Create a file with invalid YAML that will cause a parse error
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("invalid: yaml: : content}"), 0o644))

		var err error

		runner := &emptySession{output: bytes.NewBuffer(nil), options: &testexecutionOptions.Options{}}
		newSessionFunc = func(_ *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			_, err = createSession(fs, testFile, testexecutionRunner.NewRunner(&testexecutionOptions.Options{}))
		})

		// Should have an error returned
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse testsuite file")

		// Check stderr for error details
		if !strings.Contains(stderrOutput, "# /suite.yaml\nfailed to parse testsuite file") {
			t.Errorf("expected error message in stderr, got: %q", stderrOutput)
		}

		// Check stderr for FAIL status
		if !strings.Contains(stderrOutput, "FAIL\t/suite.yaml\t[invalid testsuite file]") {
			t.Errorf("expected FAIL line in stderr, got: %q", stderrOutput)
		}
	})

	t.Run("no top-level group names", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		// Create a file with no tests
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("common:\n  inputs:\n    composition: comp.yaml\n"), 0o644))

		var err error

		runner := &emptySession{output: bytes.NewBuffer(nil), options: &testexecutionOptions.Options{}}
		newSessionFunc = func(_ *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			_, err = createSession(fs, testFile, testexecutionRunner.NewRunner(&testexecutionOptions.Options{}))
		})

		if !strings.Contains(stderrOutput, "?   \t/suite.yaml\t[no test cases found]") {
			t.Errorf("expected no test cases found output, got: %q", stderrOutput)
		}

		if err != nil {
			t.Errorf("did not expect error, got: %v", err)
		}
	})

	// Table-driven tests for name validation scenarios
	t.Run("testsuite file validation", func(t *testing.T) {
		tests := []struct {
			name           string
			testSuiteSpec  *api.TestSuiteSpec
			expectedErrors []string
			description    string
		}{
			{
				name: "invalid test case ID with special chars",
				testSuiteSpec: &api.TestSuiteSpec{
					Tests: []api.TestCase{
						{
							Name: "validtest",
							ID:   "test@with#special$chars%and&more",
							Inputs: api.Inputs{
								XR:          "xr.yaml",
								Composition: "comp.yaml",
								Functions:   "functions.yaml",
							},
						},
					},
				},
				expectedErrors: []string{
					"test case ID 'test@with#special$chars%and&more' contains invalid characters (allowed: alphanumeric, underscore, hyphen)",
				},
				description: "test case IDs cannot contain special characters",
			},
			{
				name: "duplicate test case IDs",
				testSuiteSpec: &api.TestSuiteSpec{
					Tests: []api.TestCase{
						{
							Name: "Test 1",
							ID:   "test1",
							Inputs: api.Inputs{
								XR:          "xr.yaml",
								Composition: "comp.yaml",
								Functions:   "functions.yaml",
							},
						},
						{
							Name: "Test 2",
							ID:   "test1", // duplicate ID
							Inputs: api.Inputs{
								XR:          "xr2.yaml",
								Composition: "comp.yaml",
								Functions:   "functions.yaml",
							},
						},
					},
				},
				expectedErrors: []string{
					"duplicate test case ID 'test1' found",
				},
				description: "duplicate test case IDs should be detected",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fs := afero.NewMemMapFs()
				testFile := testSuiteYAML

				// Convert test spec to YAML
				testContent, err := yaml.Marshal(tt.testSuiteSpec)
				require.NoError(t, err, "Failed to marshal test spec to YAML")
				require.NoError(t, afero.WriteFile(fs, testFile, testContent, 0o644))

				var processErr error

				runner := &emptySession{output: bytes.NewBuffer(nil), options: &testexecutionOptions.Options{}}
				runner.finishFunc = func() error {
					return nil
				}
				newSessionFunc = func(_ *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
					return runner
				}

				stderrOutput := unittestsUtils.CaptureStderr(func() {
					_, processErr = createSession(fs, testFile, testexecutionRunner.NewRunner(&testexecutionOptions.Options{}))
				})

				if len(tt.expectedErrors) > 0 {
					// Should have an error
					require.Error(t, processErr)
					assert.Contains(t, processErr.Error(), "invalid testsuite file")

					// Check that stderr contains the FAIL status
					assert.Contains(t, stderrOutput, "FAIL\t/suite.yaml\t[invalid testsuite file]")

					// Check that stderr contains the general error message
					assert.Contains(t, stderrOutput, "invalid testsuite file")

					// Check that stderr contains all expected specific error messages
					for _, expectedErr := range tt.expectedErrors {
						assert.Contains(t, stderrOutput, expectedErr, "Should contain error: %s", expectedErr)
					}
				} else {
					// Should not have an error for valid test cases
					require.NoError(t, processErr)

					if strings.Contains(stderrOutput, "FAIL") {
						t.Errorf("did not expect FAIL output, got: %q", stderrOutput)
					}
				}
			})
		}
	})
}

func TestRunError(t *testing.T) {
	t.Run("nothing failed", func(t *testing.T) {
		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = runError(testSuiteYAML, nil)
		})

		require.NoError(t, err)
		assert.Empty(t, stderrOutput)
	})

	t.Run("test cases failed: they were reported as they ran", func(t *testing.T) {
		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = runError(testSuiteYAML, fmt.Errorf("%w in testsuite suite.yaml", testexecutionRunner.ErrTestsFailed))
		})

		require.ErrorIs(t, err, testexecutionRunner.ErrTestsFailed)
		assert.Contains(t, err.Error(), "test execution failed for /suite.yaml")
		assert.Empty(t, stderrOutput, "there is nothing more to say about them")
	})

	t.Run("another error is reported with the file", func(t *testing.T) {
		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = runError(testSuiteYAML, errors.New("some other error"))
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "some other error")
		assert.Contains(t, stderrOutput, "FAIL\t/suite.yaml\t[testsuite file execution error]")
		assert.Contains(t, stderrOutput, "some other error")
	})
}

func TestProcessTargets_ContainerCleanup(t *testing.T) {
	origNewSessionFunc := newSessionFunc
	origNewCoordinatorFunc := newCoordinatorFunc

	defer func() {
		newSessionFunc = origNewSessionFunc
		newCoordinatorFunc = origNewCoordinatorFunc
	}()

	t.Run("initializes Containers and cleans up every tracked name", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := "/suite_xprin.yaml"
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(suiteYAML(1)), 0o644))

		var options *testexecutionOptions.Options

		const functionsPath = "/suite/functions.yaml"
		require.NoError(t, afero.WriteFile(fs, functionsPath, []byte(`apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-a
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-a:v1.0.0
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-b
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-b:v1.0.0
`), 0o644))

		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			opts := r.Options

			options = opts

			return &emptySession{
				finishFunc: func() error {
					// Simulate a Runner applying container reuse for this suite's functions.
					require.NotNil(t, opts.Containers, "ProcessTargets must initialize Containers before running any suite")

					release, err := opts.Containers.AnnotateAndAcquireFunctions(fs, functionsPath)
					if err != nil {
						return err
					}

					release(true)

					return nil
				},
			}
		}

		mockDocker := &unittestsUtils.MockDocker{}

		newCoordinatorFunc = func() (*containers.Coordinator, error) {
			return containers.NewCoordinator(mockDocker), nil
		}

		opts := &testexecutionOptions.Options{}
		require.Nil(t, opts.Containers, "precondition: Containers starts nil")

		err := ProcessTargets(fs, []string{testFile}, opts)
		require.NoError(t, err)

		assert.Same(t, opts.Containers, options.Containers, "the same coordinator instance must be shared with the runner")

		wantNames := []string{
			options.Containers.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-a:v1.0.0"),
			options.Containers.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-b:v1.0.0"),
		}
		assert.ElementsMatch(t, wantNames, mockDocker.RemovedContainers)
	})

	t.Run("NoContainerReuse disables the registry entirely", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := "/suite_xprin.yaml"
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(suiteYAML(1)), 0o644))

		var containersSeenByRunner *containers.Coordinator

		newSessionFunc = func(r *testexecutionRunner.Runner, _ *testsuite.TestSuite) sessionInterface {
			opts := r.Options

			return &emptySession{
				finishFunc: func() error {
					containersSeenByRunner = opts.Containers
					return nil
				},
			}
		}

		newCoordinatorCalled := false

		newCoordinatorFunc = func() (*containers.Coordinator, error) {
			newCoordinatorCalled = true

			return nil, nil
		}

		opts := &testexecutionOptions.Options{NoContainerReuse: true}

		err := ProcessTargets(fs, []string{testFile}, opts)
		require.NoError(t, err)

		assert.Nil(t, opts.Containers, "Containers must stay nil when NoContainerReuse is set")
		assert.Nil(t, containersSeenByRunner, "the runner must also observe a nil registry")
		assert.False(t, newCoordinatorCalled, "no coordinator should even be created when reuse is disabled")
	})

	t.Run("coordinator creation failure is returned", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := "/suite_xprin.yaml"
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(suiteYAML(1)), 0o644))

		newCoordinatorFunc = func() (*containers.Coordinator, error) {
			return nil, errors.New("no docker")
		}

		err := ProcessTargets(fs, []string{testFile}, &testexecutionOptions.Options{})
		require.ErrorContains(t, err, "failed to set up container reuse: no docker")
	})
}

func TestProcessTargets_LoadsTheFilesBeforeRunningAnything(t *testing.T) {
	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": suiteYAML(2), "b_xprin.yaml": suiteYAML(2)}, nil)

	require.NoError(t, ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2)))

	assert.Less(t, tl.indexOf(t, "new b_xprin.yaml"), tl.indexOf(t, "start a_xprin.yaml:0"), "every file is loaded before a test case starts")
	assert.Equal(t, []string{"setup a_xprin.yaml", "setup b_xprin.yaml"}, tl.with("setup "))
	assert.Equal(t, []string{
		"report a_xprin.yaml:0", "report a_xprin.yaml:1", "finish a_xprin.yaml",
		"report b_xprin.yaml:0", "report b_xprin.yaml:1", "finish b_xprin.yaml",
	}, slices.DeleteFunc(tl.all(), func(e string) bool { return !strings.HasPrefix(e, "report ") && !strings.HasPrefix(e, "finish ") }))
}

func TestProcessTargets_TheTestCasesOfTheNextFileUseTheFreeWorkers(t *testing.T) {
	var tl *timeline

	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": suiteYAML(1), "b_xprin.yaml": suiteYAML(3)}, func(r *timelineSession) {
		if r.name == "a_xprin.yaml" {
			// The only test case of the first file does not end until the three of the second file did: until
			// the timeline has their end, which is recorded after their run returns.
			r.run = func(int) {
				deadline := time.Now().Add(waitFor)
				for len(tl.with("end b_xprin.yaml:")) < 3 {
					if time.Now().After(deadline) {
						t.Error("the test cases of the second file did not run while the first one was busy")

						return
					}

					time.Sleep(time.Millisecond)
				}
			}
		}
	})

	require.NoError(t, ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2)))

	// They ran in the other order, and are reported in the order of the files
	assert.Less(t, tl.indexOf(t, "end b_xprin.yaml:2"), tl.indexOf(t, "end a_xprin.yaml:0"))
	assert.Less(t, tl.indexOf(t, "report a_xprin.yaml:0"), tl.indexOf(t, "report b_xprin.yaml:0"))
}

func TestProcessTargets_FailIsPrintedOnceAtTheEnd(t *testing.T) {
	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": suiteYAML(1), "b_xprin.yaml": suiteYAML(1)}, func(r *timelineSession) {
		if r.name == "a_xprin.yaml" {
			r.finishErr = fmt.Errorf("%w in testsuite a_xprin.yaml", testexecutionRunner.ErrTestsFailed)
		}
	})

	var err error

	out := unittestsUtils.CaptureOutput(func() {
		err = ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2))
	})

	require.ErrorIs(t, err, discovery.ErrFailed)
	assert.Equal(t, 1, strings.Count(out.Stdout, "FAIL"), "once: %q", out.Stdout)
	assert.NotContains(t, out.Stderr, "testsuite file execution error", "the test cases that failed were reported as they ran, nothing more")
	assert.Len(t, tl.with("finish "), 2, "the other file was run to the end")
}

func TestProcessTargets_AFileThatCanNotBeLoadedIsLeftOutAndFailsTheRun(t *testing.T) {
	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": "tests: [", "b_xprin.yaml": suiteYAML(1)}, nil)

	var err error

	out := unittestsUtils.CaptureOutput(func() {
		err = ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2))
	})

	require.ErrorIs(t, err, discovery.ErrFailed)
	// FAIL is for the run, once, and after the summary of the file that did run: not when the file was found
	assert.Equal(t, "ok\tb_xprin.yaml\nFAIL\n", out.Stdout)
	assert.Empty(t, tl.with("new a_xprin.yaml"), "no runner for it")
	assert.Equal(t, []string{"report b_xprin.yaml:0"}, tl.with("report "), "the others run")
}

func TestProcessTargets_AFileThatCanNotBeSetUpIsLeftOutAndFailsTheRun(t *testing.T) {
	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": suiteYAML(1), "b_xprin.yaml": suiteYAML(1)}, func(r *timelineSession) {
		if r.name == "a_xprin.yaml" {
			r.setupErr = errors.New("cannot make the directory")
		}
	})

	var err error

	out := unittestsUtils.CaptureOutput(func() {
		err = ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2))
	})

	require.ErrorIs(t, err, discovery.ErrFailed)
	assert.Contains(t, out.Stderr, "cannot make the directory", "it is reported")
	assert.Contains(t, out.Stderr, "[testsuite file execution error]")
	assert.Empty(t, tl.with("start a_xprin.yaml"))
	assert.Equal(t, []string{"report b_xprin.yaml:0"}, tl.with("report "), "the others run")
}

func TestProcessTargets_AFileWithoutTestCasesIsNotRun(t *testing.T) {
	fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": "tests: []\n", "b_xprin.yaml": suiteYAML(1)}, nil)

	var err error

	out := unittestsUtils.CaptureOutput(func() {
		err = ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(2))
	})

	require.NoError(t, err)
	assert.Contains(t, out.Stderr, "[no test cases found]")
	assert.Empty(t, tl.with("new a_xprin.yaml"))
}

func TestProcessTargets_OneOrNoneRunsTheTestCasesOneAfterTheOther(t *testing.T) {
	for _, workers := range []int{0, 1} {
		fs, tl := setupTimelineSessions(t, map[string]string{"a_xprin.yaml": suiteYAML(2), "b_xprin.yaml": suiteYAML(1)}, nil)

		require.NoError(t, ProcessTargets(fs, []string{"/tests/..."}, optionsWithWorkers(workers)), "workers: %d", workers)

		assert.Less(t, tl.indexOf(t, "end a_xprin.yaml:0"), tl.indexOf(t, "start a_xprin.yaml:1"), "workers: %d", workers)
		assert.Less(t, tl.indexOf(t, "end a_xprin.yaml:1"), tl.indexOf(t, "start b_xprin.yaml:0"), "workers: %d", workers)
	}
}
