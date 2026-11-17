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
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestExecuteErrorsAssertions(t *testing.T) {
	const output = "crossplane: error: cannot render: evaluate hcl: got type string, expected number\n"

	for name, tt := range map[string]struct {
		assertion    api.AssertionError
		renderFailed bool
		wantStatus   engine.Status
		wantMessage  string
	}{
		"any error passes": {
			assertion:    api.AssertionError{Name: "fails", Phase: api.PhaseCrossplaneRender},
			renderFailed: true,
			wantStatus:   engine.StatusPass(),
			wantMessage:  "the render failed",
		},
		"the render succeeded": {
			assertion:   api.AssertionError{Name: "fails", Phase: api.PhaseCrossplaneRender},
			wantStatus:  engine.StatusFail(),
			wantMessage: "the render succeeded, but it was expected to fail",
		},
		"the render succeeded, with an operator": {
			assertion:   api.AssertionError{Name: "fails", Phase: api.PhaseCrossplaneRender, Operator: "contains", Value: "got type"},
			wantStatus:  engine.StatusFail(),
			wantMessage: "the render succeeded, but it was expected to fail",
		},
		"contains": {
			assertion:    api.AssertionError{Name: "text", Phase: api.PhaseCrossplaneRender, Operator: "contains", Value: "got type string"},
			renderFailed: true,
			wantStatus:   engine.StatusPass(),
			wantMessage:  `the render failed, and its output contains "got type string"`,
		},
		"does not contain": {
			assertion:    api.AssertionError{Name: "text", Phase: api.PhaseCrossplaneRender, Operator: "contains", Value: "could not be evaluated"},
			renderFailed: true,
			wantStatus:   engine.StatusFail(),
			wantMessage:  `the render failed, but its output does not contain "could not be evaluated":` + "\n" + strings.TrimSpace(output),
		},
		"the same text, ignoring the whitespace around it": {
			assertion:    api.AssertionError{Name: "exact", Phase: api.PhaseCrossplaneRender, Operator: "==", Value: "  " + strings.TrimSpace(output)},
			renderFailed: true,
			wantStatus:   engine.StatusPass(),
			wantMessage:  "the render failed, and its output is",
		},
		"is, the same as ==": {
			assertion:    api.AssertionError{Name: "exact", Phase: api.PhaseCrossplaneRender, Operator: "is", Value: strings.TrimSpace(output)},
			renderFailed: true,
			wantStatus:   engine.StatusPass(),
			wantMessage:  "the render failed, and its output is",
		},
		"not the same text": {
			assertion:    api.AssertionError{Name: "exact", Phase: api.PhaseCrossplaneRender, Operator: "==", Value: "got type string"},
			renderFailed: true,
			wantStatus:   engine.StatusFail(),
			wantMessage:  "the render failed, but its output is not",
		},
		"an operator without a value": {
			assertion:    api.AssertionError{Name: "x", Phase: api.PhaseCrossplaneRender, Operator: "contains"},
			renderFailed: true,
			wantStatus:   engine.StatusError(),
			wantMessage:  "requires operator and value together, or neither",
		},
		"a value without an operator": {
			assertion:    api.AssertionError{Name: "x", Phase: api.PhaseCrossplaneRender, Value: "text"},
			renderFailed: true,
			wantStatus:   engine.StatusError(),
			wantMessage:  "requires operator and value together, or neither",
		},
		"an operator it does not have": {
			assertion:    api.AssertionError{Name: "x", Phase: api.PhaseCrossplaneRender, Operator: "starts", Value: "text"},
			renderFailed: true,
			wantStatus:   engine.StatusError(),
			wantMessage:  "unsupported operator: starts",
		},
		"matches is not an operator": {
			assertion:    api.AssertionError{Name: "x", Phase: api.PhaseCrossplaneRender, Operator: "matches", Value: "got type"},
			renderFailed: true,
			wantStatus:   engine.StatusError(),
			wantMessage:  "unsupported operator: matches",
		},
		"a phase it does not have": {
			assertion:    api.AssertionError{Name: "x", Phase: "setup"},
			renderFailed: true,
			wantStatus:   engine.StatusError(),
			wantMessage:  `unsupported phase: "setup"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := &engine.TestCaseResult{HasFailedRender: tt.renderFailed, RawRenderOutput: []byte(output)}

			results := executeAssertionsErrors([]api.AssertionError{tt.assertion}, result)

			require.Len(t, results, 1)
			assert.Equal(t, tt.assertion.Name, results[0].Name)
			assert.Equal(t, tt.wantStatus, results[0].Status)
			assert.Contains(t, results[0].Message, tt.wantMessage)
		})
	}

	t.Run("one result for each assertion, in order", func(t *testing.T) {
		results := executeAssertionsErrors([]api.AssertionError{
			{Name: "first", Phase: api.PhaseCrossplaneRender},
			{Name: "second", Phase: api.PhaseCrossplaneRender, Operator: "contains", Value: "nope"},
		}, &engine.TestCaseResult{HasFailedRender: true, RawRenderOutput: []byte(output)})

		require.Len(t, results, 2)
		assert.Equal(t, "first", results[0].Name)
		assert.Equal(t, engine.StatusPass(), results[0].Status)
		assert.Equal(t, "second", results[1].Name)
		assert.Equal(t, engine.StatusFail(), results[1].Status)
	})
}
