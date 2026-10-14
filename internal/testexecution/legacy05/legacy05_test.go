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

package legacy05

import (
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

// hasImplicitNeeds reads the text of the actions of a template: it is true for any use of `.Tests`.
func TestHasImplicitNeeds_ReadsTheActions(t *testing.T) {
	tests := map[string]struct {
		text string
		want bool
	}{
		"no template":                    {text: "xr.yaml"},
		"other fields":                   {text: "{{ .Inputs.XR }} {{ .Repositories.x }}"},
		"dotted":                         {text: "{{ .Tests.a.Outputs.XR }}", want: true},
		"dotted without spaces":          {text: "{{.Tests.a.Outputs.XR}}", want: true},
		"the root variable":              {text: "{{ $.Tests.a.Outputs.XR }}", want: true},
		"index with a literal":           {text: `{{ (index .Tests "a").Outputs.XR }}`, want: true},
		"index with a raw string":        {text: "{{ (index .Tests `a`).Outputs.XR }}", want: true},
		"index with a hyphenated ID":     {text: `{{ (index .Tests "test-case-1").Outputs.XR }}`, want: true},
		"index on a key of Tests":        {text: `{{ index .Tests.a.Outputs.Rendered "Kind/Name" }}`, want: true},
		"index split over lines by YAML": {text: placeholder.Create("index\n    .Tests\n    \"a\""), want: true},
		"in a pipeline":                  {text: `{{ .Tests.a.Outputs.XR | printf "%s" }}`, want: true},
		"in a condition":                 {text: `{{ if .Tests.a.Outputs.XR }}x{{ end }}`, want: true},
		"in a defined template":          {text: `{{ define "x" }}{{ .Tests.a.Outputs.XR }}{{ end }}{{ template "x" . }}`, want: true},
		"range over Tests":               {text: `{{ range $id, $t := .Tests }}{{ $id }}{{ end }}`, want: true},
		"assigned to a variable":         {text: `{{ $t := .Tests }}`, want: true},
		"index with a computed key":      {text: `{{ $k := "a" }}{{ (index .Tests $k).Outputs.XR }}`, want: true},

		// A mention that is not a use counts too: it only costs a warning that was not needed.
		"another field named Tests": {text: `{{ .Foo.Tests.a }}`, want: true},
		"a comment":                 {text: `{{/* .Tests.a */}}`, want: true},
		"a string":                  {text: `{{ printf ".Tests.a" }}`, want: true},
		"prose next to a use":       {text: `Verify .Tests chaining: {{ .Tests.a.Outputs.XR }}`, want: true},

		"text outside the action is not looked at": {text: `see .Tests.a and .Tests in {{ .X }}`},
		"a longer name is not a mention":           {text: `{{ .TestsExtra.a }}`},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testCase := api.TestCase{Name: "b", Inputs: api.Inputs{XR: placeholder.Replace(tc.text)}}

			assert.Equal(t, tc.want, hasImplicitNeeds(testCase))
		})
	}
}

// hasImplicitNeeds looks at the templates the way the runner renders them.
func TestHasImplicitNeeds_LooksAtTheTemplatesTheWayTheRunnerRenders(t *testing.T) {
	template := placeholder.Replace("{{ .Tests.a.Outputs.XR }}")

	tests := map[string]struct {
		testCase api.TestCase
		want     bool
	}{
		"an assertion value (any)": {
			api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: template}}}},
			true,
		},
		"a nested assertion value": {
			api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: []any{map[string]any{"k": template}}}}}},
			true,
		},
		"a context file": {
			api.TestCase{Name: "b", Inputs: api.Inputs{ContextFiles: map[string]string{"key": template}}},
			true,
		},
		"a list of CRDs": {
			api.TestCase{Name: "b", Inputs: api.Inputs{CRDs: []string{"crd.yaml", template}}},
			true,
		},
		"a pre-test hook": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PreTest: []api.Hook{{Run: template}}}},
			true,
		},
		"a post-test hook": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PostTest: []api.Hook{{Run: template}}}},
			true,
		},
		"the name of a hook is not a template, as for the runner": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PreTest: []api.Hook{{Name: template, Run: "echo"}}}},
			false,
		},
		"a template split over lines by the YAML of a long text": {
			api.TestCase{Name: "b", Inputs: api.Inputs{XR: strings.Repeat("some text ", 30) + template + " " + strings.Repeat("more text ", 30)}},
			true,
		},
		"nothing": {
			api.TestCase{Name: "b", Inputs: api.Inputs{XR: "xr.yaml"}},
			false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasImplicitNeeds(tc.testCase))
		})
	}
}

func TestHasImplicitNeeds_ATestCaseThatCanNotBeMarshalledIsAssumedToUseThem(t *testing.T) {
	testCase := api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: make(chan int)}}}}

	// The runner fails that testcase when it runs, and nothing can be told about what it uses.
	assert.True(t, hasImplicitNeeds(testCase))
}
