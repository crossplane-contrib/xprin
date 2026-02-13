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
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/crossplane-contrib/xprin/internal/utils"
)

// hookExecutor handles execution of hooks.
type hookExecutor struct {
	repositories   map[string]string
	debug          bool
	runCommand     func(name string, args ...string) ([]byte, error)
	renderTemplate func(content string, templateContext *templateContext, templateName string) (string, error)
}

// newHookExecutor creates a new hook executor.
func newHookExecutor(
	repositories map[string]string,
	debug bool,
	runCommand func(name string, args ...string) ([]byte, error),
	renderTemplate func(content string, templateContext *templateContext, templateName string) (string, error),
) *hookExecutor {
	return &hookExecutor{
		repositories:   repositories,
		debug:          debug,
		runCommand:     runCommand,
		renderTemplate: renderTemplate,
	}
}

// processHookTemplateVariables converts the hook command (possibly with placeholders) into the executable form
// and the form to store in HookResult. The three command forms are:
//   - command with placeholders: hook.Run as in spec (e.g. __OPEN__.Repositories.myrepo__CLOSE__)
//   - command with template vars: restored form (e.g. {{ .Repositories.myrepo }}) — stored in HookResult
//   - final command: template vars rendered (e.g. /path/to/repo) — what we execute
func (e *hookExecutor) processHookTemplateVariables(hook api.Hook, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) (finalCommand, commandWithTemplateVars string, err error) {
	if !strings.Contains(hook.Run, testexecutionUtils.PlaceholderOpen) {
		return hook.Run, hook.Run, nil
	}

	commandWithTemplateVars = testexecutionUtils.RestoreTemplateVars(hook.Run)
	context := newTemplateContext(e.repositories, inputs, outputs, tests)

	finalCommand, err = e.renderTemplate(commandWithTemplateVars, context, "hook")
	if err != nil {
		return "", "", err
	}

	return finalCommand, commandWithTemplateVars, nil
}

// buildHookFailureMessage builds the error message for a failed hook (exit code, optional output, hook name/command).
func buildHookFailureMessage(hookType, hookName, commandWithTemplateVars string, exitCode int, output []byte) string {
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "\n") {
		outputStr = strings.ReplaceAll(outputStr, "\n", "\n    ")
	}

	if hookName != "" {
		msg := fmt.Sprintf("%s hook '%s' failed with exit code %d", hookType, hookName, exitCode)
		if outputStr != "" {
			return fmt.Sprintf("%s: %s", msg, outputStr)
		}

		return msg
	}

	msg := fmt.Sprintf("%s hook failed with exit code %d", hookType, exitCode)
	if outputStr != "" {
		return fmt.Sprintf("%s: %s", msg, outputStr)
	}

	return fmt.Sprintf("%s: %s", msg, commandWithTemplateVars)
}

// executeHook runs a single hook: prepare command (processHookTemplateVariables), run, return result. On template or run error returns the HookResult (for the failed hook) and a non-nil error.
func (e *hookExecutor) executeHook(hook api.Hook, hookType string, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) (engine.HookResult, error) {
	finalCommand, commandWithTemplateVars, err := e.processHookTemplateVariables(hook, inputs, outputs, tests)
	if err != nil {
		templateErr := fmt.Errorf("failed to render hook template: %w", err)

		commandWithTemplateVarsForResult := hook.Run
		if strings.Contains(hook.Run, testexecutionUtils.PlaceholderOpen) {
			commandWithTemplateVarsForResult = testexecutionUtils.RestoreTemplateVars(hook.Run)
		}

		hookResult := engine.NewHookResult(hook.Name, commandWithTemplateVarsForResult, nil, templateErr)

		var errorMsg string
		if hook.Name != "" {
			errorMsg = fmt.Sprintf("%s hook '%s' failed to render template: %s: %v", hookType, hook.Name, hook.Run, err)
		} else {
			errorMsg = fmt.Sprintf("%s hook failed to render template: %s: %v", hookType, hook.Run, err)
		}

		return hookResult, errors.New(errorMsg)
	}

	if e.debug {
		if hook.Name != "" {
			utils.DebugPrintf("Executing %s hook '%s': %s\n", hookType, hook.Name, finalCommand)
		} else {
			utils.DebugPrintf("Executing %s hook '%s'\n", hookType, finalCommand)
		}
	}

	output, err := e.runCommand("sh", "-c", finalCommand)

	hookResult := engine.NewHookResult(hook.Name, commandWithTemplateVars, output, err)
	if err != nil {
		exitCode := 1

		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}

		errorMsg := buildHookFailureMessage(hookType, hook.Name, commandWithTemplateVars, exitCode, output)

		return hookResult, errors.New(errorMsg)
	}

	return hookResult, nil
}

// executeHooks runs each hook in order via executeHook; on template or run error returns with results so far and a formatted error.
func (e *hookExecutor) executeHooks(hooks []api.Hook, hookType string, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) ([]engine.HookResult, error) {
	hookResults := make([]engine.HookResult, 0, len(hooks))
	for _, hook := range hooks {
		result, err := e.executeHook(hook, hookType, inputs, outputs, tests)

		hookResults = append(hookResults, result)
		if err != nil {
			return hookResults, err
		}
	}

	return hookResults, nil
}
