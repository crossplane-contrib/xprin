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
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const (
	testDiffGoldenPath = "/suite/golden.yaml"
	testDiffActualPath = "/out/render.yaml"
)

func TestExecuteDiffAssertions(t *testing.T) {
	testSuiteFile := filepath.Join("/suite", "test.yaml")
	expandPath := func(base, path string) (string, error) {
		if path == "" {
			return "", nil
		}

		return filepath.Join(filepath.Dir(base), path), nil
	}

	t.Run("pass when files match (full render)", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		golden := testDiffGoldenPath
		actual := testDiffActualPath

		require.NoError(t, afero.WriteFile(fs, golden, []byte("a\n"), 0o644))
		require.NoError(t, afero.WriteFile(fs, actual, []byte("a\n"), 0o644))

		outputs := &engine.Outputs{Render: actual, Rendered: map[string]string{}}
		assertions := []api.AssertionGoldenFile{
			{Name: "full render matches", Expected: "golden.yaml"},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusPass(), all[0].Status)
	})

	t.Run("fail when files differ", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		golden := testDiffGoldenPath
		actual := testDiffActualPath

		require.NoError(t, afero.WriteFile(fs, golden, []byte("expected\n"), 0o644))
		require.NoError(t, afero.WriteFile(fs, actual, []byte("actual\n"), 0o644))

		outputs := &engine.Outputs{Render: actual, Rendered: map[string]string{}}
		assertions := []api.AssertionGoldenFile{
			{Name: "should differ", Expected: "golden.yaml"},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusFail(), all[0].Status)
		assert.Contains(t, all[0].Message, "--- ")
		assert.Contains(t, all[0].Message, "+++ ")
		assert.Contains(t, all[0].Message, "expected")
		assert.Contains(t, all[0].Message, "actual")
	})

	t.Run("actual from resource when resource set", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		goldenPath := "/suite/golden-pod.yaml"
		resourcePath := "/out/Pod_foo.yaml"
		content := []byte("apiVersion: v1\nkind: Pod\n")
		require.NoError(t, afero.WriteFile(fs, goldenPath, content, 0o644))
		require.NoError(t, afero.WriteFile(fs, resourcePath, content, 0o644))

		outputs := &engine.Outputs{
			Render:   testDiffActualPath,
			Rendered: map[string]string{"Pod/foo": resourcePath},
		}
		assertions := []api.AssertionGoldenFile{
			{Name: "pod matches", Expected: "golden-pod.yaml", Resource: "Pod/foo"},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusPass(), all[0].Status)
	})

	t.Run("error when resource not in Rendered", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		outputs := &engine.Outputs{Render: testDiffActualPath, Rendered: map[string]string{}}
		assertions := []api.AssertionGoldenFile{
			{Name: "missing resource", Expected: "g.yaml", Resource: "Pod/nonexistent"},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusError(), all[0].Status)
		assert.Contains(t, all[0].Message, "not found in render output")
	})

	t.Run("error when expandPath errors", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		expandFail := func(string, string) (string, error) { return "", assert.AnError }
		outputs := &engine.Outputs{Render: testDiffActualPath, Rendered: map[string]string{}}
		assertions := []api.AssertionGoldenFile{
			{Name: "bad path", Expected: "golden.yaml"},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandFail, false)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusError(), all[0].Status)
		assert.Contains(t, all[0].Message, "invalid expected path")
	})

	t.Run("colorize adds ANSI codes when true", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		golden := testDiffGoldenPath
		actual := testDiffActualPath

		require.NoError(t, afero.WriteFile(fs, golden, []byte("a\n"), 0o644))
		require.NoError(t, afero.WriteFile(fs, actual, []byte("b\n"), 0o644))
		outputs := &engine.Outputs{Render: actual, Rendered: map[string]string{}}
		assertions := []api.AssertionGoldenFile{{Name: "diff", Expected: "golden.yaml"}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, true)
		all := exec.executeAssertionsDiff(assertions)
		require.Len(t, all, 1)
		assert.Equal(t, engine.StatusFail(), all[0].Status)
		assert.Contains(t, all[0].Message, "\033[31m")
		assert.Contains(t, all[0].Message, "\033[32m")
		assert.Contains(t, all[0].Message, "\033[0m")
	})
}
