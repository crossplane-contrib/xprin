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
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestRenderTemplate_MissingNeed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}, {ID: "c", Needs: []string{"b"}}}}
	r := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))
	ctx := newTemplateContext(nil, api.Inputs{}, nil, map[string]*engine.TestCaseResult{"b": {}})

	t.Run("a testcase that exists but is not listed", func(t *testing.T) {
		_, err := r.renderTemplate("{{ .Tests.a.Outputs.XR }}", ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "map has no entry")
	})

	t.Run("a testcase that exists but is not listed, used with index (an ID with a hyphen)", func(t *testing.T) {
		_, err := r.renderTemplate(`{{ (index .Tests "a").Outputs.XR }}`, ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "nil pointer")
	})

	t.Run("a resource of one that is not listed, with index on its outputs", func(t *testing.T) {
		_, err := r.renderTemplate(`{{ index .Tests.a.Outputs.Rendered "Kind/name" }}`, ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "map has no entry")
	})

	t.Run("index with a name that is no testcase keeps the original error", func(t *testing.T) {
		_, err := r.renderTemplate(`{{ (index .Tests "typo").Outputs.XR }}`, ctx, "test")

		require.ErrorContains(t, err, "nil pointer evaluating")
		assert.NotContains(t, err.Error(), "not listed in needs")
	})

	t.Run("index with one that is listed works", func(t *testing.T) {
		out, err := r.renderTemplate(`{{ (index .Tests "b").Outputs.XR }}x`, ctx, "test")

		require.NoError(t, err)
		assert.Equal(t, "x", out)
	})

	t.Run("something that is not a testcase", func(t *testing.T) {
		_, err := r.renderTemplate("{{ .Repositories.nope }}", ctx, "test")

		require.ErrorContains(t, err, `map has no entry for key "nope"`)
		assert.NotContains(t, err.Error(), "not listed in needs")
	})

	t.Run("a file without any needs: nothing is found from the templates", func(t *testing.T) {
		noNeeds := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}}}))
		_, err := noNeeds.renderTemplate("{{ .Tests.a.Outputs.XR }}", newTemplateContext(nil, api.Inputs{}, nil, map[string]*engine.TestCaseResult{}), "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
	})
}
