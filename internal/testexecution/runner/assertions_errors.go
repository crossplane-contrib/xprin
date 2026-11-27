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
	"fmt"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
)

// executeAssertionsErrors runs errors assertions: checks that the phase of each failed, which the result of the test case says,
// and, if it has an operator and a value, that the output of the phase is what they say.
func executeAssertionsErrors(assertions []api.AssertionError, result *engine.TestCaseResult) []engine.AssertionResult {
	results := make([]engine.AssertionResult, 0, len(assertions))
	for _, assertion := range assertions {
		results = append(results, executeAssertionError(assertion, result))
	}

	return results
}

// phaseOutcome is what an assertion on an error checks: how a phase ended.
type phaseOutcome struct {
	name      string // the phase as it is said in a message: "the render", "validate"
	skipped   bool   // the phase did not run: validate does not without CRDs, or after a render that failed
	skippedBy string // the option that skips the phase, if any
	failed    bool
	output    string
}

// outcomeOf returns how the phase ended in the result, and false for a phase that does not exist.
func outcomeOf(phase string, result *engine.TestCaseResult) (phaseOutcome, bool) {
	switch phase {
	case api.PhaseCrossplaneRender:
		return phaseOutcome{name: "the render", failed: result.HasFailedRender, output: string(result.RawRenderOutput)}, true
	case api.PhaseCrossplaneValidate:
		// The output is written when validate runs: there is none if the test case has no CRDs, or the render failed
		outcome := phaseOutcome{name: "validate", skipped: result.Outputs.Validate == nil, failed: result.HasFailedValidate, output: string(result.RawValidateOutput)}
		if result.Outputs.HasSkippedValidate {
			outcome.skippedBy = "options.skip-validate"
		}

		return outcome, true
	default:
		return phaseOutcome{}, false
	}
}

// executeAssertionError runs a single errors assertion. An operational error is its result, with the status of an error,
// as for the xprin assertions.
func executeAssertionError(assertion api.AssertionError, result *engine.TestCaseResult) engine.AssertionResult {
	outcome, ok := outcomeOf(assertion.Phase, result)
	if !ok {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("unsupported phase: %q", assertion.Phase))
	}

	if outcome.skippedBy != "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("%s: %s is skipped, there is nothing to assert on", outcome.skippedBy, outcome.name))
	}

	if outcome.skipped {
		return engine.NewAssertionResult(assertion.Name, engine.StatusFail(), outcome.name+" did not run, but it was expected to fail")
	}

	if !outcome.failed {
		return engine.NewAssertionResult(assertion.Name, engine.StatusFail(), outcome.name+" succeeded, but it was expected to fail")
	}

	if assertion.Operator == "" && assertion.Value == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusPass(), outcome.name+" failed")
	}

	if assertion.Operator == "" || assertion.Value == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "errors assertion requires operator and value together, or neither")
	}

	output := strings.TrimSpace(outcome.output)

	var (
		passed    bool
		is, isNot string // how the output relates to the value when the assertion passes, and when it does not
	)

	switch assertion.Operator {
	case "==", "is":
		passed, is, isNot = output == strings.TrimSpace(assertion.Value), "is", "is not"
	case "contains":
		passed, is, isNot = strings.Contains(output, assertion.Value), "contains", "does not contain"
	default:
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("unsupported operator: %s", assertion.Operator))
	}

	if passed {
		return engine.NewAssertionResult(assertion.Name, engine.StatusPass(), fmt.Sprintf("%s failed, and its output %s %q", outcome.name, is, assertion.Value))
	}

	return engine.NewAssertionResult(assertion.Name, engine.StatusFail(), fmt.Sprintf("%s failed, but its output %s %q:\n%s", outcome.name, isNot, assertion.Value, output))
}
