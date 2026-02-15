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
	"os/exec"
	"strings"
	"time"

	"github.com/gertd/go-pluralize"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	spaces = "    " // Global indentation constant for consistent formatting.
	// multilineBodyIndent is the prefix for multiline bodies (assertion messages, hook output). Keep equal so dyff/diff look the same.
	multilineBodyIndent = spaces + spaces + spaces // 12 spaces
)

// indentMultilineBody returns body with each line prefixed by indent (first line gets indent at start,
// subsequent lines get newline+indent). Used for hook output and assertion multi-line messages.
func indentMultilineBody(indent, body string) string {
	trimmed := strings.TrimSuffix(body, "\n")
	if trimmed == "" {
		return ""
	}

	return indent + strings.ReplaceAll(trimmed, "\n", "\n"+indent)
}

// TestCaseResult represents the result of a single test case.
type TestCaseResult struct {
	Name      string
	ID        string // Test case ID for cross-test references
	Duration  time.Duration
	Error     error
	Status    Status
	StartTime time.Time

	// Raw outputs (stored by runner)
	RawRenderOutput   []byte
	RawValidateOutput []byte

	// Parsed render resources (parsed once, used many times)
	RenderedResources []*unstructured.Unstructured

	// Formatted outputs (formatted once, displayed many times)
	FormattedRenderOutput        string
	FormattedValidateOutput      string
	FormattedPreTestHooksOutput  string
	FormattedPostTestHooksOutput string
	FormattedAssertionsOutput    string

	PreTestHooksResults  []HookResult
	PostTestHooksResults []HookResult

	AssertionsResults []AssertionResult

	// Outputs for template variables in hooks
	Outputs Outputs

	HasFailedRender        bool
	HasFailedValidate      bool
	HasFailedAssertions    bool
	HasFailedPreTestHooks  bool
	HasFailedPostTestHooks bool

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
		Status:         StatusPass(), // Default to pass
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
	tcr.Status = StatusFail()

	return tcr.Complete()
}

// Skip marks a test case as skipped.
func (tcr *TestCaseResult) Skip() {
	tcr.Status = StatusSkip()
}

// Complete finalizes a test case result with duration and returns the result for chaining.
func (tcr *TestCaseResult) Complete() *TestCaseResult {
	tcr.Duration = time.Since(tcr.StartTime)
	return tcr
}

// FailRender handles render failure with proper formatting.
// Error is not set; the failure is shown only via the render section.
func (tcr *TestCaseResult) FailRender() *TestCaseResult {
	tcr.HasFailedRender = true
	tcr.FormattedRenderOutput = tcr.formatRenderOutput()

	return tcr.Fail(nil)
}

// HasPipelineFailure returns true if validate, assertions, or post-test hooks failed.
// Used by the runner to call Fail(nil) when no infrastructure error was collected.
func (tcr *TestCaseResult) HasPipelineFailure() bool {
	return tcr.HasFailedValidate || tcr.HasFailedAssertions || tcr.HasFailedPostTestHooks
}

// MarkValidateFailed marks the test as having failed validation and returns the formatted error.
// Callers should then call Fail() with this error to mark the test as failed.
func (tcr *TestCaseResult) MarkValidateFailed() error {
	tcr.HasFailedValidate = true
	tcr.FormattedValidateOutput = tcr.formatValidateOutput()

	return fmt.Errorf("%s", tcr.FormattedValidateOutput)
}

// MarkAssertionsFailed marks the test as having failed assertions and returns the formatted error.
// Callers should then call Fail() with this error to mark the test as failed.
func (tcr *TestCaseResult) MarkAssertionsFailed() error {
	tcr.HasFailedAssertions = true
	return fmt.Errorf("%s", tcr.formatAssertionsOutput())
}

