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
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func newUpdateGoldensRunner(fs afero.Fs) *Runner {
	options := &testexecutionUtils.Options{UpdateGoldens: true}
	r := NewRunner(options, "/suite_xprin.yaml", &api.TestSuiteSpec{})
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
