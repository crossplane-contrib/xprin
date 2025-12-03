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
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestReportError(t *testing.T) {
	tests := []struct {
		name             string
		target           string
		failureReason    string
		originalErr      error
		expectedErrorMsg string
		expectedStderr   []string
	}{
		{
			name:             "basic error reporting",
			target:           "test-target",
			failureReason:    "test failure",
			originalErr:      assert.AnError,
			expectedErrorMsg: "test failure in test-target: assert.AnError general error for testing",
			expectedStderr: []string{
				"# test-target",
				"test failure in test-target: assert.AnError general error for testing",
				"FAIL\ttest-target\t[test failure]",
			},
		},
		{
			name:             "error with special characters",
			target:           "path/with/slashes",
			failureReason:    "parsing failed",
			originalErr:      assert.AnError,
			expectedErrorMsg: "parsing failed in path/with/slashes: assert.AnError general error for testing",
			expectedStderr: []string{
				"# path/with/slashes",
				"parsing failed in path/with/slashes: assert.AnError general error for testing",
				"FAIL\tpath/with/slashes\t[parsing failed]",
			},
		},
		{
			name:             "empty strings",
			target:           "",
			failureReason:    "",
			originalErr:      assert.AnError,
			expectedErrorMsg: " in : assert.AnError general error for testing",
			expectedStderr: []string{
				"# ",
				" in : assert.AnError general error for testing",
				"FAIL\t\t[]",
			},
		},
		{
			name:             "long error messages",
			target:           "very-long-target-name-that-might-cause-formatting-issues",
			failureReason:    "complex failure with multiple reasons",
			originalErr:      assert.AnError,
			expectedErrorMsg: "complex failure with multiple reasons in very-long-target-name-that-might-cause-formatting-issues: assert.AnError general error for testing",
			expectedStderr: []string{
				"# very-long-target-name-that-might-cause-formatting-issues",
				"FAIL\tvery-long-target-name-that-might-cause-formatting-issues\t[complex failure with multiple reasons]",
			},
		},
		{
			name:             "nil error handling",
			target:           "test-target",
			failureReason:    "nil error test",
			originalErr:      nil,
			expectedErrorMsg: "nil error test in test-target: <nil>",
			expectedStderr: []string{
				"# test-target",
				"nil error test in test-target: <nil>",
				"FAIL\ttest-target\t[nil error test]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			stderrOutput := unittestsUtils.CaptureStderr(func() {
				err := reportError(tt.target, tt.failureReason, tt.originalErr)

				// Verify the returned error message
				assert.Equal(t, tt.expectedErrorMsg, err.Error())
			})

			// Verify all expected stderr content
			for _, expectedContent := range tt.expectedStderr {
				assert.Contains(t, stderrOutput, expectedContent)
			}
		})
	}
}

func TestReportTestSuiteError(t *testing.T) {
	tests := []struct {
		name             string
		testSuiteFile    string
		originalErr      error
		failureReason    string
		expectedErrorMsg string
		expectedStderr   []string
	}{
		{
			name:             "basic test suite error reporting",
			testSuiteFile:    "suite.yaml",
			originalErr:      assert.AnError,
			failureReason:    "invalid testsuite file",
			expectedErrorMsg: "# suite.yaml\nassert.AnError general error for testing",
			expectedStderr: []string{
				"# suite.yaml",
				"assert.AnError general error for testing",
				"FAIL\tsuite.yaml\t[invalid testsuite file]",
			},
		},
		{
			name:             "test suite error with complex path",
			testSuiteFile:    "tests/aws/complex_xprin.yaml",
			originalErr:      assert.AnError,
			failureReason:    "invalid testsuite file",
			expectedErrorMsg: "# tests/aws/complex_xprin.yaml\nassert.AnError general error for testing",
			expectedStderr: []string{
				"# tests/aws/complex_xprin.yaml",
				"assert.AnError general error for testing",
				"FAIL\ttests/aws/complex_xprin.yaml\t[invalid testsuite file]",
			},
		},
		{
			name:             "execution error reason",
			testSuiteFile:    "execution_test.yaml",
			originalErr:      assert.AnError,
			failureReason:    "testsuite file execution error",
			expectedErrorMsg: "# execution_test.yaml\nassert.AnError general error for testing",
			expectedStderr: []string{
				"# execution_test.yaml",
				"assert.AnError general error for testing",
				"FAIL\texecution_test.yaml\t[testsuite file execution error]",
			},
		},
		{
			name:             "empty parameters",
			testSuiteFile:    "",
			originalErr:      assert.AnError,
			failureReason:    "",
			expectedErrorMsg: "# \nassert.AnError general error for testing",
			expectedStderr: []string{
				"# ",
				"assert.AnError general error for testing",
				"FAIL\t\t[]",
			},
		},
		{
			name:             "nil error handling",
			testSuiteFile:    "test.yaml",
			originalErr:      nil,
			failureReason:    "nil error test",
			expectedErrorMsg: "# test.yaml\n<nil>",
			expectedStderr: []string{
				"# test.yaml",
				"<nil>",
				"FAIL\ttest.yaml\t[nil error test]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			stderrOutput := unittestsUtils.CaptureStderr(func() {
				err := reportTestSuiteError(tt.testSuiteFile, tt.originalErr, tt.failureReason)

				// Verify the returned error message
				assert.Equal(t, tt.expectedErrorMsg, err.Error())
			})

			// Verify all expected stderr content
			for _, expectedContent := range tt.expectedStderr {
				assert.Contains(t, stderrOutput, expectedContent)
			}
		})
	}
}

