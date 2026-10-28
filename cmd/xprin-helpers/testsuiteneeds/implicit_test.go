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

package testsuiteneeds

import (
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

// findImplicitNeedsInString reads the text of a template: what is named, and whether there is a use of
// `.Tests` that can't be traced to an ID.
func TestFindImplicitNeedsInString(t *testing.T) {
	tests := map[string]struct {
		text    string
		wantIDs []string
		wantAll bool
	}{
		"no template":                            {text: "xr.yaml"},
		"other fields":                           {text: "{{ .Inputs.XR }} {{ .Repositories.x }}"},
		"dotted":                                 {text: "{{ .Tests.a.Outputs.XR }}", wantIDs: []string{"a"}},
		"dotted without spaces":                  {text: "{{.Tests.a.Outputs.XR}}", wantIDs: []string{"a"}},
		"the root variable":                      {text: "{{ $.Tests.a.Outputs.XR }}", wantIDs: []string{"a"}},
		"index with a literal":                   {text: `{{ (index .Tests "a").Outputs.XR }}`, wantIDs: []string{"a"}},
		"index with a raw string":                {text: "{{ (index .Tests `a`).Outputs.XR }}", wantIDs: []string{"a"}},
		"index without parentheses":              {text: `{{ index .Tests "a" }}`, wantIDs: []string{"a"}},
		"index with a hyphenated ID":             {text: `{{ (index .Tests "test-case-1").Outputs.XR }}`, wantIDs: []string{"test-case-1"}},
		"index on a key of Tests":                {text: `{{ index .Tests.a.Outputs.Rendered "Kind/Name" }}`, wantIDs: []string{"a"}},
		"index split over lines by YAML":         {text: placeholder.Create("index\n    .Tests\n    \"a\""), wantIDs: []string{"a"}},
		"in a pipeline":                          {text: `{{ .Tests.a.Outputs.XR | printf "%s" }}`, wantIDs: []string{"a"}},
		"as an argument":                         {text: `{{ printf "%s" .Tests.a.Outputs.XR }}`, wantIDs: []string{"a"}},
		"in a condition and its else":            {text: `{{ if .Tests.a.Outputs.XR }}x{{ else }}{{ .Tests.b.Outputs.XR }}{{ end }}`, wantIDs: []string{"a", "b"}},
		"in a range body":                        {text: `{{ range .Inputs.CRDs }}{{ $.Tests.a.Outputs.XR }}{{ end }}`, wantIDs: []string{"a"}},
		"in a defined template":                  {text: `{{ define "x" }}{{ .Tests.a.Outputs.XR }}{{ end }}{{ template "x" . }}`, wantIDs: []string{"a"}},
		"the same ID more than once":             {text: `{{ .Tests.a.Outputs.XR }} {{ (index .Tests "a").Outputs.Render }}`, wantIDs: []string{"a", "a"}},
		"several IDs":                            {text: `{{ .Tests.b.Outputs.XR }} {{ .Tests.a.Outputs.XR }}`, wantIDs: []string{"b", "a"}},
		"through a variable that is the context": {text: `{{ $d := . }}{{ $d.Tests.a.Outputs.XR }}`, wantIDs: []string{"a"}},

		// A use that is not one of the two forms can't be traced to an ID: it needs all of them.
		"range over Tests":             {text: `{{ range $id, $t := .Tests }}{{ $id }}{{ end }}`, wantAll: true},
		"assigned to a variable":       {text: `{{ $t := .Tests }}`, wantAll: true},
		"index with a computed key":    {text: `{{ $k := "a" }}{{ (index .Tests $k).Outputs.XR }}`, wantAll: true},
		"passed on":                    {text: `{{ printf "%v" .Tests }}`, wantAll: true},
		"Tests in parentheses":         {text: `{{ (.Tests).a }}`, wantAll: true},
		"Tests piped to index":         {text: `{{ .Tests | index "a" }}`, wantAll: true},
		"Tests as the key of an index": {text: `{{ index .Tests .Inputs.XR }}`, wantAll: true},
		"a number as the key":          {text: `{{ index .Tests 1 }}`, wantAll: true},
		"an ID and the whole of Tests": {text: `{{ .Tests.a.Outputs.XR }}{{ range .Tests }}x{{ end }}`, wantIDs: []string{"a"}, wantAll: true},

		// The text of the actions is read, so a mention that is not a use is a dependency too: it only
		// costs a dependency that is not needed, and never misses one.
		"another field named Tests": {text: `{{ .Foo.Tests.a }}`, wantIDs: []string{"a"}},
		"a comment":                 {text: `{{/* .Tests.a */}}`, wantIDs: []string{"a"}},
		"a string":                  {text: `{{ printf ".Tests.a" }}`, wantIDs: []string{"a"}},
		"text outside the action is not looked at": {text: `see .Tests.a and .Tests in {{ .X }}`},
		"prose next to a use":                      {text: `Verify .Tests chaining: {{ .Tests.a.Outputs.XR }}`, wantIDs: []string{"a"}},
		"a longer name is not a mention":           {text: `{{ .TestsExtra.a }}`},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, unresolved := findImplicitNeedsInString(placeholder.Replace(tc.text))

			assert.ElementsMatch(t, tc.wantIDs, ids)
			assert.Equal(t, tc.wantAll, unresolved)
		})
	}
}