// Print prints the test case result to the given writer.
func (tcr *TestCaseResult) Print(w io.Writer) {
	// In non-verbose mode, only print failures
	if tcr.Status == StatusPass() && !tcr.Verbose {
		return
	}

	// Print RUN message for this test (like go test)
	if tcr.Verbose {
		fmt.Fprintf(w, "=== RUN   %s\n", tcr.Name) //nolint:errcheck // output function, error handling not practical
	}

	// Print status line
	fmt.Fprintf(w, "--- %s: %s (%.2fs)\n", tcr.Status, tcr.Name, tcr.Duration.Seconds()) //nolint:errcheck // output function, error handling not practical

	fmt.Fprint(w, tcr.FormattedPreTestHooksOutput)  //nolint:errcheck // output function, error handling not practical
	fmt.Fprint(w, tcr.FormattedRenderOutput)        //nolint:errcheck // output function, error handling not practical
	fmt.Fprint(w, tcr.FormattedValidateOutput)      //nolint:errcheck // output function, error handling not practical
	fmt.Fprint(w, tcr.FormattedAssertionsOutput)    //nolint:errcheck // output function, error handling not practical
	fmt.Fprint(w, tcr.FormattedPostTestHooksOutput) //nolint:errcheck // output function, error handling not practical

	// Print error when set (only set for failures not represented in a section).
	if tcr.Status == StatusFail() && tcr.Error != nil {
		fmt.Fprint(w, formatErrorBlock(tcr.Error.Error())) //nolint:errcheck // output function, error handling not practical
	}
}

// formatErrorBlock formats an error for the error block: section-aligned indent.
// Used for preliminary/infrastructure errors (e.g. missing mandatory fields, failed to create dirs).
// Prefixes every line with [!] so each error is clearly an operational/other error, per crossplane beta validate semantics.
// This works well for multiple independent errors (e.g. several missing mandatory fields) and for single-line errors.
// If errMsg already starts with the section indent (e.g. pre-formatted hooks/assertions output), it is returned as-is.
func formatErrorBlock(errMsg string) string {
	if errMsg == "" {
		return ""
	}

	if strings.HasPrefix(errMsg, spaces) {
		return errMsg
	}

	split := strings.Split(strings.TrimSuffix(errMsg, "\n"), "\n")

	lines := make([]string, 0, len(split))
	for _, s := range split {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}

		lines = append(lines, spaces+StatusError().Symbol+" "+trimmed)
	}

	return strings.Join(lines, "\n") + "\n"
}

// formatRenderOutput formats the rendered YAML raw output for display.
// Returns header "Render:" then body (success: resource tree; failure: raw output indented).
// Returns "" when render succeeded and the section would not be shown (!Verbose && !ShowRender).
func (tcr *TestCaseResult) formatRenderOutput() string {
	const header = "Render:"

	if !tcr.HasFailedRender && (!tcr.Verbose || !tcr.ShowRender) {
		return ""
	}

	if tcr.HasFailedRender {
		outputStr := strings.TrimSuffix(string(tcr.RawRenderOutput), "\n")
		split := strings.Split(outputStr, "\n")
		lines := make([]string, 0, 1+len(split))
		lines = append(lines, spaces+header)
		// 1st line: "        [!] line1"
		// 2nd line: "             line2" // 13 spaces so line2 sits under [!] block
		continuationIndent := spaces + spaces + "     " // 8+5 = 13 spaces

		for i, line := range split {
			if i == 0 {
				lines = append(lines, spaces+spaces+StatusError().Symbol+" "+line)
			} else {
				lines = append(lines, continuationIndent+line)
			}
		}

		return strings.Join(lines, "\n") + "\n"
	}

	// Pre-allocate lines slice with capacity for all resources
	lines := make([]string, 1, len(tcr.RenderedResources)+1)
	lines[0] = fmt.Sprintf("%s%s", spaces, header)

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

	return strings.Join(lines, "\n") + "\n"
}

