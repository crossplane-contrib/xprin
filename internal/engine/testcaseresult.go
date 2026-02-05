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
	"io"
	"strings"
	"time"

	"github.com/gertd/go-pluralize"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// Global indentation constant for consistent formatting.
const spaces = "    "

// TestCaseResult represents the result of a single test case.
type TestCaseResult struct {
	Name      string
	ID        string // Test case ID for cross-test references
	Duration  time.Duration
	Error     error
	Status    Status // StatusPass, StatusFail, or StatusSkip
	StartTime time.Time

	// Raw outputs (stored by runner)
	RawRenderOutput   []byte
	RawValidateOutput []byte

	// Parsed render resources (parsed once, used many times)
	RenderedResources []*unstructured.Unstructured

	// Formatted outputs (formatted once, displayed many times)
	FormattedRenderOutput           string
	FormattedValidateOutput         string
	FormattedPreTestHooksOutput     string
	FormattedPostTestHooksOutput    string
	FormattedAssertionsOutput       string // All assertions (for verbose display)
	FormattedAssertionsFailedOutput string // Failed assertions only (for error messages)

	PreTestHooksResults  []HookResult
	PostTestHooksResults []HookResult

	AssertionsAllResults    []AssertionResult
	AssertionsFailedResults []AssertionResult

	// Outputs for template variables in hooks
	Outputs Outputs

	hasFailedRender     bool
	hasFailedValidate   bool
	hasFailedAssertions bool

	// Formatting flags (passed from runner)
	Verbose        bool
	ShowRender     bool
	ShowValidate   bool
	ShowHooks      bool
	ShowAssertions bool
}

// NewTestCaseResult creates a new test case result.
func NewTestCaseResult(name, id string, verbose, showRender, showValidate, showHooks, showAssertions bool) *TestCaseResult {
	return &TestCaseResult{
		Name:           name,
		ID:             id,
		Status:         StatusPass, // Default to pass
		StartTime:      time.Now(),
		Verbose:        verbose,
		ShowRender:     showRender,
		ShowValidate:   showValidate,
		ShowHooks:      showHooks,
		ShowAssertions: showAssertions,
		Outputs: Outputs{
			Rendered: make(map[string]string),
		},
	}
}

// Outputs represents the output data available in post-test hooks.
type Outputs struct {
	Render      string            // Path to rendered.yaml
	XR          string            // Path to xr.yaml
	Validate    *string           // Path to validate.yaml (nil if no CRDs)
	RenderCount int               // Number of resources in render output
	Rendered    map[string]string // Kind/Name -> file path for individual rendered resources
}

// Fail marks a test case as failed with the given error and completes it, returning the result for chaining.
func (tcr *TestCaseResult) Fail(err error) *TestCaseResult {
	tcr.Error = err
	tcr.Status = StatusFail

	return tcr.Complete()
}

// Skip marks a test case as skipped.
func (tcr *TestCaseResult) Skip() {
	tcr.Status = StatusSkip
}

// Complete finalizes a test case result with duration and returns the result for chaining.
func (tcr *TestCaseResult) Complete() *TestCaseResult {
	tcr.Duration = time.Since(tcr.StartTime)
	return tcr
}

// FailRender handles render failure with proper formatting.
func (tcr *TestCaseResult) FailRender() *TestCaseResult {
	tcr.hasFailedRender = true
	tcr.FormattedRenderOutput = strings.TrimSpace(string(tcr.RawRenderOutput))

	return tcr.Fail(fmt.Errorf("%s", tcr.FormattedRenderOutput))
}

// MarkValidateFailed marks the test as having failed validation and returns the formatted error.
// Callers should then call Fail() with this error to mark the test as failed.
func (tcr *TestCaseResult) MarkValidateFailed() error {
	tcr.hasFailedValidate = true
	tcr.FormattedValidateOutput = tcr.formatValidateOutput(tcr.RawValidateOutput)

	return fmt.Errorf("%s", tcr.FormattedValidateOutput)
}

// MarkAssertionsFailed marks the test as having failed assertions and returns the formatted error.
// Callers should then call Fail() with this error to mark the test as failed.
func (tcr *TestCaseResult) MarkAssertionsFailed() error {
	tcr.hasFailedAssertions = true
	return fmt.Errorf("%s", tcr.FormattedAssertionsFailedOutput)
}

