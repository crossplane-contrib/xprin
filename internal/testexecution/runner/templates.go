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
	// Cross-test references (available in hooks)
	Tests map[string]*engine.TestCaseResult // Test ID to test case result mapping
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
func (r *Runner) processTemplateVariables(testCase *api.TestCase, needsResults map[string]*engine.TestCaseResult) error {
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

	content, err = r.renderTemplate(content, templateContext, "testcase")
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Parse the processed YAML back to test case and restore hooks
	if err := r.restoreHooks(content, testCase, originalHooks); err != nil {
		return fmt.Errorf("failed to restore hooks: %w", err)
	}

	return nil
}

// renderTemplate renders Go template syntax with the given context.
func (r *Runner) renderTemplate(content string, templateContext *templateContext, templateName string) (string, error) {
	// Parse and execute template
	tmpl, err := templates.Parse(templateName, content)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateContext); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", r.explainMissingNeeds(err))
	}

	return buf.String(), nil
}

// missingKey finds the key in the error of a template that uses `.Tests` for a test case that is not there:
// `.Tests.<id>` fails with "map has no entry for key", and `(index .Tests "<id>")`, the way to use an ID that
// is not a field name (a hyphen), gives nothing for it, so using its outputs fails with a nil pointer.
var missingKey = regexp.MustCompile(`<[^>]*\.Tests[^>]*>: map has no entry for key "([^"]*)"|at <"([^"]*)">: nil pointer evaluating \*engine\.TestCaseResult\.`)

// explainMissingNeeds replaces the error of a template that uses `.Tests` for a test case of the file that
// is not in `needs` (the usual reason for it not being there) with one that says so. Any other error is returned as it is.
func (r *Runner) explainMissingNeeds(err error) error {
	match := missingKey.FindStringSubmatch(err.Error())
	if match == nil {
		return err
	}

	id := match[1] + match[2] // one of them is set, depending on how the test case is used

	if r.testSuite.Spec.HasID(id) {
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
