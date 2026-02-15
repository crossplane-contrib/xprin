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
	"errors"
	"fmt"
	"os/exec"
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
		assert.Equal(t, StatusPass(), result.Status)
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
		assert.Equal(t, StatusFail(), result.Status)
		assert.Equal(t, err, result.Error)
		assert.Positive(t, result.Duration) // Should be completed
	})
}

func TestTestCaseResult_Skip(t *testing.T) {
	t.Run("sets status to SKIP", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		result.Skip()

		assert.Equal(t, StatusSkip(), result.Status)
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
	t.Run("sets hasFailedRender and returns fail result with no Error", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.RawRenderOutput = []byte("render error output")

		returned := result.FailRender()

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.True(t, result.HasFailedRender)
		assert.Equal(t, StatusFail(), result.Status)
		require.NoError(t, result.Error) // Failure is shown only in render section
		assert.Contains(t, result.FormattedRenderOutput, "[!]", "render failure lines should be prefixed with [!] per crossplane beta validate semantics")
		assert.Contains(t, result.FormattedRenderOutput, "render error output")
	})
}

func TestTestCaseResult_MarkValidateFailed(t *testing.T) {
	t.Run("sets hasFailedValidate and returns formatted error", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		result.RawValidateOutput = []byte("validate error output")

		validateErr := result.MarkValidateFailed()
		returned := result.Fail(validateErr)

		assert.Equal(t, result, returned) // Should return self for chaining
		assert.True(t, result.HasFailedValidate)
		assert.Equal(t, StatusFail(), result.Status)
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
		// Preliminary/infrastructure errors are prefixed with [!] per crossplane beta validate semantics
		assert.Contains(t, buf.String(), "[!]")
	})

	t.Run("prints multiple errors each with [!] prefix", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)
		multiErr := "missing mandatory field: claim or xr\n    missing mandatory field: composition\n    missing mandatory field: functions"
		result.Fail(fmt.Errorf("%s", multiErr))

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "[!] missing mandatory field: claim or xr")
		assert.Contains(t, output, "[!] missing mandatory field: composition")
		assert.Contains(t, output, "[!] missing mandatory field: functions")
	})

	t.Run("prints render output when verbose and show-render", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: test")
		// ProcessRenderOutput sets RenderedResources and FormattedRenderOutput internally
		err := result.ProcessRenderOutput(result.RawRenderOutput)
		require.NoError(t, err)
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Render:")
		assert.Equal(t, 1, strings.Count(output, "Render:"), "Render: should appear exactly once")
	})

	t.Run("prints validate output when verbose and show-validate", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, true, false, false)
		result.RawValidateOutput = []byte("[✓] test validated successfully")
		// ProcessValidateOutput sets FormattedValidateOutput internally
		result.ProcessValidateOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Validate:")
		assert.Equal(t, 1, strings.Count(output, "Validate:"), "Validate: should appear exactly once")
	})

	t.Run("does not print render header when render output is nil", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
		result.RawRenderOutput = nil // Nil output
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Render:")
	})

	t.Run("does not print validate header when validate output is nil", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, true, false, false)
		result.RawValidateOutput = nil // Nil output
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Validate:")
	})

	t.Run("prints pre-test hooks output when verbose and show-hooks", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, true, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("test-hook", "echo 'hello\nworld'", []byte("hello\nworld"), nil),
		}
		result.ProcessPreTestHooksOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Pre-test Hooks:")
		assert.Contains(t, output, "[✓] test-hook")
		assert.Contains(t, output, "hello")
		assert.Contains(t, output, "world")
	})

	t.Run("prints post-test hooks output when verbose and show-hooks", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, true, false)
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye\nuniverse'", []byte("goodbye\nuniverse"), nil),
		}
		result.ProcessPostTestHooksOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Post-test Hooks:")
		assert.Contains(t, output, "[✓] cleanup-hook")
		assert.Contains(t, output, "goodbye")
		assert.Contains(t, output, "universe")
	})

	t.Run("does not print hooks output when show-hooks is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("test-hook", "echo 'hello'", []byte("hello"), nil),
		}
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye'", []byte("goodbye"), nil),
		}
		result.ProcessPreTestHooksOutput()
		result.ProcessPostTestHooksOutput()
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
			NewHookResult("test-hook", "echo 'hello'", []byte("hello"), nil),
		}
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'goodbye'", []byte("goodbye"), nil),
		}
		result.ProcessPreTestHooksOutput()
		result.ProcessPostTestHooksOutput()
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
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass(), "found 3 resources (as expected)"),
			NewAssertionResult("resource-exists", StatusPass(), "resource S3Bucket/my-bucket found (as expected)"),
			NewAssertionResult("field-value", StatusFail(), "expected value 'test', got 'other'"),
		}
		result.ProcessAssertionsOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "Assertions:")
		assert.Contains(t, output, "[✓] count-check - found 3 resources (as expected)")
		assert.Contains(t, output, "[✓] resource-exists - resource S3Bucket/my-bucket found (as expected)")
		assert.Contains(t, output, "[x] field-value - expected value 'test', got 'other'")
		assert.Contains(t, output, "Total: 3 assertions, 2 successful, 1 failed, 0 errors")
	})

	t.Run("does not print assertion results when show-assertions is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass(), "found 3 resources (as expected)"),
		}
		result.ProcessAssertionsOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions:")
		assert.NotContains(t, output, "count-check")
	})

	t.Run("does not print assertion results when verbose is false", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, true)
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass(), "found 3 resources (as expected)"),
		}
		result.ProcessAssertionsOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions:")
		assert.NotContains(t, output, "count-check")
	})

	t.Run("does not print assertion header when assertion results is empty", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = nil
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions:")
	})

	t.Run("does not print assertion header when assertion results is empty slice", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = []AssertionResult{}
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.NotContains(t, output, "Assertions:")
	})
}

