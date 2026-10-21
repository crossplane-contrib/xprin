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
	"slices"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func newUpdateGoldensRunner(fs afero.Fs) *Runner {
	options := &testexecutionOptions.Options{UpdateGoldens: true}
	r := NewRunner(options, testsuite.New("/suite_xprin.yaml", &api.TestSuiteSpec{}))
	r.fs = fs
	r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) { return path, nil }

	return r
}

func TestRunUpdateGoldens_IsNoOpWhenNoGoldenAssertions(t *testing.T) {
	fs := afero.NewMemMapFs()
	r := newUpdateGoldensRunner(fs)

	testCase := api.TestCase{
		Name: "no assertions",
		Assertions: api.Assertions{
			Xprin: []api.AssertionXprin{{Name: "count", Type: "Count", Value: 1}},
		},
	}
	result := engine.NewTestCaseResult(testCase.Name, "", false, false, false, false, false)

	out := r.runUpdateGoldens(testCase, result)

	// runUpdateGoldens is only called when diff/dyff assertions exist (runTestCase short-circuits
	// before reaching it otherwise). With no golden assertions, it is a no-op: PASS, nothing written.
	assert.Equal(t, engine.StatusPass(), out.Status)
	assert.Empty(t, out.WrittenGoldenFiles)
}

func TestRunUpdateGoldens_WritesDiffAndDyffAssertions(t *testing.T) {
	fs := afero.NewMemMapFs()
	r := newUpdateGoldensRunner(fs)

	renderContent := []byte("apiVersion: v1\nkind: ConfigMap\n")
	require.NoError(t, afero.WriteFile(fs, "/actual/render.yaml", renderContent, 0o600))

	testCase := api.TestCase{
		Name: "golden write",
		Assertions: api.Assertions{
			Diff: []api.AssertionGoldenFile{
				{Name: "full render diff", Expected: "/expected/golden-diff.yaml"},
			},
			Dyff: []api.AssertionGoldenFile{
				{Name: "full render dyff", Expected: "/expected/golden-dyff.yaml"},
			},
		},
	}
	result := engine.NewTestCaseResult(testCase.Name, "", false, false, false, false, false)
	result.Outputs.Render = "/actual/render.yaml"

	out := r.runUpdateGoldens(testCase, result)

	require.Equal(t, engine.StatusPass(), out.Status)
	require.Equal(t, []string{"/expected/golden-diff.yaml", "/expected/golden-dyff.yaml"}, out.WrittenGoldenFiles)

	gotDiff, err := afero.ReadFile(fs, "/expected/golden-diff.yaml")
	require.NoError(t, err)
	assert.Equal(t, renderContent, gotDiff)

	gotDyff, err := afero.ReadFile(fs, "/expected/golden-dyff.yaml")
	require.NoError(t, err)
	assert.Equal(t, renderContent, gotDyff)
}

func TestRunUpdateGoldens_WritesResourceScopedAssertion(t *testing.T) {
	fs := afero.NewMemMapFs()
	r := newUpdateGoldensRunner(fs)

	resourceContent := []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: my-pod\n")
	require.NoError(t, afero.WriteFile(fs, "/actual/pod.yaml", resourceContent, 0o600))

	testCase := api.TestCase{
		Name: "resource scoped",
		Assertions: api.Assertions{
			Diff: []api.AssertionGoldenFile{
				{Name: "pod diff", Expected: "/expected/pod.yaml", Resource: "Pod/my-pod"},
			},
		},
	}
	result := engine.NewTestCaseResult(testCase.Name, "", false, false, false, false, false)
	result.Outputs.Rendered = map[string]string{"Pod/my-pod": "/actual/pod.yaml"}

	out := r.runUpdateGoldens(testCase, result)

	require.Equal(t, engine.StatusPass(), out.Status)
	require.Equal(t, []string{"/expected/pod.yaml"}, out.WrittenGoldenFiles)

	got, err := afero.ReadFile(fs, "/expected/pod.yaml")
	require.NoError(t, err)
	assert.Equal(t, resourceContent, got)
}

func TestRunUpdateGoldens_FailsOnUnknownResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	r := newUpdateGoldensRunner(fs)

	testCase := api.TestCase{
		Name: "unknown resource",
		Assertions: api.Assertions{
			Diff: []api.AssertionGoldenFile{
				{Name: "missing", Expected: "/expected/missing.yaml", Resource: "Pod/ghost"},
			},
		},
	}
	result := engine.NewTestCaseResult(testCase.Name, "", false, false, false, false, false)
	result.Outputs.Rendered = map[string]string{}

	out := r.runUpdateGoldens(testCase, result)

	assert.Equal(t, engine.StatusFail(), out.Status)
	assert.Empty(t, out.WrittenGoldenFiles)
}

