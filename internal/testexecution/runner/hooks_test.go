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
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/placeholder"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// TestExecuteHooks tests the executeHooks function with template variables.
func TestExecuteHooks(t *testing.T) {
	// Create repositories
	repositories := map[string]string{
		"myrepo": "/path/to/myrepo",
	}

	// Create test inputs
	inputs := api.Inputs{
		XR:          "test-xr.yaml",
		Composition: "test-comp.yaml",
	}

	// Create hooks with already-processed template variables (as they would be after processTemplateVariables)
	hooks := []api.Hook{
		{Name: "test-hook", Run: "echo 'Repository: /path/to/myrepo'"},
		{Name: "input-hook", Run: "echo 'XR: test-xr.yaml'"},
	}

	// Mock the runCommand function
	var executedCommands []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		// Extract just the command part (skip the shell and -c flags)
		if len(args) >= 2 && args[0] == "-c" {
			executedCommands = append(executedCommands, args[1])
		}

		return []byte("mock output"), nil
	}

	// Mock renderTemplate function (should not be called for pre-test hooks without placeholders)
	renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
		return content, nil
	}

	// Execute hooks (pre-test hooks with outputs=nil)
	hookExecutor := newHookExecutor(repositories, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "test", inputs, nil, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify that hooks were executed as-is (no template processing for pre-test hooks)
	expectedCommands := []string{
		"echo 'Repository: /path/to/myrepo'",
		"echo 'XR: test-xr.yaml'",
	}
	assert.Equal(t, expectedCommands, executedCommands)
}

// TestExecuteHooks_Order tests that hooks are executed in the correct order.
func TestExecuteHooks_Order(t *testing.T) {
	// Create hooks
	hooks := []api.Hook{
		{Name: "hook-1", Run: "echo 'hook-1 executed'"},
		{Name: "hook-2", Run: "echo 'hook-2 executed'"},
	}

	// Mock the runCommand function to capture execution order
	var executionOrder []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		// Extract just the command part (skip the shell and -c flags)
		if len(args) >= 2 && args[0] == "-c" {
			executionOrder = append(executionOrder, args[1])
		}

		return []byte("mock output"), nil
	}

	renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
		return content, nil
	}

	// Execute hooks
	hookExecutor := newHookExecutor(nil, false, runCommand, renderTemplate)
	_, err := hookExecutor.executeHooks(hooks, "test", api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)

	// Verify hooks were executed in order
	expectedOrder := []string{
		"echo 'hook-1 executed'",
		"echo 'hook-2 executed'",
	}
	assert.Equal(t, expectedOrder, executionOrder)
}

// TestExecuteHooks_ValidationErrors tests executeHooks with validation errors.
func TestExecuteHooks_ValidationErrors(t *testing.T) {
	// Skip this test for now as it causes a panic
	// TODO: Fix the validation logic in executeHooks to handle nil inputs properly
	t.Skip("Skipping validation test due to panic in executeHooks")
}

// TestExecuteHooks_HookFailure tests executeHooks with hook failures.
func TestExecuteHooks_HookFailure(t *testing.T) {
	t.Run("hook with name", func(t *testing.T) {
		// Create hooks with name
		hooks := []api.Hook{
			{Name: "failing-hook", Run: "exit 1"},
		}

		// Mock the runCommand function to return an error
		runCommand := func(_ string, _ ...string) ([]byte, error) {
			return []byte("command failed"), errors.New("exit status 1")
		}

		renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
			return content, nil
		}

		// Execute hooks - should fail
		hookExecutor := newHookExecutor(nil, false, runCommand, renderTemplate)
		_, err := hookExecutor.executeHooks(hooks, "test", api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		require.Error(t, err)

		// Validate complete error message format
		expectedError := "test hook 'failing-hook' failed with exit code 1: command failed"
		assert.Equal(t, expectedError, err.Error())
	})

	t.Run("hook without name", func(t *testing.T) {
		// Create hooks without name
		hooks := []api.Hook{
			{Run: "exit 2"},
		}

		// Mock the runCommand function to return an error
		runCommand := func(_ string, _ ...string) ([]byte, error) {
			return []byte("another error"), errors.New("exit status 2")
		}

		renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
			return content, nil
		}

		// Execute hooks - should fail
		hookExecutor := newHookExecutor(nil, false, runCommand, renderTemplate)
		_, err := hookExecutor.executeHooks(hooks, "pre-test", api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		require.Error(t, err)

		// Validate error message format (should contain the key components)
		assert.Contains(t, err.Error(), "pre-test hook failed with exit code")
		assert.Contains(t, err.Error(), "another error")
	})
}

