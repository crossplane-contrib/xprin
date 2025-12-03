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

package engine

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestNewTestCaseResult(t *testing.T) {
	t.Run("creates result with correct initial values", func(t *testing.T) {
		result := NewTestCaseResult("test-name", "test-id", true, false, true, true, false)

		assert.Equal(t, "test-name", result.Name)
		assert.Equal(t, "test-id", result.ID)
		assert.Equal(t, StatusPass, result.Status)
		assert.True(t, result.Verbose)
		assert.False(t, result.ShowRender)
		assert.True(t, result.ShowValidate)
		assert.True(t, result.ShowHooks)
		assert.False(t, result.StartTime.IsZero())
		require.NoError(t, result.Error)
		assert.Equal(t, time.Duration(0), result.Duration)
		assert.NotNil(t, result.Outputs.Rendered)
		assert.Empty(t, result.Outputs.Render)
		assert.Empty(t, result.Outputs.XR)
		assert.Nil(t, result.Outputs.Validate)
		assert.Equal(t, 0, result.Outputs.RenderCount)
	})
}

func TestTestCaseResult_Fail(t *testing.T) {
	t.Run("sets error and status to FAIL", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		err := assert.AnError

		returned := result.Fail(err)

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.Equal(t, StatusFail, result.Status)
		assert.Equal(t, err, result.Error)
		assert.Positive(t, result.Duration) // Should be completed
	})
}

func TestTestCaseResult_Skip(t *testing.T) {
	t.Run("sets status to SKIP", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		result.Skip()

		assert.Equal(t, StatusSkip, result.Status)
	})
}

func TestTestCaseResult_Complete(t *testing.T) {
	t.Run("sets duration and returns self", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		// Add a small delay to ensure non-zero duration
		time.Sleep(1 * time.Millisecond)

		returned := result.Complete()

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.Positive(t, result.Duration)
	})
}

func TestTestCaseResult_FailRender(t *testing.T) {
	t.Run("sets hasFailedRender and returns fail result", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.RawRenderOutput = []byte("render error output")

		returned := result.FailRender()

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.True(t, result.hasFailedRender)
		assert.Equal(t, StatusFail, result.Status)
		assert.Contains(t, result.Error.Error(), "render error output")
	})
}

func TestTestCaseResult_MarkValidateFailed(t *testing.T) {
	t.Run("sets hasFailedValidate and returns formatted error", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.RawValidateOutput = []byte("validate error output")

		validateErr := result.MarkValidateFailed()
		returned := result.Fail(validateErr)

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.True(t, result.hasFailedValidate)
		assert.Equal(t, StatusFail, result.Status)
		assert.Contains(t, result.Error.Error(), "validate error output")
	})
}

