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
	"bytes"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// runWithResults runs the testcases of the spec with RunTests, where the ones in failing fail and the
// others pass, and returns what was printed and which testcases actually ran.
func runWithResults(t *testing.T, spec *api.TestSuiteSpec, failing ...string) (output string, ran []string) {
	t.Helper()

	r := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", spec)

	var out bytes.Buffer

	r.output = &out
	r.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		ran = append(ran, testCase.ID)

		result := engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
		if slices.Contains(failing, testCase.ID) {
			return result.Fail(errors.New("boom"))
		}

		return result
	}

	_ = r.RunTests() // it fails when a testcase does: what matters is what ran, and what was printed

	return out.String(), ran
}

func TestRunTests_Needs(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a"},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "d", Needs: []string{"a", "b"}},
		{Name: "E", ID: "e"},
	}}

	t.Run("nothing fails: everything runs", func(t *testing.T) {
		out, ran := runWithResults(t, spec)

		assert.Equal(t, []string{"a", "b", "c", "d", "e"}, ran)
		assert.NotContains(t, out, "SKIP")
		assert.NotContains(t, out, "FAIL")
	})

	t.Run("the dependents of a failed testcase are skipped, with the reason on follow-up lines, one per need", func(t *testing.T) {
		out, ran := runWithResults(t, spec, "a")

		assert.Equal(t, []string{"a", "e"}, ran)
		assert.Contains(t, out, "--- FAIL: A")
		assert.Contains(t, out, "--- SKIP: B")
		assert.Contains(t, out, "    [!] needs failed test case \"a\"\n")
		assert.Contains(t, out, "--- SKIP: C")
		assert.Contains(t, out, "    [!] needs skipped test case \"b\"\n")
		assert.Contains(t, out, "--- SKIP: D")
		assert.Contains(t, out, "    [!] needs failed test case \"a\"\n    [!] needs skipped test case \"b\"\n")
		assert.NotContains(t, out, "--- SKIP: E")
	})
}

func TestRunTests_Order(t *testing.T) {
	// "a" is listed first, but needs "c" and "b", which are listed after it
	tests := []api.TestCase{
		{Name: "A", ID: "a", Needs: []string{"c", "b"}},
		{Name: "B", ID: "b"},
		{Name: "C", ID: "c"},
		{Name: "D", ID: "d"},
	}

	t.Run("the test cases run after the ones they need, the rest in the order of the file", func(t *testing.T) {
		_, ran := runWithResults(t, &api.TestSuiteSpec{Tests: tests})

		assert.Equal(t, []string{"b", "c", "a", "d"}, ran)
	})

	t.Run("a legacy file runs in the order of the file", func(t *testing.T) {
		_, ran := runWithResults(t, legacySpec())

		assert.Equal(t, []string{"a", "b", "c"}, ran)
	})

	t.Run("it uses the graph it is given, and builds one when it has none", func(t *testing.T) {
		// "a" needs "b": the order is [b, a], and it is only in the graph, the spec of the runner has no needs
		given := graph.Build(&api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a", Needs: []string{"b"}}, {Name: "B", ID: "b"}}})
		r := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a"}, {Name: "B", ID: "b"}}})

		assert.Equal(t, []int{0, 1}, r.runOrder(), "the graph that it builds from its own spec")

		r.SetTestSuiteGraph(given)

		assert.Equal(t, []int{1, 0}, r.runOrder(), "the graph that it is given")
	})

	t.Run("a test case that needs one listed after it is skipped when that one fails", func(t *testing.T) {
		out, ran := runWithResults(t, &api.TestSuiteSpec{Tests: tests}, "b")

		assert.Equal(t, []string{"b", "c", "d"}, ran)
		assert.Contains(t, out, "--- SKIP: A")
		assert.Contains(t, out, `[!] needs failed test case "b"`)
	})
}

func TestRunTests_Needs_UpdateGoldens(t *testing.T) {
	golden := api.Assertions{Dyff: []api.AssertionGoldenFile{{Name: "g"}}}
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a", Assertions: golden},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"a"}, Assertions: golden},
	}}

	r := NewRunner(&testexecutionUtils.Options{UpdateGoldens: true}, "/tests/a_xprin.yaml", spec)

	var out bytes.Buffer

	r.output = &out
	r.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false).Fail(errors.New("boom"))
	}

	_ = r.RunTests()

	assert.NotContains(t, out.String(), "B", "no golden files to write: nothing to say about it, whatever happened to its needs")
	assert.Contains(t, out.String(), "--- SKIP: C")
	assert.Contains(t, out.String(), "    [!] needs failed test case \"a\"\n")
}