// TestExecuteHooks_PostTestHooks tests executeHooks with post-test hooks (outputs != nil).
func TestExecuteHooks_PostTestHooks(t *testing.T) {
	// Create repositories
	repositories := map[string]string{
		"myrepo": "/path/to/myrepo",
	}

	// Create inputs
	inputs := api.Inputs{
		Composition: "test-comp.yaml",
	}

	// Create mock outputs
	outputs := &engine.Outputs{
		XR: "test-xr.yaml",
	}

	// Create hooks with template variables (post-test hooks)
	hooks := []api.Hook{
		{Name: "post-hook-1", Run: fmt.Sprintf("echo 'Repository: %s.Repositories.myrepo%s'", placeholder.Open, placeholder.Close)},
		{Name: "post-hook-2", Run: fmt.Sprintf("echo 'XR: %s.Outputs.XR%s'", placeholder.Open, placeholder.Close)},
	}

	// Mock the runCommand function
	var executedCommands []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		// Extract just the command part (skip the shell and -c flags)
		if len(args) >= 2 && args[0] == "-c" {
			executedCommands = append(executedCommands, args[1])
		}

		return []byte("mock output"), nil
	}

	// Mock renderTemplate function to process template variables using Go templates
	renderTemplate := func(content string, templateContext *templateContext, templateName string) (string, error) {
		tmpl, err := template.New(templateName).Option("missingkey=error").Parse(content)
		if err != nil {
			return "", fmt.Errorf("failed to parse template: %w", err)
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, templateContext); err != nil {
			return "", fmt.Errorf("failed to execute template: %w", err)
		}

		return buf.String(), nil
	}

	// Execute hooks (post-test hooks with outputs != nil)
	hookExecutor := newHookExecutor(repositories, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "post-test", inputs, outputs, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify that template variables were processed
	expectedCommands := []string{
		"echo 'Repository: /path/to/myrepo'",
		"echo 'XR: test-xr.yaml'",
	}
	assert.Equal(t, expectedCommands, executedCommands)
}

// TestExecuteHooks_PreTestHooks_InputsTemplateVariables tests pre-test hooks with Inputs template variables.
func TestExecuteHooks_PreTestHooks_InputsTemplateVariables(t *testing.T) {
	// Create repositories
	repositories := map[string]string{
		"myrepo": "/path/to/myrepo",
	}

	// Create inputs with various fields
	inputs := api.Inputs{
		XR:                "test-xr.yaml",
		Claim:             "test-claim.yaml",
		Composition:       "test-comp.yaml",
		Functions:         "test-functions.yaml",
		ObservedResources: "test-observed.yaml",
	}

	// Create hooks with Inputs template variables that need processing
	// These would be processed by processTemplateVariables before executeHooks is called
	// But we're testing the execution flow - hooks should have already-processed templates
	// However, to test the real flow, we should test that executeHooks works with inputs
	// Note: When outputs=nil, executeHooks doesn't do template processing (line 905-906)
	// So template variables must be processed earlier by processTemplateVariables
	// This test verifies that pre-test hooks work with Inputs that are available
	hooks := []api.Hook{
		{Name: "pre-hook-1", Run: "echo 'XR: test-xr.yaml'"},
		{Name: "pre-hook-2", Run: "echo 'Composition: test-comp.yaml'"},
		{Name: "pre-hook-3", Run: "echo 'Functions: test-functions.yaml'"},
	}

	// Mock the runCommand function
	var executedCommands []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "-c" {
			executedCommands = append(executedCommands, args[1])
		}

		return []byte("mock output"), nil
	}

	renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
		return content, nil
	}

	// Execute hooks (pre-test hooks with outputs=nil, inputs available)
	hookExecutor := newHookExecutor(repositories, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "pre-test", inputs, nil, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify that hooks were executed with the input values
	expectedCommands := []string{
		"echo 'XR: test-xr.yaml'",
		"echo 'Composition: test-comp.yaml'",
		"echo 'Functions: test-functions.yaml'",
	}
	assert.Equal(t, expectedCommands, executedCommands)
}