// findImplicitNeeds looks at the templates the way the runner renders them.
func TestFindImplicitNeeds_LooksAtTheTemplatesTheWayTheRunnerRenders(t *testing.T) {
	template := placeholder.Replace("{{ .Tests.a.Outputs.XR }}")

	tests := map[string]struct {
		testCase api.TestCase
		want     []string
	}{
		"an assertion value (any)": {
			api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: template}}}},
			[]string{"a"},
		},
		"a nested assertion value": {
			api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: []any{map[string]any{"k": template}}}}}},
			[]string{"a"},
		},
		"a context file": {
			api.TestCase{Name: "b", Inputs: api.Inputs{ContextFiles: map[string]string{"key": template}}},
			[]string{"a"},
		},
		"a context value": {
			api.TestCase{Name: "b", Inputs: api.Inputs{ContextValues: map[string]string{"key": template}}},
			[]string{"a"},
		},
		"a list of CRDs": {
			api.TestCase{Name: "b", Inputs: api.Inputs{CRDs: []string{"crd.yaml", template}}},
			[]string{"a"},
		},
		"a pre-test hook": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PreTest: []api.Hook{{Run: template}}}},
			[]string{"a"},
		},
		"a post-test hook": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PostTest: []api.Hook{{Run: template}}}},
			[]string{"a"},
		},
		"the name of a hook is not a template, as for the runner": {
			api.TestCase{Name: "b", Hooks: api.Hooks{PreTest: []api.Hook{{Name: template, Run: "echo"}}}},
			nil,
		},
		"a template split over lines by the YAML of a long text": {
			api.TestCase{Name: "b", Inputs: api.Inputs{XR: strings.Repeat("some text ", 30) + template + " " + strings.Repeat("more text ", 30)}},
			[]string{"a"},
		},
		"nothing": {
			api.TestCase{Name: "b", Inputs: api.Inputs{XR: "xr.yaml"}},
			nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ids, needsAll := findImplicitNeeds(tc.testCase)

			assert.ElementsMatch(t, tc.want, ids)
			assert.False(t, needsAll)
		})
	}
}

func TestFindImplicitNeeds_ATestCaseThatCanNotBeMarshalledNeedsWhatCanNotBeTold(t *testing.T) {
	testCase := api.TestCase{Name: "b", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Type: "FieldValue", Value: make(chan int)}}}}

	ids, needsAll := findImplicitNeeds(testCase)

	// The runner fails that testcase when it runs, and nothing can be told about what it uses.
	assert.Empty(t, ids)
	assert.True(t, needsAll)
}

func TestFindImplicitNeeds_BothKindsOfUse(t *testing.T) {
	testCase := api.TestCase{
		Name:   "b",
		Inputs: api.Inputs{XR: placeholder.Replace("{{ .Tests.a.Outputs.XR }}")},
		Hooks:  api.Hooks{PreTest: []api.Hook{{Run: placeholder.Replace("echo {{ range .Tests }}x{{ end }}")}}},
	}

	ids, needsAll := findImplicitNeeds(testCase)

	assert.Equal(t, []string{"a"}, ids)
	assert.True(t, needsAll)
}
