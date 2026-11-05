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
	"io"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/crossplane-contrib/xprin/internal/utils"
)

// debugLog is where the --debug lines of a run go: to a writer, which is stderr, or the buffer of a test case that runs
// at the same time as others, whose lines are then printed with its result so that they are not mixed with theirs.
type debugLog struct {
	w io.Writer
}

// printf prints a --debug line.
func (d *debugLog) printf(format string, args ...any) {
	utils.DebugFprintf(d.w, format, args...)
}

// debugf is the function that the hooks print their --debug lines with: nil without --debug.
func (r *Runner) debugf() func(format string, args ...any) {
	if !r.Debug {
		return nil
	}

	return r.debug.printf
}

// debugPrintPatches prints patches in a consistent format.
func (r *Runner) debugPrintPatches(patches api.Patches) {
	if patches.HasPatches() {
		r.debug.printf("  Patches:\n")

		if patches.XRD != "" {
			r.debug.printf("  - XRD: %s\n", patches.XRD)
		}

		if patches.ConnectionSecret != nil {
			r.debug.printf("  - Connection Secret: %t\n", *patches.ConnectionSecret)
		}

		if patches.ConnectionSecretName != "" {
			r.debug.printf("  - Connection Secret Name: %s\n", patches.ConnectionSecretName)
		}

		if patches.ConnectionSecretNamespace != "" {
			r.debug.printf("  - Connection Secret Namespace: %s\n", patches.ConnectionSecretNamespace)
		}
	}
}

// debugPrintInputs prints the content of inputs with the header.
func (r *Runner) debugPrintInputs(inputs api.Inputs) {
	r.debug.printf("  Inputs:\n")

	if inputs.XR != "" {
		r.debug.printf("  - XR: %s\n", inputs.XR)
	}

	if inputs.Claim != "" {
		r.debug.printf("  - Claim: %s\n", inputs.Claim)
	}

	if inputs.Composition != "" {
		r.debug.printf("  - Composition: %s\n", inputs.Composition)
	}

	if inputs.Functions != "" {
		r.debug.printf("  - Functions: %s\n", inputs.Functions)
	}

	if len(inputs.CRDs) > 0 {
		r.debug.printf("  - CRDs:\n")

		for _, crd := range inputs.CRDs {
			r.debug.printf("    - %s\n", crd)
		}
	}

	if len(inputs.ContextFiles) > 0 {
		r.debug.printf("  - Context Files:\n")

		for contextKey, contextFile := range inputs.ContextFiles {
			r.debug.printf("      %s: %s\n", contextKey, contextFile)
		}
	}

	if len(inputs.ContextValues) > 0 {
		r.debug.printf("  - Context Values:\n")

		for contextKey, contextValue := range inputs.ContextValues {
			r.debug.printf("      %s: %s\n", contextKey, contextValue)
		}
	}

	if inputs.ObservedResources != "" {
		r.debug.printf("  - Observed Resources: %s\n", inputs.ObservedResources)
	}

	if inputs.ExtraResources != "" {
		r.debug.printf("  - Extra Resources: %s\n", inputs.ExtraResources)
	}

	if inputs.FunctionCredentials != "" {
		r.debug.printf("  - Function Credentials: %s\n", inputs.FunctionCredentials)
	}
}

// debugPrintHooks prints hooks in a consistent format.
func (r *Runner) debugPrintHooks(hooks api.Hooks) {
	if hooks.HasHooks() {
		r.debug.printf("  Hooks:\n")

		if hooks.HasPreTestHooks() {
			r.debug.printf("    Pre-Test:\n")

			for _, hook := range hooks.PreTest {
				if hook.Name != "" {
					r.debug.printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				r.debug.printf("      run: %s\n", processedCommand)
			}
		}

		if hooks.HasPostTestHooks() {
			r.debug.printf("    Post-Test:\n")

			for _, hook := range hooks.PostTest {
				if hook.Name != "" {
					r.debug.printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				r.debug.printf("      run: %s\n", processedCommand)
			}
		}
	}
}

// debugPrintCommon prints debug information for common configuration.
func (r *Runner) debugPrintCommon(common api.Common, header string) {
	r.debug.printf("%s\n", header)
	r.debugPrintPatches(common.Patches)
	r.debugPrintInputs(common.Inputs)
	r.debugPrintHooks(common.Hooks)
}

// debugPrintTestCase prints debug information for a test case.
func (r *Runner) debugPrintTestCase(testCase api.TestCase, header string) {
	r.debug.printf("%s\n", header)
	r.debugPrintPatches(testCase.Patches)
	r.debugPrintInputs(testCase.Inputs)
	r.debugPrintHooks(testCase.Hooks)
}