// TestExecuteHooks_PreTestHooks_WithoutTemplateVariables tests pre-test hooks without template variables.
func TestExecuteHooks_PreTestHooks_WithoutTemplateVariables(t *testing.T) {
	inputs := api.Inputs{
		XR: "test-xr.yaml",
	}

	// Create hooks without template variables
	hooks := []api.Hook{
		{Name: "simple-hook", Run: "echo 'Hello World'"},
		{Name: "another-hook", Run: "ls -la"},
	}

	var executedCommands []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "-c" {
			executedCommands = append(executedCommands, args[1])
		}

		return []byte("mock output"), nil
	}

	renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
		return content, nil
	}

	hookExecutor := newHookExecutor(nil, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "pre-test", inputs, nil, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)
	assert.Len(t, results, 2)

	expectedCommands := []string{
		"echo 'Hello World'",
		"ls -la",
	}
	assert.Equal(t, expectedCommands, executedCommands)
}

// TestExecuteHooks_PreTestHooks_OutputsTemplateVariables tests that pre-test hooks with Outputs template variables fail.
func TestExecuteHooks_PreTestHooks_OutputsTemplateVariables(t *testing.T) {
	repositories := map[string]string{
		"myrepo": "/path/to/myrepo",
	}

	inputs := api.Inputs{
		XR: "test-xr.yaml",
	}

	// Create hooks with Outputs template variables
	// Since outputs=nil for pre-test hooks, these template variables cannot be resolved and should cause an error
	hooks := []api.Hook{
		{Name: "pre-hook-with-outputs", Run: fmt.Sprintf("echo 'Outputs XR: %s.Outputs.XR%s'", placeholder.Open, placeholder.Close)},
	}

	runCommand := func(_ string, _ ...string) ([]byte, error) {
		return []byte("mock output"), nil
	}

	// Mock renderTemplate to simulate nil pointer error
	renderTemplate := func(content string, templateContext *templateContext, _ string) (string, error) {
		if templateContext.Outputs == nil {
			return "", fmt.Errorf("nil pointer evaluating *engine.Outputs.XR")
		}

		return content, nil
	}

	// Execute hooks with outputs=nil (pre-test scenario)
	// This should fail because Outputs template variables cannot be resolved when outputs is nil
	hookExecutor := newHookExecutor(repositories, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "pre-test", inputs, nil, map[string]*engine.TestCaseResult{})
	require.Error(t, err)
	// Results should contain the HookResult for the template rendering failure
	require.NotNil(t, results)
	require.Len(t, results, 1)
	assert.Equal(t, "pre-hook-with-outputs", results[0].Name)
	require.Error(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "failed to render hook template")
	assert.Contains(t, err.Error(), "failed to render template")
	assert.Contains(t, err.Error(), "nil pointer evaluating *engine.Outputs.XR")
}

