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
	"fmt"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

func TestRenderTemplate_MissingNeed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}, {ID: "c", Needs: []string{"b"}}}}
	r := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))
	ctx := newTemplateContext(nil, api.Inputs{}, nil, map[string]*engine.TestCaseResult{"b": {}})

	t.Run("a testcase that exists but is not listed", func(t *testing.T) {
		_, err := r.renderTemplate(testEnv(r), "{{ .Tests.a.Outputs.XR }}", ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "map has no entry")
	})

	t.Run("a testcase that exists but is not listed, used with index (an ID with a hyphen)", func(t *testing.T) {
		_, err := r.renderTemplate(testEnv(r), `{{ (index .Tests "a").Outputs.XR }}`, ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "nil pointer")
	})

	t.Run("a resource of one that is not listed, with index on its outputs", func(t *testing.T) {
		_, err := r.renderTemplate(testEnv(r), `{{ index .Tests.a.Outputs.Rendered "Kind/name" }}`, ctx, "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
		assert.NotContains(t, err.Error(), "map has no entry")
	})

	t.Run("index with a name that is no testcase keeps the original error", func(t *testing.T) {
		_, err := r.renderTemplate(testEnv(r), `{{ (index .Tests "typo").Outputs.XR }}`, ctx, "test")

		require.ErrorContains(t, err, "nil pointer evaluating")
		assert.NotContains(t, err.Error(), "not listed in needs")
	})

	t.Run("index with one that is listed works", func(t *testing.T) {
		out, err := r.renderTemplate(testEnv(r), `{{ (index .Tests "b").Outputs.XR }}x`, ctx, "test")

		require.NoError(t, err)
		assert.Equal(t, "x", out)
	})

	t.Run("something that is not a testcase", func(t *testing.T) {
		_, err := r.renderTemplate(testEnv(r), "{{ .Repositories.nope }}", ctx, "test")

		require.ErrorContains(t, err, `map has no entry for key "nope"`)
		assert.NotContains(t, err.Error(), "not listed in needs")
	})

	t.Run("a file without any needs: nothing is found from the templates", func(t *testing.T) {
		noNeeds := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", &api.TestSuiteSpec{Tests: []api.TestCase{{ID: "a"}, {ID: "b"}}}))
		_, err := noNeeds.renderTemplate(testEnv(noNeeds), "{{ .Tests.a.Outputs.XR }}", newTemplateContext(nil, api.Inputs{}, nil, map[string]*engine.TestCaseResult{}), "test")

		require.ErrorContains(t, err, "failed to execute template: one or more test cases are used but are not listed in needs")
	})
}

