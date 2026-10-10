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

package discovery

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/placeholder"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestLoad_Parse(t *testing.T) {
	fs := afero.NewMemMapFs()

	// load calls Load without printing what it reports.
	load := func(fs afero.Fs, path string) (*api.TestSuiteSpec, error) {
		var (
			spec *api.TestSuiteSpec
			err  error
		)

		unittestsUtils.CaptureStderr(func() {
			spec, err = Load(fs, path, Options{})
		})

		return spec, err
	}

	// Setup directories for testing relative paths
	functionsDir := "/myfunctions"
	testSubDir := "/tests"

	require.NoError(t, fs.MkdirAll(functionsDir, 0o755))
	require.NoError(t, fs.MkdirAll(testSubDir, 0o755))

	// Test with absolute path
	absolutePathContent := `
common:
  inputs:
    functions: ` + functionsDir + `
tests:
- name: test1
  inputs:
    xr: xr1.yaml
    composition: comp1.yaml
`

	// Test with relative path (relative to test file)
	relativePathContent := `
common:
  inputs:
    functions: ../myfunctions
tests:
- name: test1
  inputs:
    xr: xr1.yaml
    composition: comp1.yaml
- name: test3
  inputs:
    xr: xr3.yaml
    composition: comp3.yaml
`
	absoluteTestFile := "/absolute_xprin.yaml"
	relativeTestFile := "/tests/relative_xprin.yaml"

	require.NoError(t, afero.WriteFile(fs, absoluteTestFile, []byte(absolutePathContent), 0o644))
	require.NoError(t, afero.WriteFile(fs, relativeTestFile, []byte(relativePathContent), 0o644))

	// Create an invalid test file (missing functions)
	invalidTestContent := `
tests:
- name: test1
  inputs:
    xr: xr1.yaml
    composition: comp1.yaml
`
	invalidTestFile := "/invalid_xprin.yaml"
	require.NoError(t, afero.WriteFile(fs, invalidTestFile, []byte(invalidTestContent), 0o644))

	// Create an invalid test file (missing tests)
	invalidTestContent2 := `
common:
  functions: ./myfunctions
`
	invalidTestFile2 := "/invalid2_xprin.yaml"
	require.NoError(t, afero.WriteFile(fs, invalidTestFile2, []byte(invalidTestContent2), 0o644))

	// Create an invalid YAML file
	invalidYAMLContent := `
common:
  functions: ./myfunctions
tests:
  - this is invalid YAML syntax
    : Missing key
`
	invalidYAMLFile := "/invalid_yaml_xprin.yaml"
	require.NoError(t, afero.WriteFile(fs, invalidYAMLFile, []byte(invalidYAMLContent), 0o644))

	// Load and check the absolute path test file
	config, err := load(fs, absoluteTestFile)
	require.NoError(t, err)
	assert.Equal(t, functionsDir, config.Common.Inputs.Functions)
	assert.Len(t, config.Tests, 1)

	// Load and check the relative path test file - the functions path should be resolved relative to test file
	relConfig, err := load(fs, relativeTestFile)
	require.NoError(t, err)
	assert.Equal(t, "../myfunctions", relConfig.Common.Inputs.Functions) // Should match the raw value, not expanded
	assert.Len(t, relConfig.Tests, 2)

	// Test invalid files
	_, err = load(fs, invalidTestFile)
	require.NoError(t, err) // test is indeed invalid, but we expect it to load without functions, it will fail later in execution

	spec, err := load(fs, invalidTestFile2)
	require.NoError(t, err)
	assert.Nil(t, spec, "a file without test cases has no spec")

	_, err = load(fs, invalidYAMLFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse testsuite file")

	// Test file that doesn't exist
	_, err = load(fs, "/nonexistent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read testsuite file")

	// Test with non-existent functions path
	badFunctionsContent := `
common:
  functions: ./nonexistent_dir
tests:
- name: test1
  inputs:
    xr: xr.yaml
    composition: comp.yaml
`
	badFunctionsFile := "/bad_functions_xprin.yaml"
	require.NoError(t, afero.WriteFile(fs, badFunctionsFile, []byte(badFunctionsContent), 0o644))

	_, err = load(fs, badFunctionsFile)
	require.NoError(t, err)

	// --- Additional coverage for error branches ---
	t.Run("yaml.Unmarshal error", func(t *testing.T) {
		// This YAML is valid but not valid for the struct (e.g., a string instead of a map)
		testFile := "/unmarshal_error.yaml"
		require.NoError(t, afero.WriteFile(fs, testFile, []byte(`common: string_instead_of_map`), 0o644))
		_, err := load(fs, testFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse testsuite file")
	})

	// Test template variable handling
	t.Run("template variable handling", func(t *testing.T) {
		t.Run("with template variables", func(t *testing.T) {
			contentWithTemplateVars := `
common:
  inputs:
    functions: {{ .Repositories.myrepo }}/functions
    crds:
    - {{ .Repositories.otherrepo }}/crds1
    - {{ .Repositories.otherrepo }}/crds2
  hooks:
    pre-test:
    - name: "pre-test hook"
      run: "echo 'pre-test' {{ .Inputs.XR }}"
    post-test:
    - name: "post-test hook"
      run: "echo 'post-test' {{ .Outputs.XR }}"
tests:
- name: test1
  inputs:
    xr: xr1.yaml
    composition: comp1.yaml
`

			testFile := "/template_test_xprin.yaml"
			require.NoError(t, afero.WriteFile(fs, testFile, []byte(contentWithTemplateVars), 0o644))

			// Load should succeed even with template variables (they're processed later)
			config, err := load(fs, testFile)
			require.NoError(t, err)

			// Check that template variables are converted to placeholders during load
			assert.Contains(t, config.Common.Inputs.Functions, placeholder.Create(".Repositories.myrepo"))
			assert.Contains(t, config.Common.Inputs.CRDs[0], placeholder.Create(".Repositories.otherrepo"))
			assert.Contains(t, config.Common.Inputs.CRDs[1], placeholder.Create(".Repositories.otherrepo"))
			assert.Contains(t, config.Common.Hooks.PreTest[0].Run, placeholder.Create(".Inputs.XR"))
			assert.Contains(t, config.Common.Hooks.PostTest[0].Run, placeholder.Create(".Outputs.XR"))
		})

		t.Run("mixed content", func(t *testing.T) {
			contentMixed := `
common:
  inputs:
    functions: {{ .Repositories.myrepo }}/functions
    crds:
    - ./static-crd
    - {{ .Repositories.otherrepo }}/dynamic-crd
  hooks:
    pre-test:
    - name: "mixed hook"
      run: "echo 'static' && echo '{{ .Inputs.XR }}'"
tests:
- name: test1
  inputs:
    xr: xr1.yaml
    composition: comp1.yaml
`

			testFile := "/mixed_test_xprin.yaml"
			require.NoError(t, afero.WriteFile(fs, testFile, []byte(contentMixed), 0o644))

			config, err := load(fs, testFile)
			require.NoError(t, err)

			// Check mixed content
			assert.Contains(t, config.Common.Inputs.Functions, placeholder.Create(".Repositories.myrepo"))
			assert.Equal(t, "./static-crd", config.Common.Inputs.CRDs[0])
			assert.Contains(t, config.Common.Inputs.CRDs[1], placeholder.Create(".Repositories.otherrepo"))
			assert.Contains(t, config.Common.Hooks.PreTest[0].Run, placeholder.Create(".Inputs.XR"))
		})
	})
}

func TestLoad(t *testing.T) {
	write := func(t *testing.T, content string) afero.Fs {
		t.Helper()

		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/a_xprin.yaml", []byte(content), 0o644))

		return fs
	}

	t.Run("a valid file", func(t *testing.T) {
		spec, err := Load(write(t, "tests:\n- name: one\n  id: one\n"), "/a_xprin.yaml", Options{})
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Equal(t, "one", spec.Tests[0].Name)
	})

	t.Run("a file without test cases is reported, and is not an error", func(t *testing.T) {
		var (
			spec *api.TestSuiteSpec
			err  error
		)

		out := unittestsUtils.CaptureStderr(func() {
			spec, err = Load(write(t, "tests: []\n"), "/a_xprin.yaml", Options{})
		})

		require.NoError(t, err)
		assert.Nil(t, spec)
		assert.Contains(t, out, "?   \t/a_xprin.yaml\t[no test cases found]")
	})

	t.Run("quiet does not report a file without test cases", func(t *testing.T) {
		var err error

		out := unittestsUtils.CaptureStderr(func() {
			_, err = Load(write(t, "tests: []\n"), "/a_xprin.yaml", Options{Quiet: true})
		})

		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("a file that does not parse is reported as an error", func(t *testing.T) {
		var err error

		out := unittestsUtils.CaptureStderr(func() {
			_, err = Load(write(t, "tests: [unclosed"), "/a_xprin.yaml", Options{})
		})

		require.Error(t, err)
		assert.Contains(t, out, "FAIL\t/a_xprin.yaml\t[invalid testsuite file]")
	})

	t.Run("duplicate IDs are reported as an error", func(t *testing.T) {
		var err error

		out := unittestsUtils.CaptureStderr(func() {
			_, err = Load(write(t, "tests:\n- name: one\n  id: same\n- name: two\n  id: same\n"), "/a_xprin.yaml", Options{})
		})

		require.ErrorContains(t, err, "duplicate test case ID")
		assert.Contains(t, out, "FAIL\t/a_xprin.yaml\t[invalid testsuite file]")
	})
}
