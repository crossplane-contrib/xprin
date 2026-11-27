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

package runner

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/templates"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"sigs.k8s.io/yaml"
)

// templateContext provides variables available in test suite templates.
type templateContext struct {
	// Repository variables (for compatibility with existing {{ .Repositories.name }} syntax)
	Repositories map[string]string // Repository name to path mapping
	// Input variables (available in hooks)
	Inputs api.Inputs
	// Output variables (available in post-test hooks)
	Outputs *engine.Outputs
	// What the claim was converted into (available in the post-claim-conversion and the post-test hooks)
	Converted converted
	// Cross-test references (available in hooks)
	Tests map[string]*engine.TestCaseResult // Test ID to test case result mapping
}

// converted is what the claim of a test case was converted into, for `.Converted`. Its fields are methods so that
// using one of them when there is nothing to use is a template error that says why.
type converted struct {
	xr       string // the path of the converted XR: empty when there was no conversion (yet)
	hasClaim bool   // whether the test case has a claim, to say why it is empty
}

// XR is the path of the XR that the claim was converted into: `{{ .Converted.XR }}`.
func (c converted) XR() (string, error) {
	switch {
	case c.xr != "":
		return c.xr, nil
	case c.hasClaim:
		return "", errors.New(".Converted.XR is not available before the claim is converted: use it in the post-claim-conversion and the post-test hooks")
	default:
		return "", errors.New(".Converted.XR is not available: the test case has no claim to convert")
	}
}

// newTemplateContext creates a new template context with the given parameters.
func newTemplateContext(repositories map[string]string, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) *templateContext {
	if repositories == nil {
		repositories = make(map[string]string)
	}

	return &templateContext{
		Repositories: repositories,
		Inputs:       inputs,
		Outputs:      outputs,
		Tests:        tests,
	}
}

// processTemplateVariables processes template variables for a test case.
func (r *Runner) processTemplateVariables(env *testCaseEnv, testCase *api.TestCase, needsResults map[string]*engine.TestCaseResult) error {
	// Check if there are any template variables by converting to YAML temporarily
	yamlData, err := yaml.Marshal(testCase)
	if err != nil {
		return fmt.Errorf("failed to marshal test case to YAML: %w", err)
	}

	content := string(yamlData)

	// Check if there are any template variables
	if !strings.Contains(content, placeholder.Open) {
		return nil // No template variables to process
	}

	// The hooks are rendered when they run, not with the rest of the testcase
	originalHooks := testCase.Hooks

	withoutHooks, err := templates.MarshalWithoutHooks(*testCase)
	if err != nil {
		return fmt.Errorf("failed to remove hooks from YAML: %w", err)
	}

	content = placeholder.Restore(string(withoutHooks))

	// Render template
	templateContext := newTemplateContext(r.Repositories, testCase.Inputs, nil, needsResults)

	content, err = r.renderTemplate(env, content, templateContext, "testcase")
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Parse the processed YAML back to test case and restore hooks
	if err := r.restoreHooks(content, testCase, originalHooks); err != nil {
		return fmt.Errorf("failed to restore hooks: %w", err)
	}

	return nil
}

// runCommandIn is runCommand in the directory of the testsuite file of env, as the hooks take it.
func (r *Runner) runCommandIn(env *testCaseEnv) func(string, ...string) ([]byte, error) {
	return func(name string, args ...string) ([]byte, error) {
		return r.runCommand(env.TestSuite.GetDir(), name, args...)
	}
}

// renderTemplateOf is renderTemplate for the test case of env, as the hooks take it.
func (r *Runner) renderTemplateOf(env *testCaseEnv) func(string, *templateContext, string) (string, error) {
	return func(content string, templateContext *templateContext, templateName string) (string, error) {
		return r.renderTemplate(env, content, templateContext, templateName)
	}
}

// renderTemplate renders Go template syntax with the given context.
func (r *Runner) renderTemplate(env *testCaseEnv, content string, templateContext *templateContext, templateName string) (string, error) {
	// Parse and execute template
	tmpl, err := templates.Parse(templateName, content)
	if err != nil {
		return "", err
	}

	templateContext.Converted = converted{xr: env.convertedXR, hasClaim: templateContext.Inputs.Claim != ""}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateContext); err != nil {
		// index, len and range of an output that is a nil pointer
		if nilPointerUse.MatchString(err.Error()) && renderedUse.MatchString(content) {
			return "", fmt.Errorf("failed to execute template: %s", noValueMessage(content, templateContext.Outputs))
		}

		return "", fmt.Errorf("failed to execute template: %w", r.explainMissingNeeds(env, err))
	}

	// A template does not fail on an output that is not there (a nil pointer): it prints <nil>, and a command that
	// uses it would run with that. A guard (`{{ if .Outputs.Validate }}`) is false for it, so it is not affected.
	if strings.Contains(buf.String(), nilValue) {
		return "", fmt.Errorf("failed to execute template: %s", noValueMessage(content, templateContext.Outputs))
	}

	return buf.String(), nil
}