func TestTestCaseResult_Print_Integration(t *testing.T) {
	t.Run("successful test with render and validate output", func(t *testing.T) {
		result := NewTestCaseResult("integration-test", "integration-test-id", true, true, true, false, false)
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test")
		// ProcessRenderOutput sets RenderedResources and FormattedRenderOutput internally
		err := result.ProcessRenderOutput(result.RawRenderOutput)
		require.NoError(t, err)

		result.RawValidateOutput = []byte("[✓] test validated successfully")
		// ProcessValidateOutput sets FormattedValidateOutput internally
		result.ProcessValidateOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "=== RUN   integration-test")
		assert.Contains(t, output, "--- PASS: integration-test")
		assert.Contains(t, output, "Render:")
		assert.Contains(t, output, "Validate:")
	})

	t.Run("successful test with all outputs including hooks", func(t *testing.T) {
		result := NewTestCaseResult("full-integration-test", "full-integration-test-id", true, true, true, true, false)
		result.PreTestHooksResults = []HookResult{
			NewHookResult("setup-hook", "echo 'pre-test setup'", []byte("pre-test setup"), nil),
		}
		result.RawRenderOutput = []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: testpod")
		// ProcessRenderOutput sets RenderedResources and FormattedRenderOutput internally
		err := result.ProcessRenderOutput(result.RawRenderOutput)
		require.NoError(t, err)

		result.RawValidateOutput = []byte("[✓] testpod validated successfully")
		// ProcessValidateOutput sets FormattedValidateOutput internally
		result.ProcessValidateOutput()
		result.PostTestHooksResults = []HookResult{
			NewHookResult("cleanup-hook", "echo 'post-test cleanup'", []byte("post-test cleanup"), nil),
		}
		result.ProcessPreTestHooksOutput()
		result.ProcessPostTestHooksOutput()
		result.Complete()

		var buf bytes.Buffer
		result.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "=== RUN   full-integration-test")
		assert.Contains(t, output, "--- PASS: full-integration-test")
		// Check order: pre-test hooks, render, validate, post-test hooks
		preTestIndex := strings.Index(output, "Pre-test Hooks:")
		renderIndex := strings.Index(output, "Render:")
		validateIndex := strings.Index(output, "Validate:")
		postTestIndex := strings.Index(output, "Post-test Hooks:")

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

			// Parse first to set RenderedResources (formatRenderOutput requires it)
			resources, err := result.parseRenderOutput(tc.input)
			if tc.expectErr {
				// For invalid YAML, parsing should fail
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			result.RenderedResources = resources

			output := result.formatRenderOutput()

			if tc.wantOutput {
				assert.True(t, strings.HasPrefix(output, "    Render:"))
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

	t.Run("render failure multiline has [!] only on first line with rest indented inside", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, true, false, false, false)
		result.HasFailedRender = true
		result.RawRenderOutput = []byte(`crossplane: error: composition's compositeTypeRef.kind (XAWSInfrastructure) does not match XR's kind (AWSInfrastructure)
  at: spec.compositeTypeRef
  hint: check composition and XR/claim types`)

		output := result.formatRenderOutput()

		require.NotEmpty(t, output)
		assert.True(t, strings.HasPrefix(output, "    Render:\n"))
		// First line of body: "    [!] errorline1"
		assert.Contains(t, output, "    [!] crossplane: error: composition's compositeTypeRef.kind")
		// Continuation lines indented inside the [!] block (continuationIndent + line, so 14 spaces + "  " + "at:")
		assert.Contains(t, output, "at: spec.compositeTypeRef")
		assert.Contains(t, output, "hint: check composition and XR/claim types")
		assert.Contains(t, output, "              ") // 14 spaces: continuation indent
		assert.NotContains(t, output, "[!] at:")
		assert.NotContains(t, output, "[!] hint:")
		// Count [!] in the body: should appear exactly once (first line only)
		body := output[strings.Index(output, "Render:")+7:]
		assert.Equal(t, 1, strings.Count(body, "[!]"), "render failure body should have exactly one [!] (first line only)")
	})
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
		expectEmpty  bool // true when section is not shown (success and !Verbose && !ShowValidate)
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
			name:         "success no-show-validate",
			input:        validateSuccess,
			verbose:      false,
			showValidate: false,
			expectEmpty:  true, // section not shown when success and !Verbose && !ShowValidate
		},
		{
			name:  "failure show-validate",
			input: validateFailure,
			want: []string{
				"Validate:",
				"[!] could not find CRD/XRD for",
				"myapp validated successfully",
				"Total 2 resources: 1 missing schemas, 1 success cases, 0 failure cases",
				"crossplane: error: cannot validate resources",
			},
			verbose:      true,
			showValidate: true,
		},
		{
			name:  "failure no-show-validate",
			input: validateFailure,
			want: []string{
				"Validate:",
				"[!] could not find CRD/XRD for",
				"Total 2 resources: 1 missing schemas, 1 success cases, 0 failure cases",
				"crossplane: error: cannot validate resources",
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
				result.HasFailedValidate = true
			}

			result.RawValidateOutput = []byte(tc.input)
			output := result.formatValidateOutput()

			if tc.expectEmpty {
				assert.Empty(t, output)
				return
			}

			for _, want := range tc.want {
				assert.Contains(t, output, want)
			}

			if len(tc.notWant) > 0 {
				for _, notWant := range tc.notWant {
					assert.NotContains(t, output, notWant)
				}
			}

			// Single format for both success and failure: leading indent + header + indented body
			assert.True(t, strings.HasPrefix(output, "    Validate:"))

			if !strings.Contains(tc.input, "crossplane: error") {
				assert.Contains(t, output, "\n        [✓] myorg.example.com/v1alpha1, Kind=XApp, myapp validated successfully")
			} else {
				assert.Contains(t, output, "crossplane: error: cannot validate resources")
			}
		})
	}
}

