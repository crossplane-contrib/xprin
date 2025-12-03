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
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"gopkg.in/yaml.v3"
)

const (
	testContentWithTests = `tests:
  - name: test1`
	testSuiteYAML = "/suite.yaml"
)

// mockRunner is a mock implementation of runnerInterface for testing.
type mockRunner struct {
	runTestsFunc func(*api.TestSuiteSpec, string) error
	output       io.Writer
	options      *testexecutionUtils.Options
}

func (m *mockRunner) RunTests(testSuiteSpec *api.TestSuiteSpec, testSuiteFile string) error {
	if m.runTestsFunc != nil {
		return m.runTestsFunc(testSuiteSpec, testSuiteFile)
	}

	return nil
}

func TestProcessTargets(t *testing.T) {
	originalNewRunnerFunc := newRunnerFunc

	defer func() {
		newRunnerFunc = originalNewRunnerFunc
	}()

	t.Run("non-existent path", func(t *testing.T) {
		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{options: options}
		}

		err := ProcessTargets(afero.NewMemMapFs(), []string{"non-existent-path"}, &testexecutionUtils.Options{Debug: false})

		assert.NoError(t, err, "expected no error for non-existent path")
	})

	t.Run("directory with no test files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/testdir", 0o755))

		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{options: options}
		}

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{"/testdir"}, &testexecutionUtils.Options{})
		})

		require.NoError(t, err, "expected no error for directory path")
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})

	t.Run("recursive path with ...", func(t *testing.T) {
		// Create directory structure
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/base/subdir", 0o755))

		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{options: options}
		}

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{"/base..."}, &testexecutionUtils.Options{})
		})

		// No errors expected
		require.NoError(t, err)
		// Should contain output for both directories
		assert.Contains(t, stderrOutput, "[no testsuite files]")
	})

	t.Run("recursive path with error", func(t *testing.T) {
		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{options: options}
		}

		// Invalid glob pattern will cause recursiveDirs to fail
		var err error

		_ = unittestsUtils.CaptureOutput(func() {
			err = ProcessTargets(afero.NewMemMapFs(), []string{"[]..."}, &testexecutionUtils.Options{})
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

		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{options: options}
		}

		var err error

		output := unittestsUtils.CaptureOutput(func() {
			err = ProcessTargets(fs, []string{testFile}, &testexecutionUtils.Options{})
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
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(testContentWithTests), 0o644))

		// Mock runner with custom runTests implementation
		runner := &mockRunner{
			options: &testexecutionUtils.Options{},
			runTestsFunc: func(*api.TestSuiteSpec, string) error {
				return nil // Success
			},
		}

		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		err := ProcessTargets(fs, []string{testFile}, &testexecutionUtils.Options{})

		// Should be no errors
		assert.NoError(t, err)
	})

	// Test with recursive path that has a trailing slash
	t.Run("recursive path with trailing slash", func(t *testing.T) {
		// Create directory structure
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/base/subdir", 0o755))

		// Create mock runner
		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{
				options: options,
			}
		}

		// Use a path with a trailing slash and ...
		path := "/base/..."

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{path}, &testexecutionUtils.Options{})
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
		newRunnerFunc = func(options *testexecutionUtils.Options) runnerInterface {
			return &mockRunner{
				options: options,
			}
		}

		// Use a path without a trailing slash and ...
		path := "/base..."

		var err error

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = ProcessTargets(fs, []string{path}, &testexecutionUtils.Options{})
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
			require.NoError(t, afero.WriteFile(fs, file, []byte(testContentWithTests), 0o644))
		}

		// Create runner with output capture
		runner := &mockRunner{
			output:  &buf,
			options: &testexecutionUtils.Options{},
			runTestsFunc: func(_ *api.TestSuiteSpec, filePath string) error {
				// Simulate what the real runTests would do by printing file results
				// Write directly to the buffer we already have
				fmt.Fprintf(&buf, "ok  \t%s\t%.3fs\n", filePath, 0.123)
				return nil
			},
		}

		// Replace the runner factory
		origNewRunnerFunc := newRunnerFunc

		defer func() { newRunnerFunc = origNewRunnerFunc }()

		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		// Process individual files first to confirm direct file handling works as expected
		err := ProcessTargets(fs, []string{invalidFile1, invalidFile2, invalidFile3, invalidFile4, validFile1, validFile2, validFile3}, &testexecutionUtils.Options{})

		// Should not error since invalid files are just ignored
		require.NoError(t, err)
		// Note: We can't easily track which files were processed without loadFunc,
		// but the test verifies that invalid files don't cause errors
		assert.Contains(t, buf.String(), "test_xprin.yaml", "Output should include the valid files")

		// Reset for directory test
		buf.Reset()

		// Process the directory containing both invalid and valid files
		err = ProcessTargets(fs, []string{"/testdir"}, &testexecutionUtils.Options{})

		// After directory processing, should not error
		require.NoError(t, err, "Processing directory with mixed valid/invalid files should not error")
		assert.Contains(t, buf.String(), "test_xprin.yaml", "Output should include the valid file names")
	})
}