func TestTestCaseResult_Print(t *testing.T) {
	t.Run("prints nothing for passing test in non-verbose mode", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		assert.Empty(t, buf.String())
	})

	t.Run("prints RUN message for verbose mode", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		assert.Contains(t, buf.String(), "=== RUN   test")
		assert.Contains(t, buf.String(), "--- PASS: test")
	})

	t.Run("prints error for failed test", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.Fail(assert.AnError)

		var buf bytes.Buffer
		result.Print(&buf)

		assert.Contains(t, buf.String(), "--- FAIL: test")
		assert.Contains(t, buf.String(), assert.AnError.Error())
	})

	t.Run("prints render output when verbose and show-render", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: test")
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Rendered resources:")
		assert.Equal(t, 1, strings.Count(output, "Rendered resources:"), "Rendered resources: should appear exactly once")
	})

	t.Run("prints validate output when verbose and show-validate", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, true, false, false)
		result.RawValidateOutput = []byte("[✓] test validated successfully")
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Validation results:")
		assert.Equal(t, 1, strings.Count(output, "Validation results:"), "Validation results: should appear exactly once")
	})

	t.Run("does not print render header when render output is nil", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
		result.RawRenderOutput = nil // Nil output
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Rendered resources:")
	})

	t.Run("does not print validate header when validate output is nil", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, true, false, false)
		result.RawValidateOutput = nil // Nil output
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Validation results:")
	})

	t.Run("prints pre-test hooks output when verbose and show-hooks", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, true, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("test-hook", "echo 'hello\nworld'", []byte("hello\nworld"), []byte(""), nil),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "pre-test hooks results:")
		assert.Contains(t, output, "- test-hook")
		assert.Contains(t, output, "hello")
		assert.Contains(t, output, "world")
	})

	t.Run("prints post-test hooks output when verbose and show-hooks", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, true, false)
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye\nuniverse'", []byte("goodbye\nuniverse"), []byte(""), nil),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "post-test hooks results:")
		assert.Contains(t, output, "- cleanup-hook")
		assert.Contains(t, output, "goodbye")
		assert.Contains(t, output, "universe")
	})

	t.Run("does not print hooks output when show-hooks is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("test-hook", "echo 'hello'", []byte("hello"), []byte(""), nil),
		}
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye'", []byte("goodbye"), []byte(""), nil),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "pre-test hooks output:")
		assert.NotContains(t, output, "post-test hooks output:")
		assert.NotContains(t, output, "hello")
		assert.NotContains(t, output, "goodbye")
	})

	t.Run("does not print hooks output when verbose is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, true, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("test-hook", "echo 'hello'", []byte("hello"), []byte(""), nil),
		}
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye'", []byte("goodbye"), []byte(""), nil),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "pre-test hooks output:")
		assert.NotContains(t, output, "post-test hooks output:")
	})

	t.Run("does not print hooks header when hooks output is nil", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, true, false)
		result.PreTestHooksResults = nil
		result.PostTestHooksResults = nil
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "pre-test hooks output:")
		assert.NotContains(t, output, "post-test hooks output:")
	})

	t.Run("prints assertion results when verbose and show-assertions", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsAllResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass, "found 3 resources (as expected)"),
			NewAssertionResult("resource-exists", StatusPass, "resource S3Bucket/my-bucket found (as expected)"),
			NewAssertionResult("field-value", StatusFail, "expected value 'test', got 'other'"),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Assertions results:")
		assert.Contains(t, output, "PASS: count-check - found 3 resources (as expected)")
		assert.Contains(t, output, "PASS: resource-exists - resource S3Bucket/my-bucket found (as expected)")
		assert.Contains(t, output, "FAIL: field-value - expected value 'test', got 'other'")
	})

	t.Run("does not print assertion results when show-assertions is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		result.AssertionsAllResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass, "found 3 resources (as expected)"),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions results:")
		assert.NotContains(t, output, "count-check")
	})

	t.Run("does not print assertion results when verbose is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, true)
		result.AssertionsAllResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass, "found 3 resources (as expected)"),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions results:")
		assert.NotContains(t, output, "count-check")
	})

	t.Run("does not print assertion header when assertion results is empty", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsAllResults = nil
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions results:")
	})

	t.Run("does not print assertion header when assertion results is empty slice", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsAllResults = []AssertionResult{}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions results:")
	})
}

func TestTestCaseResult_Print_Integration(t *testing.T) {
	t.Run("successful test with render and validate output", func(t *testing.T) {
		result := NewTestCaseResult("integration-test", "integration-test-id", true, true, true, false, false)
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test")
		result.RawValidateOutput = []byte("[✓] test validated successfully")
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "=== RUN   integration-test")
		assert.Contains(t, output, "--- PASS: integration-test")
		assert.Contains(t, output, "Rendered resources:")
		assert.Contains(t, output, "Validation results:")
	})

	t.Run("successful test with all outputs including hooks", func(t *testing.T) {
		result := NewTestCaseResult("full-integration-test", "full-integration-test-id", true, true, true, true, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("setup-hook", "echo 'pre-test setup'", []byte("pre-test setup"), []byte(""), nil),
		}
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: testpod")
		result.RawValidateOutput = []byte("[✓] testpod validated successfully")
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'post-test cleanup'", []byte("post-test cleanup"), []byte(""), nil),
		}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "=== RUN   full-integration-test")
		assert.Contains(t, output, "--- PASS: full-integration-test")
		// Check order: pre-test hooks, render, validate, post-test hooks
		preTestIndex := strings.Index(output, "pre-test hooks results:")
		renderIndex := strings.Index(output, "Rendered resources:")
		validateIndex := strings.Index(output, "Validation results:")
		postTestIndex := strings.Index(output, "post-test hooks results:")

		assert.Less(t, preTestIndex, renderIndex, "Pre-test hooks should appear before render")
		assert.Less(t, renderIndex, validateIndex, "Render should appear before validate")
		assert.Less(t, validateIndex, postTestIndex, "Validate should appear before post-test hooks")

		assert.Contains(t, output, "pre-test setup")
		assert.Contains(t, output, "testpod")
		assert.Contains(t, output, "post-test cleanup")
	})

	t.Run("failed test with render and validate output", func(t *testing.T) {
		result := NewTestCaseResult("failed-test", "failed-test-id", true, true, true, false, false)
		result.RawRenderOutput = []byte("render error")
		result.RawValidateOutput = []byte("validate error")
		result.Fail(result.MarkValidateFailed())

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "=== RUN   failed-test")
		assert.Contains(t, output, "--- FAIL: failed-test")
		assert.Contains(t, output, "validate error")
		// Should not show formatted outputs for failed tests since error message contains them
	})
}