// Print prints the test case result to the given writer.
func (tcr *TestCaseResult) Print(w io.Writer) {
	// In non-verbose mode, only print failures
	if tcr.Status == StatusPass && !tcr.Verbose {
		return
	}

	// Print RUN message for this test (like go test)
	if tcr.Verbose {
		fmt.Fprintf(w, "=== RUN   %s\n", tcr.Name) //nolint:errcheck // output function, error handling not practical
	}

	// Print status line
	fmt.Fprintf(w, "--- %s: %s (%.2fs)\n", tcr.Status, tcr.Name, tcr.Duration.Seconds()) //nolint:errcheck // output function, error handling not practical

	if tcr.FormattedPreTestHooksOutput != "" && tcr.Verbose && tcr.ShowHooks {
		fmt.Fprintf(w, "%s\n", tcr.FormattedPreTestHooksOutput) //nolint:errcheck // output function, error handling not practical
	}

	if tcr.FormattedRenderOutput != "" && !tcr.hasFailedRender && tcr.Verbose && tcr.ShowRender {
		fmt.Fprintf(w, "%s\n", tcr.FormattedRenderOutput) //nolint:errcheck // output function, error handling not practical
	}

	if tcr.FormattedValidateOutput != "" && !tcr.hasFailedValidate && tcr.Verbose && tcr.ShowValidate {
		fmt.Fprintf(w, "%s\n", tcr.FormattedValidateOutput) //nolint:errcheck // output function, error handling not practical
	}

	if tcr.FormattedAssertionsOutput != "" && tcr.Verbose && tcr.ShowAssertions {
		fmt.Fprintf(w, "%s\n", tcr.FormattedAssertionsOutput) //nolint:errcheck // output function, error handling not practical
	}

	if tcr.FormattedPostTestHooksOutput != "" && tcr.Verbose && tcr.ShowHooks {
		fmt.Fprintf(w, "%s\n", tcr.FormattedPostTestHooksOutput) //nolint:errcheck // output function, error handling not practical
	}

	// Print error message for failed tests
	if tcr.Status == StatusFail && tcr.Error != nil {
		errorLines := strings.Split(tcr.Error.Error(), "\n")
		// Validate and assertion errors are already formatted with proper indentation (formatValidateOutput / formatAssertionsOutput).
		// Other errors (e.g. from CheckMandatoryFields) need normalization so multiline messages don't get double-indented.
		if tcr.hasFailedValidate || tcr.hasFailedAssertions {
			// Print as-is (already formatted)
			for _, line := range errorLines {
				fmt.Fprintf(w, "    %s\n", line) //nolint:errcheck // output function, error handling not practical
			}
		} else {
			// Normalize: remove leading whitespace from all lines then add consistent 4-space indentation
			for _, line := range errorLines {
				fmt.Fprintf(w, "    %s\n", strings.TrimLeft(line, " \t")) //nolint:errcheck // output function, error handling not practical
			}
		}
	}
}

// formatRenderOutput formats the rendered YAML raw output as a summary.
func (tcr *TestCaseResult) formatRenderOutput() string {
	// Pre-allocate lines slice with capacity for all resources
	lines := make([]string, 1, len(tcr.RenderedResources)+1)
	lines[0] = fmt.Sprintf("%sRendered resources:", spaces)

	// Loop over the resources and extract kind/name
	for _, resource := range tcr.RenderedResources {
		kind := resource.GetKind()
		name := resource.GetName()

		// Add line with same prefix for all
		lines = append(lines, fmt.Sprintf("%s├── %s/%s", spaces+spaces, kind, name))
	}

	// Join lines and fix the last prefix
	lastLineIndex := len(lines) - 1
	lines[lastLineIndex] = strings.Replace(lines[lastLineIndex], "├──", "└──", 1)

	return strings.Join(lines, "\n")
}

// formatValidateOutput formats the validation raw output for display.
func (tcr *TestCaseResult) formatValidateOutput(output []byte) string {
	outputStr := strings.TrimSpace(string(output))

	if tcr.hasFailedValidate {
		// Get the crossplane error line (last line) and the rest
		lines := strings.Split(outputStr, "\n")
		crossplaneError := lines[len(lines)-1]
		rest := lines[:len(lines)-1]

		if tcr.ShowValidate {
			// Show all lines with crossplane error first
			return fmt.Sprintf("%s\n%s%s", crossplaneError, spaces, strings.Join(rest, "\n"+spaces))
		}

		// Filter out "validated successfully" lines and show crossplane error first
		var filtered []string

		for _, line := range rest {
			if !strings.HasSuffix(line, "validated successfully") {
				filtered = append(filtered, line)
			}
		}

		return fmt.Sprintf("%s\n%s%s", crossplaneError, spaces, strings.Join(filtered, "\n"+spaces))
	}

	// For successful validation, just format with proper indentation
	return fmt.Sprintf("%sValidation results:\n%s%s", spaces, spaces+spaces, strings.ReplaceAll(outputStr, "\n", "\n"+spaces+spaces))
}