// legacySpec is a testsuite file in the format of xprin up to 0.5: the last testcase uses the outputs of
// "a" without listing it.
func legacySpec() *api.TestSuiteSpec {
	return &api.TestSuiteSpec{Tests: []api.TestCase{
		{ID: "a"},
		{ID: "b"},
		{ID: "c", Inputs: api.Inputs{XR: placeholder.Create(".Tests.a.Outputs.XR")}},
	}}
}

func TestGetNeedsResults(t *testing.T) {
	results := engine.NewTestSuiteResult("/tests/a_xprin.yaml", false)
	results.AddResult(engine.NewTestCaseResult("A", "a", false, false, false, false, false))
	results.AddResult(engine.NewTestCaseResult("B", "b", false, false, false, false, false))

	t.Run("a legacy file: everything that ran", func(t *testing.T) {
		r := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", legacySpec())

		needsResults, skipReasons := r.getNeedsResults(api.TestCase{ID: "c"}, results)

		assert.Len(t, needsResults, 2)
		assert.Empty(t, skipReasons)
	})

	t.Run("any other file: only the ones listed", func(t *testing.T) {
		r := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}, {ID: "c", Needs: []string{"b"}}}})

		needsResults, _ := r.getNeedsResults(api.TestCase{ID: "c", Needs: []string{"b"}}, results)
		assert.Equal(t, []string{"b"}, slices.Collect(maps.Keys(needsResults)))

		needsResults, _ = r.getNeedsResults(api.TestCase{ID: "d"}, results)
		assert.Empty(t, needsResults, "a testcase without needs sees nothing")
	})
}

func TestRenderTemplate_MissingNeed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}, {ID: "c", Needs: []string{"b"}}}}
	r := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", spec)
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

	t.Run("a legacy file, also with index", func(t *testing.T) {
		legacy := NewRunner(&testexecutionUtils.Options{}, "/tests/a_xprin.yaml", legacySpec())
		_, err := legacy.renderTemplate("{{ .Tests.a.Outputs.XR }}", ctx, "test")

		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not listed in needs")

		_, err = legacy.renderTemplate(`{{ (index .Tests "a").Outputs.XR }}`, ctx, "test")

		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not listed in needs")
	})
}

func TestRunTests_WarnsAboutAFileInTheFormatOf05(t *testing.T) {
	warn := func(spec *api.TestSuiteSpec) string {
		return unittestsUtils.CaptureStderr(func() {
			runWithResults(t, spec)
		})
	}

	t.Run("a file in the old format gets one warning, with its path and the docs", func(t *testing.T) {
		out := warn(legacySpec())

		assert.Equal(t, 1, strings.Count(out, "WARNING"))
		assert.Contains(t, out, "/tests/a_xprin.yaml: testcases that use the outputs of other testcases (.Tests) will have to list them in `needs` from xprin 0.7.0")
		assert.Contains(t, out, "/release-0.6/docs/how-it-works.md#dependencies-between-test-cases-needs")
	})

	t.Run("any other file gets none", func(t *testing.T) {
		out := warn(&api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b", Needs: []string{"a"}}}})

		assert.Empty(t, out)
	})
}