func TestFormatRenderOutput(t *testing.T) {
	testCases := []struct {
		name       string
		input      []byte
		expectErr  bool
		wantOutput bool
		wantKind   string
		wantName   string
	}{
		{
			name:       "summary output",
			input:      []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: testpod"),
			expectErr:  false,
			wantOutput: true,
			wantKind:   "Pod",
			wantName:   "testpod",
		},
		{
			name:       "invalid yaml",
			input:      []byte("invalid: [yaml: "),
			expectErr:  true,
			wantOutput: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a TestCaseResult with the necessary flags
			result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
			output := result.formatRenderOutput(tc.input)

			if tc.expectErr {
				// For invalid YAML, the method should return an error message
				assert.Contains(t, output, "Error parsing YAML")
				return
			}

			if tc.wantOutput {
				assert.True(t, strings.HasPrefix(output, "    Rendered resources:"))
				assert.Contains(t, output, "\n        └── Pod/testpod")

				if tc.wantKind != "" {
					assert.Contains(t, output, tc.wantKind)
				}

				if tc.wantName != "" {
					assert.Contains(t, output, tc.wantName)
				}

				assert.Contains(t, output, "/") // Kind/Name format
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestFormatValidateOutput(t *testing.T) {
	validateSuccess := `[✓] myorg.example.com/v1alpha1, Kind=XApp, myapp validated successfully
Total 1 resources: 0 missing schemas, 1 success cases, 0 failure cases
`
	validateFailure := `[✓] myorg.example.com/v1alpha1, Kind=XApp, myapp validated successfully
[!] could not find CRD/XRD for: kubernetes.crossplane.io/v1alpha2, Kind=Object
Total 2 resources: 1 missing schemas, 1 success cases, 0 failure cases
crossplane: error: cannot validate resources: could not validate all resources, schema(s) missing
`
	cases := []struct {
		name         string
		input        string
		want         []string
		notWant      []string
		verbose      bool
		showValidate bool
	}{
		{
			name:  "success show-validate",
			input: validateSuccess,
			want: []string{
				"myapp validated successfully",
				"0 missing schemas, 1 success cases, 0 failure cases",
			},
			verbose:      true,
			showValidate: true,
		},
		{
			name:  "success no-show-validate",
			input: validateSuccess,
			want: []string{
				"myapp validated successfully",
				"0 missing schemas, 1 success cases, 0 failure cases",
			},
			verbose:      true,
			showValidate: false,
		},
		{
			name:  "failure show-validate",
			input: validateFailure,
			want: []string{
				"crossplane: error: cannot validate resources",
				"myapp validated successfully",
				"\n    [!] could not find CRD/XRD for",
			},
			verbose:      true,
			showValidate: true,
		},
		{
			name:  "failure no-show-validate",
			input: validateFailure,
			want: []string{
				"crossplane: error: cannot validate resources",
				"\n    [!] could not find CRD/XRD for",
			},
			notWant:      []string{"myapp validated successfully"},
			verbose:      false,
			showValidate: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a TestCaseResult with the necessary flags
			result := NewTestCaseResult("test", "test-id", tc.verbose, false, tc.showValidate, false, false)

			// Set hasFailedValidate for failure test cases
			if strings.Contains(tc.name, "failure") {
				result.hasFailedValidate = true
			}

			output := result.formatValidateOutput([]byte(tc.input))

			for _, want := range tc.want {
				assert.Contains(t, output, want)
			}

			if len(tc.notWant) > 0 {
				for _, notWant := range tc.notWant {
					assert.NotContains(t, output, notWant)
				}
			}

			if !tc.verbose && strings.HasPrefix(output, "crossplane:") {
				assert.Contains(t, output, "    ") // Check for extra spaces
			}
			// For success cases, should have proper indentation
			if !strings.HasPrefix(output, "crossplane:") {
				assert.True(t, strings.HasPrefix(output, "    Validation results:"))
				assert.Contains(t, output, "\n        [✓] myorg.example.com/v1alpha1, Kind=XApp, myapp validated successfully")
			}
			// Should always move crossplane: error from last line to first line
			if strings.Contains(output, "crossplane:") {
				// Ensure it starts with "crossplane:"
				assert.True(t, strings.HasPrefix(output, "crossplane:"))
			}
		})
	}
}

func TestTestCaseResult_formatAssertionsOutput(t *testing.T) {
	t.Run("formats assertion results with header", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		assertionResults := []AssertionResult{
			NewAssertionResult("count-check", StatusPass, "found 3 resources (as expected)"),
			NewAssertionResult("resource-exists", StatusPass, "resource S3Bucket/my-bucket found (as expected)"),
			NewAssertionResult("field-value", StatusFail, "expected value 'test', got 'other'"),
		}
		formatted := result.formatAssertionsOutput(assertionResults, false)

		expected := "    Assertions results:\n        PASS: count-check - found 3 resources (as expected)\n        PASS: resource-exists - resource S3Bucket/my-bucket found (as expected)\n        FAIL: field-value - expected value 'test', got 'other'"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles empty results", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		assertionResults := []AssertionResult{}
		formatted := result.formatAssertionsOutput(assertionResults, false)

		expected := ""
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles single assertion", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		assertionResults := []AssertionResult{
			NewAssertionResult("single-check", StatusPass, "all good"),
		}
		formatted := result.formatAssertionsOutput(assertionResults, false)

		expected := "    Assertions results:\n        PASS: single-check - all good"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles multiple assertions with mixed statuses", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		assertionResults := []AssertionResult{
			NewAssertionResult("pass-1", StatusPass, "passed"),
			NewAssertionResult("fail-1", StatusFail, "failed"),
			NewAssertionResult("pass-2", StatusPass, "passed again"),
			NewAssertionResult("fail-2", StatusFail, "failed again"),
		}
		formatted := result.formatAssertionsOutput(assertionResults, false)

		expected := "    Assertions results:\n        PASS: pass-1 - passed\n        FAIL: fail-1 - failed\n        PASS: pass-2 - passed again\n        FAIL: fail-2 - failed again"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles assertions with long messages", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		longMessage := "This is a very long assertion message that contains multiple words and describes what went wrong in great detail so that the user can understand the issue"
		assertionResults := []AssertionResult{
			NewAssertionResult("long-message", StatusFail, longMessage),
		}
		formatted := result.formatAssertionsOutput(assertionResults, false)

		expected := "    Assertions results:\n        FAIL: long-message - " + longMessage
		assert.Equal(t, expected, formatted)
	})
}