func TestTestCaseResult_formatAssertionsOutput(t *testing.T) {
	t.Run("formats assertion results with header", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true) // Verbose, ShowAssertions => show all
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("count-check", StatusPass(), "found 3 resources (as expected)"),
			NewAssertionResult("resource-exists", StatusPass(), "resource S3Bucket/my-bucket found (as expected)"),
			NewAssertionResult("field-value", StatusFail(), "expected value 'test', got 'other'"),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [✓] count-check - found 3 resources (as expected)\n        [✓] resource-exists - resource S3Bucket/my-bucket found (as expected)\n        [x] field-value - expected value 'test', got 'other'\n        Total: 3 assertions, 2 successful, 1 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles empty results", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = []AssertionResult{}
		formatted := result.formatAssertionsOutput()

		assert.Empty(t, formatted)
	})

	t.Run("handles single assertion", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("single-check", StatusPass(), "all good"),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [✓] single-check - all good\n        Total: 1 assertions, 1 successful, 0 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles multiple assertions with mixed statuses", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("pass-1", StatusPass(), "passed"),
			NewAssertionResult("fail-1", StatusFail(), "failed"),
			NewAssertionResult("pass-2", StatusPass(), "passed again"),
			NewAssertionResult("fail-2", StatusFail(), "failed again"),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [✓] pass-1 - passed\n        [x] fail-1 - failed\n        [✓] pass-2 - passed again\n        [x] fail-2 - failed again\n        Total: 4 assertions, 2 successful, 2 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles assertions with long messages", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		longMessage := "This is a very long assertion message that contains multiple words and describes what went wrong in great detail so that the user can understand the issue"
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("long-message", StatusFail(), longMessage),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [x] long-message - " + longMessage + "\n        Total: 1 assertions, 0 successful, 1 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("multiline message uses title then indented body", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.HasFailedAssertions = true
		multilineMsg := "--- expected.yaml\n+++ actual.yaml\n@@ -1,2 +1,2 @@\n- old\n+ new"
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("diff-check", StatusFail(), multilineMsg),
		}
		formatted := result.formatAssertionsOutput()

		// Multi-line messages: name on first line, then each message line with multilineBodyIndent (12 spaces), same as hooks.
		assert.Contains(t, formatted, "    Assertions:\n        [x] diff-check\n")
		assert.Contains(t, formatted, "            --- expected.yaml\n")
		assert.Contains(t, formatted, "            +++ actual.yaml\n")
		assert.Contains(t, formatted, "            - old\n")
		assert.Contains(t, formatted, "            + new")
		assert.Contains(t, formatted, "Total: 1 assertions, 0 successful, 1 failed, 0 errors\n")
	})

	t.Run("includes Assertions header and 8-space body (same format for error block and verbose)", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("broken", StatusFail(), "something broke"),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [x] broken - something broke\n        Total: 1 assertions, 0 successful, 1 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("showOnlyFailed lists only failed assertions but total uses full counts", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false) // !Verbose && !ShowAssertions + hasFailedAssertions => failed only
		result.HasFailedAssertions = true
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("pass-one", StatusPass(), "ok"),
			NewAssertionResult("fail-one", StatusFail(), "expected 0, got 2"),
		}
		formatted := result.formatAssertionsOutput()

		expected := "    Assertions:\n        [x] fail-one - expected 0, got 2\n        Total: 2 assertions, 1 successful, 1 failed, 0 errors\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("error assertions show [!] and total includes error count", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, true)
		result.HasFailedAssertions = true
		result.AssertionsResults = []AssertionResult{
			NewAssertionResult("pass-one", StatusPass(), "found 2 resources (as expected)"),
			NewAssertionResult("error-one", StatusError(), "resource SomeKind/some-name not found"),
		}
		formatted := result.formatAssertionsOutput()

		assert.Contains(t, formatted, "[!] error-one - resource SomeKind/some-name not found")
		assert.Contains(t, formatted, "Total: 2 assertions, 1 successful, 0 failed, 1 error")
	})
}