func TestHelperFunctionsConsistency(t *testing.T) {
	t.Run("both helpers produce consistent stderr format", func(t *testing.T) {
		// Test that both helper functions produce similar stderr format
		target := "test-file.yaml"
		originalErr := assert.AnError

		// Test reportError stderr output
		reportErrorStderr := unittestsUtils.CaptureStderr(func() {
			_ = reportError(target, "test failure", originalErr)
		})

		// Test reportTestSuiteError stderr output
		reportTestSuiteErrorStderr := unittestsUtils.CaptureStderr(func() {
			_ = reportTestSuiteError(target, originalErr, "test failure")
		})

		// Both should contain the target file name prefixed with #
		assert.Contains(t, reportErrorStderr, "# test-file.yaml")
		assert.Contains(t, reportTestSuiteErrorStderr, "# test-file.yaml")

		// Both should contain FAIL status lines
		assert.Contains(t, reportErrorStderr, "FAIL\ttest-file.yaml\t[test failure]")
		assert.Contains(t, reportTestSuiteErrorStderr, "FAIL\ttest-file.yaml\t[test failure]")

		// Both should contain the error message
		assert.Contains(t, reportErrorStderr, "assert.AnError general error for testing")
		assert.Contains(t, reportTestSuiteErrorStderr, "assert.AnError general error for testing")
	})

	t.Run("error return values are properly formatted", func(t *testing.T) {
		target := "example.yaml"
		originalErr := assert.AnError

		var err1, err2 error

		// Capture output to avoid test noise
		_ = unittestsUtils.CaptureOutput(func() {
			// Test reportError return value
			err1 = reportError(target, "failure type", originalErr)

			// Test reportTestSuiteError return value
			err2 = reportTestSuiteError(target, originalErr, "failure type")
		})

		require.Error(t, err1)
		assert.Equal(t, "failure type in example.yaml: assert.AnError general error for testing", err1.Error())

		require.Error(t, err2)
		assert.Equal(t, "# example.yaml\nassert.AnError general error for testing", err2.Error())

		// The two functions produce different error message formats by design
		assert.NotEqual(t, err1.Error(), err2.Error())
	})

	t.Run("integration test with real use cases", func(t *testing.T) {
		// Test scenarios that mirror actual usage in the codebase
		scenarios := []struct {
			name           string
			target         string
			failureReason  string
			originalErr    error
			expectedStderr []string
		}{
			{
				name:          "YAML parsing error",
				target:        "tests/aws_xprin.yaml",
				failureReason: "invalid testsuite file",
				originalErr:   assert.AnError,
				expectedStderr: []string{
					"# tests/aws_xprin.yaml",
					"FAIL\ttests/aws_xprin.yaml\t[invalid testsuite file]",
					"assert.AnError general error for testing",
				},
			},
			{
				name:          "directory access error",
				target:        "/non/existent/path",
				failureReason: "failed to access test path",
				originalErr:   assert.AnError,
				expectedStderr: []string{
					"# /non/existent/path",
					"FAIL\t/non/existent/path\t[failed to access test path]",
					"assert.AnError general error for testing",
				},
			},
			{
				name:          "test execution failure",
				target:        "suite.yaml",
				failureReason: "testsuite file execution error",
				originalErr:   assert.AnError,
				expectedStderr: []string{
					"# suite.yaml",
					"FAIL\tsuite.yaml\t[testsuite file execution error]",
					"assert.AnError general error for testing",
				},
			},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Test reportError
				stderrOutput := unittestsUtils.CaptureStderr(func() {
					err := reportError(scenario.target, scenario.failureReason, scenario.originalErr)
					require.Error(t, err)
					assert.Contains(t, err.Error(), scenario.failureReason)
					assert.Contains(t, err.Error(), scenario.target)
				})

				for _, expectedContent := range scenario.expectedStderr {
					assert.Contains(t, stderrOutput, expectedContent, "stderr should contain: %s", expectedContent)
				}

				// Test reportTestSuiteError
				stderrOutput2 := unittestsUtils.CaptureStderr(func() {
					err := reportTestSuiteError(scenario.target, scenario.originalErr, scenario.failureReason)
					require.Error(t, err)
					assert.Contains(t, err.Error(), scenario.target)
				})

				// Should contain similar stderr patterns
				assert.Contains(t, stderrOutput2, "# "+scenario.target)
				assert.Contains(t, stderrOutput2, "FAIL\t"+scenario.target+"\t["+scenario.failureReason+"]")
			})
		}
	})
}
