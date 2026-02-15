/*
Copyright 2025 The Crossplane Authors.

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

func TestNewAssertionExecutor(t *testing.T) {
	outputs := &engine.Outputs{
		Rendered: make(map[string]string),
	}

	executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

	assert.NotNil(t, executor)
	assert.Equal(t, outputs, executor.outputs)
	assert.False(t, executor.debug)

	executorWithDebug := newAssertionExecutor(afero.NewMemMapFs(), outputs, true, "", nil, false)
	assert.True(t, executorWithDebug.debug)
}

func TestAssertionExecutor_resolveAndReadGoldenFile(t *testing.T) {
	testSuiteFile := filepath.Join("/suite", "test.yaml")
	expandPath := func(base, path string) (string, error) {
		if path == "" {
			return "", nil
		}

		return filepath.Join(filepath.Dir(base), path), nil
	}

	t.Run("success with full render", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		expectedPath := testDiffGoldenPath
		actualPath := testDiffActualPath
		content := []byte("same content\n")
		require.NoError(t, afero.WriteFile(fs, expectedPath, content, 0o644))
		require.NoError(t, afero.WriteFile(fs, actualPath, content, 0o644))

		outputs := &engine.Outputs{Render: actualPath, Rendered: map[string]string{}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		a := api.AssertionGoldenFile{Name: "full", Expected: "golden.yaml"}

		expPath, actPath, expBytes, actBytes, fail := exec.resolveAndReadGoldenFile(a)
		require.Nil(t, fail)
		assert.Equal(t, expectedPath, expPath)
		assert.Equal(t, actualPath, actPath)
		assert.Equal(t, expBytes, content)
		assert.Equal(t, content, actBytes)
	})

	t.Run("success with resource", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		expectedPath := "/suite/golden-pod.yaml"
		resourcePath := "/out/Pod_foo.yaml"
		content := []byte("apiVersion: v1\nkind: Pod\n")
		require.NoError(t, afero.WriteFile(fs, expectedPath, content, 0o644))
		require.NoError(t, afero.WriteFile(fs, resourcePath, content, 0o644))

		outputs := &engine.Outputs{
			Render:   "/out/render.yaml",
			Rendered: map[string]string{"Pod/foo": resourcePath},
		}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		a := api.AssertionGoldenFile{Name: "pod", Expected: "golden-pod.yaml", Resource: "Pod/foo"}

		expPath, actPath, expBytes, actBytes, fail := exec.resolveAndReadGoldenFile(a)
		require.Nil(t, fail)
		assert.Equal(t, expectedPath, expPath)
		assert.Equal(t, resourcePath, actPath)
		assert.Equal(t, expBytes, content)
		assert.Equal(t, content, actBytes)
	})

	t.Run("error when expandPath errors", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		expandFail := func(string, string) (string, error) { return "", assert.AnError }
		outputs := &engine.Outputs{Render: "/out/render.yaml", Rendered: map[string]string{}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandFail, false)
		a := api.AssertionGoldenFile{Name: "bad path", Expected: "golden.yaml"}

		_, _, _, _, result := exec.resolveAndReadGoldenFile(a)
		require.NotNil(t, result)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "invalid expected path")
	})

	t.Run("error when resource not in Rendered", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		outputs := &engine.Outputs{Render: "/out/render.yaml", Rendered: map[string]string{}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		a := api.AssertionGoldenFile{Name: "missing", Expected: "g.yaml", Resource: "Pod/nonexistent"}

		_, _, _, _, result := exec.resolveAndReadGoldenFile(a)
		require.NotNil(t, result)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "not found in render output")
	})

	t.Run("error when expected file cannot be read", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		actualPath := testDiffActualPath
		require.NoError(t, afero.WriteFile(fs, actualPath, []byte("x"), 0o644))
		outputs := &engine.Outputs{Render: actualPath, Rendered: map[string]string{}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		a := api.AssertionGoldenFile{Name: "no expected", Expected: "nonexistent.yaml"}

		_, _, _, _, result := exec.resolveAndReadGoldenFile(a)
		require.NotNil(t, result)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "read expected file")
	})

	t.Run("error when actual file cannot be read", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		expectedPath := testDiffGoldenPath
		require.NoError(t, afero.WriteFile(fs, expectedPath, []byte("x"), 0o644))

		outputs := &engine.Outputs{Render: "/nonexistent/actual.yaml", Rendered: map[string]string{}}
		exec := newAssertionExecutor(fs, outputs, false, testSuiteFile, expandPath, false)
		a := api.AssertionGoldenFile{Name: "no actual", Expected: "golden.yaml"}

		_, _, _, _, result := exec.resolveAndReadGoldenFile(a)
		require.NotNil(t, result)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "read actual file")
	})
}