func TestTestCaseResult_formatHooksOutput(t *testing.T) {
	t.Run("formats hooks output with label", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'hello\nworld'", []byte("hello\nworld"), nil),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "pre-test", true)

		expected := "    Pre-test Hooks:\n        [✓] test-hook\n            hello\n            world\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles empty output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{}
		formatted := result.formatHooksOutputWithShow(hookResults, "pre-test", true)

		assert.Empty(t, formatted)
	})

	t.Run("handles whitespace-only output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo '   \n  \t  \n  '", []byte("   \n  \t  \n  "), nil),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "post-test", true)

		// body: "   \n  \t  \n  " -> indented with 12 spaces after each newline, then 12 spaces prefix (first line: 12+3)
		expected := "    Post-test Hooks:\n        [✓] test-hook\n               \n              \t  \n              \n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles single line output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'single line'", []byte("single line"), nil),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "pre-test", true)

		expected := "    Pre-test Hooks:\n        [✓] test-hook\n            single line\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("handles multiline output", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("test-hook", "echo 'line 1\nline 2\nline 3'", []byte("line 1\nline 2\nline 3"), nil),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "post-test", true)

		expected := "    Post-test Hooks:\n        [✓] test-hook\n            line 1\n            line 2\n            line 3\n"
		assert.Equal(t, expected, formatted)
	})

	t.Run("showAll false returns only failed hooks", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult("ok-hook", "echo ok", []byte("ok"), nil),
			NewHookResult("fail-hook", "exit 1", []byte("out\nerr"), errors.New("exit status 1")),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "pre-test", false)

		// Plain errors (not *exec.ExitError) get [!] (operational/other).
		assert.Contains(t, formatted, "[!] fail-hook")
		assert.Contains(t, formatted, "error: exit status 1")
		assert.Contains(t, formatted, "out")
		assert.Contains(t, formatted, "err")
		assert.NotContains(t, formatted, "ok-hook")
		assert.NotContains(t, formatted, "exit code")
	})

	t.Run("non-exit failures include root cause details", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)

		hookResults := []HookResult{
			NewHookResult(
				"post-hook-template",
				"echo '{{ .Outputs.UnknownField }}'",
				nil,
				errors.New("failed to render hook template: hook:1:17: executing \"hook\" at <.Outputs.UnknownField>: can't evaluate field UnknownField in type *engine.Outputs"),
			),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "post-test", true)

		assert.Contains(t, formatted, "Post-test Hooks:")
		// Template/rendering failures get [!] (operational/other).
		assert.Contains(t, formatted, "[!] post-hook-template")
		assert.Contains(t, formatted, "error: failed to render hook template")
		assert.Contains(t, formatted, "can't evaluate field UnknownField")
		assert.NotContains(t, formatted, "exit code")
	})

	t.Run("exit code failures show [x]", func(t *testing.T) {
		err := exec.Command("sh", "-c", "exit 2").Run()
		require.Error(t, err)

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "Run returns *exec.ExitError on non-zero exit")

		result := NewTestCaseResult("test", "test-id", true, false, false, false, false)
		hookResults := []HookResult{
			NewHookResult("exit-hook", "exit 2", nil, err),
		}
		formatted := result.formatHooksOutputWithShow(hookResults, "pre-test", true)

		assert.Contains(t, formatted, "[x] exit-hook")
		assert.Contains(t, formatted, "exit code: 2")
	})
}