// goldenCase is a test case with golden files, which update-goldens writes.
func goldenCase(id string, needs ...string) api.TestCase {
	return api.TestCase{
		Name:       id,
		ID:         id,
		Needs:      needs,
		Assertions: api.Assertions{Diff: []api.AssertionGoldenFile{{Name: "g", Expected: id + ".yaml"}}},
	}
}

// plainCase is a test case without golden files.
func plainCase(id string, needs ...string) api.TestCase {
	return api.TestCase{Name: id, ID: id, Needs: needs}
}

// updateGoldensRan runs the test cases of the spec as update-goldens does, where the ones in failing fail
// and the others pass, and returns what was printed and which of them actually ran.
func updateGoldensRan(t *testing.T, spec *api.TestSuiteSpec, failing ...string) (output string, ran []string) {
	t.Helper()

	r := NewRunner(&testexecutionOptions.Options{UpdateGoldens: true}, testsuite.New("/tests/a_xprin.yaml", spec))

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

	_ = r.RunTests() // it fails when a test case does: what matters is what ran, and what was printed

	return out.String(), ran
}

func TestRunTests_UpdateGoldensRunsTheOnesThatWriteAndWhatTheyNeed(t *testing.T) {
	tests := []struct {
		name    string
		tests   []api.TestCase
		failing []string
		ran     []string
		// shown in the output, and not shown
		shown, hidden []string
	}{
		{
			name:  "S1: a test case without golden files that one with them needs runs",
			tests: []api.TestCase{plainCase("a"), goldenCase("b", "a")},
			ran:   []string{"a", "b"},
		},
		{
			name:  "S2: needed only by a test case that is skipped, it is skipped too",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a")},
			ran:   nil,
		},
		{
			name:  "S3: a chain that ends in one with golden files runs, all of it",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a"), goldenCase("c", "b")},
			ran:   []string{"a", "b", "c"},
		},
		{
			name:  "S4: a chain without golden files at all is skipped, all of it",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a"), plainCase("c", "b")},
			ran:   nil,
		},
		{
			name:    "S5: one that fails and is needed by one with golden files: it is skipped, with the reason",
			tests:   []api.TestCase{plainCase("a"), goldenCase("b", "a")},
			failing: []string{"a"},
			ran:     []string{"a"},
			shown:   []string{"--- FAIL: a", "--- SKIP: b", `[!] needs failed test case "a"`},
		},
		{
			name:    "S6: one that fails and is needed only by one that is skipped: it is skipped too, quietly",
			tests:   []api.TestCase{plainCase("a"), plainCase("b", "a")},
			failing: []string{"a"},
			ran:     nil,
			hidden:  []string{"FAIL", "needs failed"},
		},
		{
			name:  "S7: a test case with golden files that something needs is not a reason to run what needs it",
			tests: []api.TestCase{goldenCase("a"), plainCase("b", "a")},
			ran:   []string{"a"},
		},
		{
			name:  "S8: two that need the same one: it runs once, and the one without golden files is skipped",
			tests: []api.TestCase{plainCase("a"), goldenCase("b", "a"), plainCase("c", "a")},
			ran:   []string{"a", "b"},
		},
		{
			name:  "S9: a chain with golden files at the end runs, and a branch that ends without them does not",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a"), goldenCase("c", "b"), plainCase("d", "a")},
			ran:   []string{"a", "b", "c"},
		},
		{
			name:    "S10: a chain with golden files at the end, when its first one fails: the rest is skipped, with the reasons",
			tests:   []api.TestCase{plainCase("a"), plainCase("b", "a"), goldenCase("c", "b")},
			failing: []string{"a"},
			ran:     []string{"a"},
			shown:   []string{"--- FAIL: a", "--- SKIP: b", "--- SKIP: c", `[!] needs failed test case "a"`, `[!] needs skipped test case "b"`},
		},
		{
			name:  "the needs are listed after the test case that needs them",
			tests: []api.TestCase{goldenCase("c", "b"), plainCase("b", "a"), plainCase("a"), plainCase("d")},
			ran:   []string{"a", "b", "c"},
		},
		{
			name:  "a test case that is skipped is listed between the ones that run: the others are not shifted",
			tests: []api.TestCase{goldenCase("x", "y"), plainCase("z"), plainCase("y")},
			ran:   []string{"y", "x"},
		},
		{
			name:  "a test case without an ID that has golden files runs, and nothing else does",
			tests: []api.TestCase{{Name: "no id", Assertions: goldenCase("x").Assertions}, plainCase("a")},
			ran:   []string{""},
		},
		{
			name:  "a legacy file has no needs: only the ones with golden files run",
			tests: append(legacySpec().Tests, goldenCase("d")),
			ran:   []string{"d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, ran := updateGoldensRan(t, &api.TestSuiteSpec{Tests: tt.tests}, tt.failing...)

			assert.Equal(t, tt.ran, ran)

			for _, s := range tt.shown {
				assert.Contains(t, out, s)
			}

			for _, s := range tt.hidden {
				assert.NotContains(t, out, s)
			}
		})
	}
}
