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

package templates

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/placeholder"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestParse(t *testing.T) {
	t.Run("a valid template", func(t *testing.T) {
		tmpl, err := Parse("t", "{{ .Name }}")
		require.NoError(t, err)
		assert.Equal(t, "t", tmpl.Name())
	})

	t.Run("a syntax error", func(t *testing.T) {
		_, err := Parse("t", "{{ if .Name }}")
		require.ErrorContains(t, err, "failed to parse template")
	})

	t.Run("an unknown function", func(t *testing.T) {
		_, err := Parse("t", "{{ nope .Name }}")
		require.Error(t, err)
	})

	t.Run("a hyphen in a dotted field", func(t *testing.T) {
		_, err := Parse("t", "{{ .Tests.test-case-1.Outputs.XR }}")
		require.ErrorContains(t, err, "bad character U+002D '-'")
	})
}

func TestMarshalWithoutHooks(t *testing.T) {
	xr := placeholder.Create(".Repositories.myrepo") + "/xr.yaml"
	hookVar := placeholder.Create(".Inputs.XR")
	assertionVar := placeholder.Create(".Tests.a.Outputs.XR")

	testCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          xr,
			Composition: "comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest:  []api.Hook{{Name: "setup", Run: "echo " + hookVar}},
			PostTest: []api.Hook{{Name: "cleanup", Run: "echo cleaning"}},
		},
		Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "assertion", Type: "Count", Value: assertionVar}}},
	}

	data, err := MarshalWithoutHooks(testCase)
	require.NoError(t, err)

	out := string(data)

	assert.Contains(t, out, "name: test")
	assert.Contains(t, out, xr, "the placeholders of the rest of the testcase are kept")
	assert.Contains(t, out, assertionVar, "so are the ones of the assertions")

	for _, fromTheHooks := range []string{"setup", "cleanup", "echo", hookVar} {
		assert.NotContains(t, out, fromTheHooks)
	}

	// The caller's testcase keeps its hooks
	require.Len(t, testCase.Hooks.PreTest, 1)
	require.Len(t, testCase.Hooks.PostTest, 1)
	assert.Equal(t, "echo "+hookVar, testCase.Hooks.PreTest[0].Run)
}

func TestMarshalWithoutHooks_NoHooks(t *testing.T) {
	data, err := MarshalWithoutHooks(api.TestCase{Name: "test"})

	require.NoError(t, err)
	assert.Contains(t, string(data), "name: test")
}

func TestMarshalWithoutHooks_Error(t *testing.T) {
	testCase := api.TestCase{Name: "test", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "n", Value: make(chan int)}}}}

	_, err := MarshalWithoutHooks(testCase)

	require.ErrorContains(t, err, "failed to marshal test case to YAML")
}