func TestTestCaseResult_ProcessRenderOutput(t *testing.T) {
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

		err := result.ProcessRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		assert.Len(t, result.RenderedResources, 2)
		assert.Equal(t, "Pod", result.RenderedResources[0].GetKind())
		assert.Equal(t, "test-pod", result.RenderedResources[0].GetName())
		assert.Equal(t, "Service", result.RenderedResources[1].GetKind())
		assert.Equal(t, "test-service", result.RenderedResources[1].GetName())
	})

	t.Run("parses single resource", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		yamlInput := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config`

		err := result.ProcessRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		assert.Len(t, result.RenderedResources, 1)
		assert.Equal(t, "ConfigMap", result.RenderedResources[0].GetKind())
		assert.Equal(t, "test-config", result.RenderedResources[0].GetName())
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		err := result.ProcessRenderOutput([]byte(""))

		require.NoError(t, err)
		assert.Empty(t, result.RenderedResources)
	})

	t.Run("handles invalid YAML", func(t *testing.T) {
		result := NewTestCaseResult("test", "test-id", false, false, false, false, false)

		err := result.ProcessRenderOutput([]byte("invalid: [yaml: "))

		require.Error(t, err)
		assert.Nil(t, result.RenderedResources)
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

		err := result.ProcessRenderOutput([]byte(yamlInput))

		require.NoError(t, err)
		// Empty documents are parsed as empty objects, so we get 3 items: Pod, empty, Service
		assert.Len(t, result.RenderedResources, 3)
		assert.Equal(t, "Pod", result.RenderedResources[0].GetKind())
		assert.Equal(t, "Service", result.RenderedResources[2].GetKind())
	})
}