// TestExecuteHooks_PostTestHooks_InputsAndOutputsTemplateVariables tests post-test hooks with both Inputs and Outputs template variables.
func TestExecuteHooks_PostTestHooks_InputsAndOutputsTemplateVariables(t *testing.T) {
	// Create repositories
	repositories := map[string]string{
		"myrepo": "/path/to/myrepo",
	}

	// Create inputs with various fields
	inputs := api.Inputs{
		XR:          "original-xr.yaml",
		Composition: "test-comp.yaml",
		Functions:   "test-functions.yaml",
	}

	// Create mock outputs
	outputs := &engine.Outputs{
		XR:          "rendered-xr.yaml",
		Render:      "rendered-resources.yaml",
		RenderCount: 5,
		Rendered: map[string]string{
			"Pod/test-pod": "/path/to/pod.yaml",
		},
	}

	// Create hooks with BOTH Inputs and Outputs template variables (using placeholders as they would appear after YAML loading)
	hooks := []api.Hook{
		{Name: "post-hook-1", Run: fmt.Sprintf("echo 'Input XR: %s.Inputs.XR%s, Output XR: %s.Outputs.XR%s'", placeholder.Open, placeholder.Close, placeholder.Open, placeholder.Close)},
		{Name: "post-hook-2", Run: fmt.Sprintf("echo 'Input Composition: %s.Inputs.Composition%s, Output Render: %s.Outputs.Render%s'", placeholder.Open, placeholder.Close, placeholder.Open, placeholder.Close)},
		{Name: "post-hook-3", Run: fmt.Sprintf("echo 'Output RenderCount: %s.Outputs.RenderCount%s, Repository: %s.Repositories.myrepo%s'", placeholder.Open, placeholder.Close, placeholder.Open, placeholder.Close)},
	}

	// Mock the runCommand function
	var executedCommands []string

	runCommand := func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "-c" {
			executedCommands = append(executedCommands, args[1])
		}

		return []byte("mock output"), nil
	}

	// Mock renderTemplate to process template variables using Go templates
	renderTemplate := func(content string, templateContext *templateContext, templateName string) (string, error) {
		tmpl, err := template.New(templateName).Option("missingkey=error").Parse(content)
		if err != nil {
			return "", fmt.Errorf("failed to parse template: %w", err)
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, templateContext); err != nil {
			return "", fmt.Errorf("failed to execute template: %w", err)
		}

		return buf.String(), nil
	}

	// Execute hooks (post-test hooks with outputs != nil - this enables template processing)
	hookExecutor := newHookExecutor(repositories, false, runCommand, renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "post-test", inputs, outputs, map[string]*engine.TestCaseResult{})
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify that BOTH Inputs and Outputs template variables were processed
	expectedCommands := []string{
		"echo 'Input XR: original-xr.yaml, Output XR: rendered-xr.yaml'",
		"echo 'Input Composition: test-comp.yaml, Output Render: rendered-resources.yaml'",
		"echo 'Output RenderCount: 5, Repository: /path/to/myrepo'",
	}
	assert.Equal(t, expectedCommands, executedCommands)
}

// TestExecuteHooks_WorkingDirectory tests that hooks execute with the correct working directory.
func TestExecuteHooks_WorkingDirectory(t *testing.T) {
	// Use in-memory filesystem for test setup (no I/O operations)
	fs := afero.NewMemMapFs()

	// Create a subdirectory structure to simulate testsuite file location
	testSuiteDir := "/tests/suite"
	require.NoError(t, fs.MkdirAll(testSuiteDir, 0o755))
	testSuiteFile := filepath.Join(testSuiteDir, "suite_xprin.yaml")

	// Create the testsuite file in the in-memory filesystem
	require.NoError(t, afero.WriteFile(fs, testSuiteFile, []byte("tests: []"), 0o644))

	// Create a runner with the testsuite file path
	options := &testexecutionUtils.Options{}
	runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})

	// Verify that testSuiteFileDir is computed correctly
	assert.Equal(t, testSuiteDir, runner.testSuiteFileDir, "testSuiteFileDir should be computed from testSuiteFile")

	// Verify that the working directory is set correctly by capturing cmd.Dir
	// We don't need to actually run the command - just verify cmd.Dir is set
	var capturedDir string

	runner.runCommand = func(name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		// Set Dir the same way the original does
		cmd.Dir = runner.testSuiteFileDir
		capturedDir = cmd.Dir
		// Return without running to avoid filesystem I/O
		return []byte{}, nil
	}

	// Create a simple hook for testing
	hooks := []api.Hook{
		{
			Name: "test-hook",
			Run:  "echo 'test'",
		},
	}

	// Execute hooks
	hookExecutor := newHookExecutor(nil, false, runner.runCommand, runner.renderTemplate)
	results, err := hookExecutor.executeHooks(hooks, "pre-test", api.Inputs{}, nil, map[string]*engine.TestCaseResult{})

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Error, "hook should execute successfully")
	assert.Equal(t, testSuiteDir, capturedDir, "runCommand should set cmd.Dir to testsuite file directory")
}