// formatValidateOutput formats the validation raw output for display.
// Returns header "Validate:" plus upstream body (original order), indented.
// When failed and !ShowValidate, filters out "[✓] ... validated successfully" lines.
// Returns "" when validate passed and the section would not be shown (!Verbose && !ShowValidate).
func (tcr *TestCaseResult) formatValidateOutput() string {
	const header = "Validate:"

	if !tcr.HasFailedValidate && (!tcr.Verbose || !tcr.ShowValidate) {
		return ""
	}

	outputStr := strings.TrimSpace(string(tcr.RawValidateOutput))
	if tcr.HasFailedValidate && !tcr.ShowValidate {
		lines := strings.Split(outputStr, "\n")

		filtered := lines[:0]
		for _, line := range lines {
			if !strings.HasSuffix(line, "validated successfully") {
				filtered = append(filtered, line)
			}
		}

		outputStr = strings.Join(filtered, "\n")
	}

	bodyIndent := "\n" + spaces + spaces
	body := strings.ReplaceAll(outputStr, "\n", bodyIndent)

	return spaces + header + "\n" + spaces + spaces + body + "\n"
}

// formatHooksOutput formats the hooks output for display for the pre-test or post-test section.
// label is "pre-test" or "post-test". Returns "" when the section would not be shown.
// Otherwise returns either all hooks or only failed, based on hasFailed*, Verbose, and ShowHooks.
func (tcr *TestCaseResult) formatHooksOutput(label string) string {
	var (
		results   []HookResult
		hasFailed bool
	)

	switch label {
	case "pre-test":
		results = tcr.PreTestHooksResults
		hasFailed = tcr.HasFailedPreTestHooks
	case "post-test":
		results = tcr.PostTestHooksResults
		hasFailed = tcr.HasFailedPostTestHooks
	default:
		return ""
	}

	if len(results) == 0 {
		return ""
	}

	if !hasFailed && (!tcr.Verbose || !tcr.ShowHooks) {
		return ""
	}

	showAll := !hasFailed || (tcr.Verbose && tcr.ShowHooks)

	return tcr.formatHooksOutputWithShow(results, label, showAll)
}

// formatHooksOutputWithShow builds the hooks output string. showAll: when true, all hooks; when false, only failed.
func (tcr *TestCaseResult) formatHooksOutputWithShow(hooksResults []HookResult, label string, showAll bool) string {
	const (
		headerPreTest  = "Pre-test Hooks:"
		headerPostTest = "Post-test Hooks:"
	)

	if len(hooksResults) == 0 {
		return ""
	}

	hooks := hooksResults
	if !showAll {
		filtered := hooksResults[:0]
		for i := range hooksResults {
			if hooksResults[i].Error != nil {
				filtered = append(filtered, hooksResults[i])
			}
		}

		if len(filtered) == 0 {
			return ""
		}

		hooks = filtered
	}

	var out []string

	header := headerPostTest
	if label == "pre-test" {
		header = headerPreTest
	}

	out = append(out, fmt.Sprintf("%s%s", spaces, header))

	for _, hook := range hooks {
		title := hook.Command
		if hook.Name != "" {
			title = hook.Name
		}

		if hook.Error != nil {
			var exitErr *exec.ExitError
			if errors.As(hook.Error, &exitErr) {
				out = append(out, fmt.Sprintf("%s%s %s [exit code: %d]", spaces+spaces, StatusFail().Symbol, title, exitErr.ExitCode()))
			} else {
				// Template/rendering and other non-execution failures: use [!] (operational/other).
				out = append(out,
					fmt.Sprintf("%s%s %s", spaces+spaces, StatusError().Symbol, title),
					fmt.Sprintf("%serror: %s", spaces+spaces+spaces, hook.Error.Error()),
				)
			}
		} else {
			out = append(out, fmt.Sprintf("%s%s %s", spaces+spaces, StatusPass().Symbol, title))
		}

		if len(hook.Output) != 0 {
			out = append(out, indentMultilineBody(multilineBodyIndent, string(hook.Output)))
		}
	}

	return strings.Join(out, "\n") + "\n"
}

