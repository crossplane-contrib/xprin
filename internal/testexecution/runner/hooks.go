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
	runCommand     func(name string, args ...string) ([]byte, []byte, error)
	renderTemplate func(content string, templateContext *templateContext, templateName string) (string, error)
}

// newHookExecutor creates a new hook executor.
func newHookExecutor(
	repositories map[string]string,
	debug bool,
	runCommand func(name string, args ...string) ([]byte, []byte, error),
	renderTemplate func(content string, templateContext *templateContext, templateName string) (string, error),
) *hookExecutor {
	return &hookExecutor{
		repositories:   repositories,
		debug:          debug,
		runCommand:     runCommand,
		renderTemplate: renderTemplate,
	}
}

// executeHooks executes a list of hook commands and returns the combined output.
func (e *hookExecutor) executeHooks(hooks []api.Hook, hookType string, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) ([]engine.HookResult, error) {
	hookResults := make([]engine.HookResult, 0, len(hooks))

	for _, hook := range hooks {
		var err error

		finalCommand := hook.Run

		// Restore template variables in hook command and re-process with current context
		if strings.Contains(hook.Run, testexecutionUtils.PlaceholderOpen) {
			hook.Run = testexecutionUtils.RestoreTemplateVars(hook.Run)

			// Render template with context
			templateContext := newTemplateContext(e.repositories, inputs, outputs, tests)

			finalCommand, err = e.renderTemplate(hook.Run, templateContext, "hook")
			if err != nil {
				return nil, fmt.Errorf("failed to render hook template: %w", err)
			}
		}

		if e.debug {
			if hook.Name != "" {
				utils.DebugPrintf("Executing %s hook '%s': %s\n", hookType, hook.Name, finalCommand)
			} else {
				utils.DebugPrintf("Executing %s hook '%s'\n", hookType, finalCommand)
			}
		}

		stdout, stderr, err := e.runCommand("sh", "-c", finalCommand)

		// Use original hook for the result (to preserve original command in error messages)
		hookResult := engine.NewHookResult(hook.Name, hook.Run, stdout, stderr, err)
		hookResults = append(hookResults, hookResult)

		if err != nil {
			// Create error message with command, stderr, and exit code
			var errorMsg string

			stderrStr := strings.TrimSpace(string(stderr))
			// Indent multiline stderr output for better readability
			if strings.Contains(stderrStr, "\n") {
				stderrStr = strings.ReplaceAll(stderrStr, "\n", "\n    ")
			}

			// Get exit code from error
			exitCode := 1 // default

			exitError := &exec.ExitError{}
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			}

			if hook.Name != "" {
				errorMsg = fmt.Sprintf("%s hook '%s' failed: %s: %s: exit code %d", hookType, hook.Name, hook.Run, stderrStr, exitCode)
			} else {
				errorMsg = fmt.Sprintf("%s hook failed: %s: %s: exit code %d", hookType, hook.Run, stderrStr, exitCode)
			}

			return hookResults, errors.New(errorMsg)
		}
	}

	return hookResults, nil
}