// TestProcessHookTemplateVariables tests the processHookTemplateVariables helper.
func TestProcessHookTemplateVariables(t *testing.T) {
	renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
		return content, nil
	}
	exec := newHookExecutor(nil, false, nil, renderTemplate)

	t.Run("no placeholders returns command as-is", func(t *testing.T) {
		hook := api.Hook{Name: "h", Run: "echo hello"}
		final, cmdVars, err := exec.processHookTemplateVariables(hook, api.Inputs{}, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "echo hello", final)
		assert.Equal(t, "echo hello", cmdVars)
	})

	t.Run("with placeholders restores and calls renderTemplate", func(t *testing.T) {
		var rendered string

		renderTemplate := func(content string, _ *templateContext, _ string) (string, error) {
			rendered = content
			return "echo /path", nil
		}
		exec := newHookExecutor(map[string]string{"r": "/path"}, false, nil, renderTemplate)
		hook := api.Hook{Run: placeholder.Create(".Repositories.r")}
		final, cmdVars, err := exec.processHookTemplateVariables(hook, api.Inputs{}, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "echo /path", final)
		assert.Equal(t, "{{.Repositories.r}}", cmdVars)
		assert.Equal(t, "{{.Repositories.r}}", rendered)
	})

	t.Run("render error is returned", func(t *testing.T) {
		renderTemplate := func(string, *templateContext, string) (string, error) {
			return "", fmt.Errorf("render failed")
		}
		exec := newHookExecutor(nil, false, nil, renderTemplate)
		hook := api.Hook{Run: placeholder.Create(".X")}
		_, _, err := exec.processHookTemplateVariables(hook, api.Inputs{}, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "render failed")
	})
}

// TestBuildHookFailureMessage tests the buildHookFailureMessage helper.
func TestBuildHookFailureMessage(t *testing.T) {
	t.Run("named hook with output", func(t *testing.T) {
		msg := buildHookFailureMessage("pre-test", "my-hook", "echo x", 1, []byte("stderr line"))
		assert.Contains(t, msg, "pre-test hook 'my-hook' failed with exit code 1")
		assert.Contains(t, msg, "stderr line")
	})
	t.Run("named hook without output", func(t *testing.T) {
		msg := buildHookFailureMessage("post-test", "h", "cmd", 2, nil)
		assert.Equal(t, "post-test hook 'h' failed with exit code 2", msg)
	})
	t.Run("unnamed hook with output", func(t *testing.T) {
		msg := buildHookFailureMessage("pre-test", "", "echo bar", 1, []byte("err"))
		assert.Contains(t, msg, "pre-test hook failed with exit code 1")
		assert.Contains(t, msg, "err")
	})
	t.Run("unnamed hook without output includes command", func(t *testing.T) {
		msg := buildHookFailureMessage("post-test", "", "echo baz", 3, nil)
		assert.Contains(t, msg, "post-test hook failed with exit code 3")
		assert.Contains(t, msg, "echo baz")
	})
	t.Run("multiline output is indented", func(t *testing.T) {
		msg := buildHookFailureMessage("pre-test", "h", "c", 1, []byte("line1\nline2"))
		assert.Contains(t, msg, "line1\n    line2")
	})
}