// formatAssertionsOutput formats assertion results for display for the assertions section.
// Returns "" when the section would not be shown (!hasFailedAssertions && !(Verbose && ShowAssertions)).
// Otherwise returns either all assertions or only failed, based on hasFailedAssertions, Verbose, and ShowAssertions.
func (tcr *TestCaseResult) formatAssertionsOutput() string {
	const header = "Assertions:"

	if !tcr.HasFailedAssertions && (!tcr.Verbose || !tcr.ShowAssertions) {
		return ""
	}

	if len(tcr.AssertionsResults) == 0 {
		return ""
	}

	var passCount, failCount, errorCount int

	for _, r := range tcr.AssertionsResults {
		switch r.Status {
		case StatusPass():
			passCount++
		case StatusFail():
			failCount++
		case StatusError():
			errorCount++
		case StatusSkip():
			// skipped assertions are not counted in pass/fail/error totals
		default:
			errorCount++ // backward compatibility for unknown status
		}
	}

	toList := tcr.AssertionsResults

	if tcr.HasFailedAssertions && (!tcr.Verbose || !tcr.ShowAssertions) {
		failedOrError := make([]AssertionResult, 0, failCount+errorCount)

		for _, r := range tcr.AssertionsResults {
			if r.Status != StatusPass() {
				failedOrError = append(failedOrError, r)
			}
		}

		toList = failedOrError
		if len(toList) == 0 {
			return ""
		}
	}

	lines := make([]string, 0, 2+len(toList))
	lines = append(lines, fmt.Sprintf("%s%s", spaces, header))

	for _, r := range toList {
		if strings.Contains(r.Message, "\n") {
			lines = append(lines,
				fmt.Sprintf("%s%s %s", spaces+spaces, r.Status.Symbol, r.Name),
				indentMultilineBody(multilineBodyIndent, r.Message))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s %s - %s", spaces+spaces, r.Status.Symbol, r.Name, r.Message))
		}
	}

	plural := pluralize.NewClient()
	errorLabel := plural.Pluralize("error", errorCount, true)
	totalLine := fmt.Sprintf("Total: %d assertions, %d successful, %d failed, %s", len(tcr.AssertionsResults), passCount, failCount, errorLabel)
	lines = append(lines, fmt.Sprintf("%s%s", spaces+spaces, totalLine))

	return strings.Join(lines, "\n") + "\n"
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
func (tcr *TestCaseResult) ProcessValidateOutput() {
	tcr.FormattedValidateOutput = tcr.formatValidateOutput()
}

// ProcessPreTestHooksOutput formats the pre-test hooks results and sets hasFailedPreTestHooks.
// It sets FormattedPreTestHooksOutput to the single string that will be printed (or "" when section not shown).
func (tcr *TestCaseResult) ProcessPreTestHooksOutput() {
	if len(tcr.PreTestHooksResults) == 0 {
		return
	}

	for i := range tcr.PreTestHooksResults {
		if tcr.PreTestHooksResults[i].Error != nil {
			tcr.HasFailedPreTestHooks = true
			break
		}
	}

	tcr.FormattedPreTestHooksOutput = tcr.formatHooksOutput("pre-test")
}

// ProcessPostTestHooksOutput formats the post-test hooks results and sets hasFailedPostTestHooks.
// It sets FormattedPostTestHooksOutput to the single string that will be printed (or "" when section not shown).
func (tcr *TestCaseResult) ProcessPostTestHooksOutput() {
	if len(tcr.PostTestHooksResults) == 0 {
		return
	}

	for i := range tcr.PostTestHooksResults {
		if tcr.PostTestHooksResults[i].Error != nil {
			tcr.HasFailedPostTestHooks = true
			break
		}
	}

	tcr.FormattedPostTestHooksOutput = tcr.formatHooksOutput("post-test")
}

// ProcessAssertionsOutput formats the assertion results and sets hasFailedAssertions.
// It sets FormattedAssertionsOutput to the single string that will be printed (all or failed-only, or "" when section not shown).
func (tcr *TestCaseResult) ProcessAssertionsOutput() {
	if len(tcr.AssertionsResults) == 0 {
		return
	}

	for i := range tcr.AssertionsResults {
		s := tcr.AssertionsResults[i].Status
		if s == StatusFail() || s == StatusError() {
			tcr.HasFailedAssertions = true
			break
		}
	}

	tcr.FormattedAssertionsOutput = tcr.formatAssertionsOutput()
}