// formatHooksOutput formats the hooks output for display
// Success: shows both stdout and stderr
// Failure: shows only stdout (stderr goes in error message).
func (tcr *TestCaseResult) formatHooksOutput(hooksResults []HookResult, label string) string {
	if len(hooksResults) == 0 {
		return ""
	}

	var lines []string

	lines = append(lines, fmt.Sprintf("%s%s hooks results:", spaces, label))

	for _, hook := range hooksResults {
		// Format hook header (name only if present, otherwise command)
		if hook.Name != "" {
			lines = append(lines, fmt.Sprintf("%s- %s", spaces+spaces, hook.Name))
		} else {
			lines = append(lines, fmt.Sprintf("%s- %s", spaces+spaces, hook.Command))
		}

		// Add hook output based on success/failure
		// Always show stdout (both success and failure cases)
		if len(hook.Stdout) > 0 {
			stdoutStr := strings.TrimSpace(string(hook.Stdout))
			if stdoutStr != "" {
				outputLines := strings.Split(stdoutStr, "\n")
				for _, line := range outputLines {
					lines = append(lines, fmt.Sprintf("%s%s", spaces+spaces+spaces, line))
				}
			}
		}

		// Only show stderr for successful hooks (failure stderr goes in error message)
		if hook.Error == nil && len(hook.Stderr) > 0 {
			stderrStr := strings.TrimSpace(string(hook.Stderr))
			if stderrStr != "" {
				outputLines := strings.Split(stderrStr, "\n")
				for _, line := range outputLines {
					lines = append(lines, fmt.Sprintf("%s%s", spaces+spaces+spaces, line))
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// formatAssertionsOutput formats assertion results for display.
// When failed is true, formats only failed assertions for error messages (no indentation, as Print() handles it).
// When failed is false, formats all assertions for verbose output (with indentation).
func (tcr *TestCaseResult) formatAssertionsOutput(assertionResults []AssertionResult, failed bool) string {
	if len(assertionResults) == 0 {
		return ""
	}

	// Pre-allocate: 1 header line + len(assertionResults) item lines
	lines := make([]string, 0, 1+len(assertionResults))

	var itemIndent string // Indentation for individual assertion items

	if failed {
		// For error messages: no header indentation (Print() will indent), items get 4 spaces
		plural := pluralize.NewClient()
		noun := plural.Pluralize("assertion", len(assertionResults), true)
		lines = append(lines, fmt.Sprintf("%s failed:", noun))
		itemIndent = spaces
	} else {
		// For verbose output: header gets 4 spaces, items get 8 spaces
		lines = append(lines, fmt.Sprintf("%sAssertions results:", spaces))
		itemIndent = spaces + spaces
	}

	for _, result := range assertionResults {
		lines = append(lines, fmt.Sprintf("%s%s: %s - %s", itemIndent, string(result.Status), result.Name, result.Message))
	}

	return strings.Join(lines, "\n")
}

// parseRenderOutput parses the raw render output and returns the resources.
func (tcr *TestCaseResult) parseRenderOutput(output []byte) ([]*unstructured.Unstructured, error) {
	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(output))

	var resources []*unstructured.Unstructured

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}

		resources = append(resources, obj)
	}

	return resources, nil
}

// ProcessRenderOutput parses the raw render output and formats it.
// It sets both RenderedResources and FormattedRenderOutput.
func (tcr *TestCaseResult) ProcessRenderOutput(output []byte) error {
	// Parse first and store in RenderedResources
	resources, err := tcr.parseRenderOutput(output)
	if err != nil {
		return err
	}

	tcr.RenderedResources = resources

	// Format using the already-parsed resources
	tcr.FormattedRenderOutput = tcr.formatRenderOutput()

	return nil
}

// ProcessValidateOutput formats the validation raw output.
// It sets FormattedValidateOutput.
func (tcr *TestCaseResult) ProcessValidateOutput(output []byte) {
	tcr.FormattedValidateOutput = tcr.formatValidateOutput(output)
}

// ProcessHooksOutput formats the hooks results.
// It sets FormattedPreTestHooksOutput and/or FormattedPostTestHooksOutput.
func (tcr *TestCaseResult) ProcessHooksOutput() {
	if len(tcr.PreTestHooksResults) > 0 {
		tcr.FormattedPreTestHooksOutput = tcr.formatHooksOutput(tcr.PreTestHooksResults, "pre-test")
	}

	if len(tcr.PostTestHooksResults) > 0 {
		tcr.FormattedPostTestHooksOutput = tcr.formatHooksOutput(tcr.PostTestHooksResults, "post-test")
	}
}

// ProcessAssertionsOutput formats the assertion results.
// It sets FormattedAssertionsOutput (all assertions) and FormattedAssertionsFailedOutput (failed only).
func (tcr *TestCaseResult) ProcessAssertionsOutput() {
	if len(tcr.AssertionsAllResults) > 0 {
		tcr.FormattedAssertionsOutput = tcr.formatAssertionsOutput(tcr.AssertionsAllResults, false)
	}

	if len(tcr.AssertionsFailedResults) > 0 {
		tcr.FormattedAssertionsFailedOutput = tcr.formatAssertionsOutput(tcr.AssertionsFailedResults, true)
	}
}