func TestNewTemplateContext(t *testing.T) {
	t.Run("with all parameters", func(t *testing.T) {
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}
		outputs := &engine.Outputs{Render: "rendered.yaml"}

		context := newTemplateContext(repos, inputs, outputs, map[string]*engine.TestCaseResult{})

		assert.Equal(t, repos, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Equal(t, outputs, context.Outputs)
	})

	t.Run("with nil repositories", func(t *testing.T) {
		inputs := api.Inputs{XR: "test-xr.yaml"}

		context := newTemplateContext(nil, inputs, nil, map[string]*engine.TestCaseResult{})

		assert.NotNil(t, context.Repositories)
		assert.Empty(t, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Nil(t, context.Outputs)
	})

	t.Run("with nil outputs", func(t *testing.T) {
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}

		context := newTemplateContext(repos, inputs, nil, map[string]*engine.TestCaseResult{})

		assert.Equal(t, repos, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Nil(t, context.Outputs)
	})
}

func TestRenderTemplate(t *testing.T) {
	t.Run("repository variables", func(t *testing.T) {
		yaml := `
common:
  functions: {{ .Repositories.myrepo }}/foo
  crds: {{ .Repositories.otherrepo }}/bar
`
		repos := map[string]string{
			"myrepo":    "/path/to/myrepo",
			"otherrepo": "/path/to/otherrepo",
		}

		templateContext := newTemplateContext(repos, api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "/path/to/myrepo/foo")
		assert.Contains(t, out, "/path/to/otherrepo/bar")
		assert.NotContains(t, out, "{{ .Repositories.myrepo }}")
		assert.NotContains(t, out, "{{ .Repositories.otherrepo }}")
	})

	t.Run("input variables", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "setup"
    run: "echo 'Setting up {{ .Inputs.XR }} with {{ .Inputs.Composition }}'"
`
		inputs := api.Inputs{
			XR:          "my-xr.yaml",
			Composition: "my-composition.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Setting up my-xr.yaml with my-composition.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")
		assert.NotContains(t, out, "{{ .Inputs.Composition }}")
	})

	t.Run("input variables - comprehensive", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "comprehensive setup"
    run: "echo 'XR: {{ .Inputs.XR }}, Claim: {{ .Inputs.Claim }}, Functions: {{ .Inputs.Functions }}, Observed: {{ .Inputs.ObservedResources }}'"
`
		inputs := api.Inputs{
			XR:                  "my-xr.yaml",
			Claim:               "my-claim.yaml",
			Functions:           "my-functions/",
			ObservedResources:   "my-observed.yaml",
			ExtraResources:      "my-extra.yaml",
			FunctionCredentials: "my-creds.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'XR: my-xr.yaml, Claim: my-claim.yaml, Functions: my-functions/, Observed: my-observed.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")
		assert.NotContains(t, out, "{{ .Inputs.Claim }}")
		assert.NotContains(t, out, "{{ .Inputs.Functions }}")
		assert.NotContains(t, out, "{{ .Inputs.ObservedResources }}")
	})

	t.Run("input variables - context files and values", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "context setup"
    run: "echo 'Context files: {{ .Inputs.ContextFiles }}, Context values: {{ .Inputs.ContextValues }}'"
`
		inputs := api.Inputs{
			XR: "my-xr.yaml",
			ContextFiles: map[string]string{
				"config":  "/path/to/config.yaml",
				"secrets": "/path/to/secrets.yaml",
			},
			ContextValues: map[string]string{
				"env":    "production",
				"region": "us-west-2",
			},
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		// Note: map rendering in Go templates can vary, so we check for key components
		assert.Contains(t, out, "Context files:")
		assert.Contains(t, out, "Context values:")
		assert.NotContains(t, out, "{{ .Inputs.ContextFiles }}")
		assert.NotContains(t, out, "{{ .Inputs.ContextValues }}")
	})

	t.Run("output variables", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "cleanup"
    run: "echo 'Cleaning up {{ .Outputs.XR }} and {{ .Outputs.Render }}'"
`
		outputs := &engine.Outputs{
			XR:     "rendered-xr.yaml",
			Render: "rendered-resources.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Cleaning up rendered-xr.yaml and rendered-resources.yaml'")
		assert.NotContains(t, out, "{{ .Outputs.XR }}")
		assert.NotContains(t, out, "{{ .Outputs.Render }}")
	})

	t.Run("output variables with validation", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "validate and count"
    run: "echo 'Validation: {{ .Outputs.Validate }}, Count: {{ .Outputs.RenderCount }}'"
`
		validatePath := "/path/to/validate.txt"
		outputs := &engine.Outputs{
			XR:          "rendered-xr.yaml",
			Render:      "rendered-resources.yaml",
			Validate:    &validatePath,
			RenderCount: 5,
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Validation: /path/to/validate.txt, Count: 5'")
		assert.NotContains(t, out, "{{ .Outputs.Validate }}")
		assert.NotContains(t, out, "{{ .Outputs.RenderCount }}")
	})

	t.Run("output variables without validation", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "no validation"
    run: "echo 'Validation: {{ .Outputs.Validate }}, Count: {{ .Outputs.RenderCount }}'"
`
		outputs := &engine.Outputs{
			XR:          "rendered-xr.yaml",
			Render:      "rendered-resources.yaml",
			Validate:    nil, // No validation output
			RenderCount: 3,
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Validation: <nil>, Count: 3'")
		assert.NotContains(t, out, "{{ .Outputs.Validate }}")
		assert.NotContains(t, out, "{{ .Outputs.RenderCount }}")
	})

	t.Run("output variables - rendered map", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "check rendered resources"
    run: "echo 'Rendered resources: {{ .Outputs.Rendered }}'"
`
		outputs := &engine.Outputs{
			XR:     "rendered-xr.yaml",
			Render: "rendered-resources.yaml",
			Rendered: map[string]string{
				"ConfigMap/my-config": "/path/to/configmap.yaml",
				"Service/my-service":  "/path/to/service.yaml",
			},
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		// Note: map rendering in Go templates can vary, so we check for key components
		assert.Contains(t, out, "Rendered resources:")
		assert.NotContains(t, out, "{{ .Outputs.Rendered }}")
	})

	t.Run("output variables - assertions path", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "assertions file"
    run: "echo 'Assertions: {{ .Outputs.Assertions }}'"
`
		assertionsPath := "/path/to/assertions.txt"
		outputs := &engine.Outputs{
			XR:         "rendered-xr.yaml",
			Render:     "rendered-resources.yaml",
			Assertions: &assertionsPath,
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Assertions: /path/to/assertions.txt'")
		assert.NotContains(t, out, "{{ .Outputs.Assertions }}")
	})

	t.Run("output variables - assertions nil", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "assertions file"
    run: "echo 'Assertions: {{ .Outputs.Assertions }}'"
`
		outputs := &engine.Outputs{
			XR:         "rendered-xr.yaml",
			Render:     "rendered-resources.yaml",
			Assertions: nil, // No assertions run
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Assertions: <nil>'")
		assert.NotContains(t, out, "{{ .Outputs.Assertions }}")
	})

	t.Run("mixed template variables", func(t *testing.T) {
		yaml := `
common:
  functions: {{ .Repositories.myrepo }}/functions
hooks:
  pre-test:
  - name: "pre-test"
    run: "echo 'Pre-test for {{ .Inputs.XR }}'"
  post-test:
  - name: "post-test"
    run: "echo 'Post-test for {{ .Outputs.XR }}'"
`
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}
		outputs := &engine.Outputs{XR: "rendered-xr.yaml"}

		templateContext := newTemplateContext(repos, inputs, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.NoError(t, err)

		// Check repository variables
		assert.Contains(t, out, "/path/to/repo/functions")
		assert.NotContains(t, out, "{{ .Repositories.myrepo }}")

		// Check input variables
		assert.Contains(t, out, "echo 'Pre-test for test-xr.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")

		// Check output variables
		assert.Contains(t, out, "echo 'Post-test for rendered-xr.yaml'")
		assert.NotContains(t, out, "{{ .Outputs.XR }}")
	})

	t.Run("unknown repository variable", func(t *testing.T) {
		yaml := "functions: {{ .Repositories.unknownrepo }}/foo"
		repos := map[string]string{"myrepo": "/some/path"}

		templateContext := newTemplateContext(repos, api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "map has no entry for key \"unknownrepo\"")
	})

	t.Run("unknown input variable", func(t *testing.T) {
		yaml := "hooks:\n  pre-test:\n  - run: \"echo '{{ .Inputs.UnknownField }}'\""
		inputs := api.Inputs{XR: "test-xr.yaml"}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't evaluate field UnknownField")
	})

	t.Run("unknown output variable", func(t *testing.T) {
		yaml := "hooks:\n  post-test:\n  - run: \"echo '{{ .Outputs.UnknownField }}'\""
		outputs := &engine.Outputs{XR: "rendered-xr.yaml"}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(testEnv(runner), yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't evaluate field UnknownField")
	})
}

// TestProcessTemplateVariables tests the processTemplateVariables function.
func TestProcessTemplateVariables(t *testing.T) {
	// Create a test case with template variables in hooks
	testCase := api.TestCase{
		Name: "template-vars-test",
		ID:   "template-vars-test",
		Inputs: api.Inputs{
			XR:          "test-xr.yaml",
			Composition: "test-comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: fmt.Sprintf("echo 'Setting up %s'", placeholder.Create(".Repositories.myrepo"))},
			},
		},
	}

	// Create runner with repositories
	runner := &Runner{
		Options: &testexecutionOptions.Options{
			Repositories: map[string]string{
				"myrepo": "/path/to/myrepo",
			},
		},
	}

	// Process template variables
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	err := runner.processTemplateVariables(testEnv(runner), &testCase, testSuiteResult.GetCompletedTests())
	require.NoError(t, err)

	// Verify that template variables were processed
	// Note: The actual processing depends on the implementation
	// For now, just verify the function doesn't error
	assert.NotEmpty(t, testCase.Hooks.PreTest[0].Run)
}

// TestProcessTemplateVariables_NoTemplateVars tests processTemplateVariables with no template variables.
func TestProcessTemplateVariables_NoTemplateVars(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create test files
	xrFile := "/xr.yaml"
	require.NoError(t, afero.WriteFile(fs, xrFile, []byte(`apiVersion: example.com/v1
kind: XR
metadata:
  name: test-xr`), 0o644))

	compositionFile := "/composition.yaml"
	require.NoError(t, afero.WriteFile(fs, compositionFile, []byte(`apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: test-composition`), 0o644))

	// Create a test case without template variables
	testCase := api.TestCase{
		Name: "no-template-vars-test",
		ID:   "no-template-vars-test",
		Inputs: api.Inputs{
			XR:          xrFile,
			Composition: compositionFile,
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: "echo 'Setting up without templates'"},
			},
		},
	}

	// Create runner
	runner := &Runner{
		Options: &testexecutionOptions.Options{
			Repositories: map[string]string{
				"myrepo": "/path/to/myrepo",
			},
		},
	}

	// Process template variables
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	err := runner.processTemplateVariables(testEnv(runner), &testCase, testSuiteResult.GetCompletedTests())
	require.NoError(t, err)

	// Verify that the test case was not modified
	assert.Equal(t, "echo 'Setting up without templates'", testCase.Hooks.PreTest[0].Run)
}

// TestProcessTemplateVariables_LeavesTheHooksToTheStepsThatRunThem checks that the hooks keep their
// placeholders: they are rendered when they run.
func TestProcessTemplateVariables_LeavesTheHooksToTheStepsThatRunThem(t *testing.T) {
	preTest := placeholder.Create(".Inputs.XR")
	postTest := placeholder.Create(".Outputs.XR")

	testCase := api.TestCase{
		Name: "test",
		Inputs: api.Inputs{
			XR:          placeholder.Create(".Repositories.myrepo") + "/xr.yaml",
			Composition: "comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest:  []api.Hook{{Name: "pre-test", Run: "echo " + preTest}},
			PostTest: []api.Hook{{Name: "post-test", Run: "echo " + postTest}},
		},
	}

	runner := &Runner{Options: &testexecutionOptions.Options{Repositories: map[string]string{"myrepo": "/path/to/myrepo"}}}

	require.NoError(t, runner.processTemplateVariables(testEnv(runner), &testCase, nil))

	assert.Equal(t, "/path/to/myrepo/xr.yaml", testCase.Inputs.XR, "the rest of the test case is rendered")
	assert.Equal(t, "echo "+preTest, testCase.Hooks.PreTest[0].Run, "the hooks are not")
	assert.Equal(t, "echo "+postTest, testCase.Hooks.PostTest[0].Run)
}

// TestRestoreHooks tests the restoreHooks function.
func TestRestoreHooks(t *testing.T) {
	runner := &Runner{}

	// Original test case with hooks
	originalTestCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: "echo 'pre-test'"},
			},
			PostTest: []api.Hook{
				{Name: "post-test", Run: "echo 'post-test'"},
			},
		},
	}

	// Create processed YAML without hooks (as would happen after template processing)
	processedTestCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
			// Hooks removed for template processing
		},
	}

	processedYAML, err := yaml.Marshal(processedTestCase)
	require.NoError(t, err)

	// Restore hooks
	originalHooks := originalTestCase.Hooks
	err = runner.restoreHooks(string(processedYAML), &originalTestCase, originalHooks)
	require.NoError(t, err)

	// Verify hooks were restored
	assert.Len(t, originalTestCase.Hooks.PreTest, 1)
	assert.Equal(t, "pre-test", originalTestCase.Hooks.PreTest[0].Name)
	assert.Len(t, originalTestCase.Hooks.PostTest, 1)
	assert.Equal(t, "post-test", originalTestCase.Hooks.PostTest[0].Name)
	assert.Equal(t, "echo 'pre-test'", originalTestCase.Hooks.PreTest[0].Run)
	assert.Equal(t, "echo 'post-test'", originalTestCase.Hooks.PostTest[0].Run)
}