// nilValue is what a template prints for an output that is not there.
const nilValue = "<nil>"

// nilPointerUse is the error of index, len and range on an output that is a nil pointer, and renderedUse is a template
// that uses .Outputs.Rendered, the one of the outputs that they are used on.
var (
	nilPointerUse = regexp.MustCompile(`(index|len) of nil pointer|range can't iterate over <nil>`)
	renderedUse   = regexp.MustCompile(`\.Outputs\.Rendered\b`)
)

// noValueMessage says which outputs that the template uses have no value, and why. With the outputs of the test case it
// says only the ones that are not there, unless the template uses `.Tests`, whose outputs it can't tell.
func noValueMessage(content string, outputs *engine.Outputs) string {
	// the outputs that are not there when what writes them did not run
	candidates := []struct {
		name, reason string
		isNil        func(*engine.Outputs) bool
	}{
		{".Outputs.Render", "there is no render output", func(o *engine.Outputs) bool { return o.Render == nil }},
		{".Outputs.XR", "there is no rendered XR", func(o *engine.Outputs) bool { return o.XR == nil }},
		{".Outputs.RenderCount", "there is no render output", func(o *engine.Outputs) bool { return o.RenderCount == nil }},
		{".Outputs.Rendered", "there is no render output", func(o *engine.Outputs) bool { return o.Rendered == nil }},
		{".Outputs.Validate", "validate did not run", func(o *engine.Outputs) bool { return o.Validate == nil }},
		{".Outputs.Assertions", "no assertions ran", func(o *engine.Outputs) bool { return o.Assertions == nil }},
	}
	known := outputs != nil && !strings.Contains(content, ".Tests")

	var messages []string

	for _, candidate := range candidates {
		// as a whole name: .Outputs.Render is not .Outputs.RenderCount
		if !regexp.MustCompile(regexp.QuoteMeta(candidate.name) + `\b`).MatchString(content) {
			continue
		}

		if known && !candidate.isNil(outputs) {
			continue
		}

		messages = append(messages, fmt.Sprintf("%s has no value: %s", candidate.name, candidate.reason))
	}

	if len(messages) == 0 {
		return "a template variable has no value (" + nilValue + ")"
	}

	return strings.Join(messages, "; ")
}

// missingKey finds the key in the error of a template that uses `.Tests` for a test case that is not there:
// `.Tests.<id>` fails with "map has no entry for key", and `(index .Tests "<id>")`, the way to use an ID that
// is not a field name (a hyphen), gives nothing for it, so using its outputs fails with a nil pointer.
var missingKey = regexp.MustCompile(`<[^>]*\.Tests[^>]*>: map has no entry for key "([^"]*)"|at <"([^"]*)">: nil pointer evaluating \*engine\.TestCaseResult\.`)

// explainMissingNeeds replaces the error of a template that uses `.Tests` for a test case of the file that
// is not in `needs` (the usual reason for it not being there) with one that says so. Any other error is returned as it is.
func (r *Runner) explainMissingNeeds(env *testCaseEnv, err error) error {
	match := missingKey.FindStringSubmatch(err.Error())
	if match == nil {
		return err
	}

	id := match[1] + match[2] // one of them is set, depending on how the test case is used

	if env.TestSuite.Spec.HasID(id) {
		return errors.New("one or more test cases are used but are not listed in needs")
	}

	return err
}

// restoreHooks parses the processed YAML back to test case and restores hooks.
func (r *Runner) restoreHooks(content string, testCase *api.TestCase, originalHooks api.Hooks) error {
	// Parse the processed YAML back to test case
	if err := yaml.Unmarshal([]byte(content), testCase); err != nil {
		return fmt.Errorf("failed to parse processed YAML: %w", err)
	}

	// Restore hooks (they will be processed in executeHooks with the correct context)
	testCase.Hooks = originalHooks

	return nil
}