func TestProcessDirectory(t *testing.T) {
	// Table-driven test for "no test files" scenarios
	t.Run("no test files scenarios", func(t *testing.T) {
		tests := []struct {
			name        string
			description string
		}{
			{"empty directory", "standard empty directory case"},
			{"empty directory verification", "verify no test files found message works consistently"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fs := afero.NewMemMapFs()
				dir := "/testdir"
				require.NoError(t, fs.MkdirAll(dir, 0o755))

				var err error

				out := unittestsUtils.CaptureStderr(func() {
					err = processDirectory(fs, dir, &testexecutionUtils.Options{})
				})
				assert.Contains(t, out, "?   \t"+dir+"\t[no testsuite files]", "expected no testsuite files message")
				assert.NoError(t, err, "did not expect error for empty directory")
			})
		}
	})

	// Test with a non-existent directory
	t.Run("non-existent directory", func(t *testing.T) {
		// afero.Glob doesn't error on invalid patterns like filepath.Glob does,
		// and processDirectory treats "no test files found" as a special case (no error)
		fs := afero.NewMemMapFs()
		badPattern := "/nonexistent" // Non-existent directory

		var err error

		out := unittestsUtils.CaptureStderr(func() {
			err = processDirectory(fs, badPattern, &testexecutionUtils.Options{})
		})
		// processDirectory treats "no test files found" as a special case and doesn't return an error
		// It just prints a message to stderr
		require.NoError(t, err, "processDirectory doesn't error on non-existent directories")
		assert.Contains(t, out, "?   \t"+badPattern+"\t[no testsuite files]",
			"expected no testsuite files message")
	})

	// Integration: processDirectory prints file names for each file
	t.Run("prints file names for each file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		dir := "/testdir"
		require.NoError(t, fs.MkdirAll(dir, 0o755))

		file1 := "/testdir/a_xprin.yaml"
		file2 := "/testdir/b_xprin.yaml"

		require.NoError(t, afero.WriteFile(fs, file1, []byte("dummy"), 0o644))
		require.NoError(t, afero.WriteFile(fs, file2, []byte("dummy"), 0o644))

		var err error

		out := unittestsUtils.CaptureOutput(func() {
			err = processDirectory(fs, dir, &testexecutionUtils.Options{})
		})
		// Since we're writing dummy files, there will likely be errors during processing
		// but that's not what we're testing here - we're testing file discovery
		_ = err // We don't care about the error in this test

		assert.Contains(t, out.Stderr, "a_xprin.yaml", "expected output to mention file1")
		assert.Contains(t, out.Stderr, "b_xprin.yaml", "expected output to mention file2")
	})
}

func TestProcessTestSuiteFile(t *testing.T) {
	t.Run("no test cases found error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		// Create a file with no tests
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("common:\n  inputs:\n    composition: comp.yaml\n"), 0o644))

		var err error

		out := unittestsUtils.CaptureStderr(func() {
			err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
		})
		if !strings.Contains(out, "?   \t/suite.yaml\t[no test cases found]") {
			t.Errorf("expected no test cases found output, got: %q", out)
		}

		if err != nil {
			t.Errorf("did not expect error, got: %v", err)
		}
	})

	t.Run("invalid config error", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		// Create a file with invalid YAML that will cause a parse error
		require.NoError(t, afero.WriteFile(fs, testFile, []byte("invalid: yaml: : content}"), 0o644))

		var err error

		runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
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

		runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
		})

		if !strings.Contains(stderrOutput, "?   \t/suite.yaml\t[no test cases found]") {
			t.Errorf("expected no test cases found output, got: %q", stderrOutput)
		}

		if err != nil {
			t.Errorf("did not expect error, got: %v", err)
		}
	})

	t.Run("test failure with test case", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(testContentWithTests), 0o644))

		var err error

		runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
		runner.runTestsFunc = func(_ *api.TestSuiteSpec, _ string) error {
			return errors.New("tests failed in testsuite suite.yaml: something failed")
		}
		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		assert.Contains(t, err.Error(), "tests failed in testsuite suite.yaml: something failed")
	})

	t.Run("test failure without group", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(testContentWithTests), 0o644))

		var err error

		runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
		runner.runTestsFunc = func(_ *api.TestSuiteSpec, _ string) error {
			return errors.New("some other error") // Use a generic error, not a group error
		}
		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
		})

		// Check stderr for FAIL status
		if !strings.Contains(stderrOutput, "FAIL\t/suite.yaml\t[testsuite file execution error]") {
			t.Errorf("expected file fail output in stderr, got: %q", stderrOutput)
		}

		// Check stderr for detailed error message
		if !strings.Contains(stderrOutput, "some other error") {
			t.Errorf("expected error output in stderr, got: %q", stderrOutput)
		}

		if err == nil {
			t.Errorf("expected error, got nil")
		}

		assert.Contains(t, err.Error(), "some other error")
	})

	t.Run("success", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		testFile := testSuiteYAML
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(testContentWithTests), 0o644))

		var err error

		runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
		runner.runTestsFunc = func(_ *api.TestSuiteSpec, _ string) error {
			return nil
		}
		newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
			return runner
		}

		stderrOutput := unittestsUtils.CaptureStderr(func() {
			err = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
		})

		if strings.Contains(stderrOutput, "FAIL") {
			t.Errorf("did not expect FAIL output, got: %q", stderrOutput)
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

				runner := &mockRunner{output: bytes.NewBuffer(nil), options: &testexecutionUtils.Options{}}
				runner.runTestsFunc = func(_ *api.TestSuiteSpec, _ string) error {
					return nil
				}
				newRunnerFunc = func(_ *testexecutionUtils.Options) runnerInterface {
					return runner
				}

				stderrOutput := unittestsUtils.CaptureStderr(func() {
					processErr = processTestSuiteFile(fs, testFile, &testexecutionUtils.Options{})
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