func TestTestCaseResult_formatHooksOutput(t *testing.T) {
	t.Run("formats hooks output with label", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'hello\nworld'", []byte("hello\nworld"), []byte(""), nil),
		}
		formatted := result.formatHooksOutput(hookResults, "pre-test")

		expected := "    pre-test hooks results:\n        - test-hook\n            hello\n            world"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles empty output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{}
		formatted := result.formatHooksOutput(hookResults, "pre-test")

		assert.Empty(t, formatted)
	})

	t.Run("handles whitespace-only output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo '   \n  \t  \n  '", []byte("   \n  \t  \n  "), []byte(""), nil),
		}
		formatted := result.formatHooksOutput(hookResults, "post-test")

		// Should show hook header but no output content since it's whitespace-only
		expected := "    post-test hooks results:\n        - test-hook"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles single line output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'single line'", []byte("single line"), []byte(""), nil),
		}
		formatted := result.formatHooksOutput(hookResults, "pre-test")

		expected := "    pre-test hooks results:\n        - test-hook\n            single line"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles multiline output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'line 1\nline 2\nline 3'", []byte("line 1\nline 2\nline 3"), []byte(""), nil),
		}
		formatted := result.formatHooksOutput(hookResults, "post-test")

		expected := "    post-test hooks results:\n        - test-hook\n            line 1\n            line 2\n            line 3"
		assert.Equal(t, expected, formatted)
	})
}

func TestTestCaseResult_ParseRenderOutput(t *testing.T) {
	t.Run("parses valid YAML with multiple resources", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		yamlInput := `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
---
apiVersion: v1
kind: Service
metadata:
  name: test-service`

		resources, err := result.ParseRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "Pod", resources[0].GetKind())
		assert.Equal(t, "test-pod", resources[0].GetName())
		assert.Equal(t, "Service", resources[1].GetKind())
		assert.Equal(t, "test-service", resources[1].GetName())
	})

	t.Run("parses single resource", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		yamlInput := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config`

		resources, err := result.ParseRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, "ConfigMap", resources[0].GetKind())
		assert.Equal(t, "test-config", resources[0].GetName())
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		resources, err := result.ParseRenderOutput([]byte(""))

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		resources, err := result.ParseRenderOutput([]byte("invalid: [yaml: "))

		require.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("handles YAML with comments and empty documents", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		yamlInput := `# This is a comment
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
---
# Another comment
---
apiVersion: v1
kind: Service
metadata:
  name: test-service`

		resources, err := result.ParseRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		// Empty documents are parsed as empty objects, so we get 3 items: Pod, empty, Service
		assert.Len(t, resources, 3)
		assert.Equal(t, "Pod", resources[0].GetKind())
		assert.Equal(t, "Service", resources[2].GetKind())
	})
}
