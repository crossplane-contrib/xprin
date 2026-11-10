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
	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
)

// debugPrintPatches prints patches in a consistent format.
func (r *Runner) debugPrintPatches(patches api.Patches) {
	if patches.HasPatches() {
		r.debug.Printf("  Patches:\n")

		if patches.XRD != "" {
			r.debug.Printf("  - XRD: %s\n", patches.XRD)
		}

		if patches.ConnectionSecret != nil {
			r.debug.Printf("  - Connection Secret: %t\n", *patches.ConnectionSecret)
		}

		if patches.ConnectionSecretName != "" {
			r.debug.Printf("  - Connection Secret Name: %s\n", patches.ConnectionSecretName)
		}

		if patches.ConnectionSecretNamespace != "" {
			r.debug.Printf("  - Connection Secret Namespace: %s\n", patches.ConnectionSecretNamespace)
		}
	}
}

// debugPrintInputs prints the content of inputs with the header.
func (r *Runner) debugPrintInputs(inputs api.Inputs) {
	r.debug.Printf("  Inputs:\n")

	if inputs.XR != "" {
		r.debug.Printf("  - XR: %s\n", inputs.XR)
	}

	if inputs.Claim != "" {
		r.debug.Printf("  - Claim: %s\n", inputs.Claim)
	}

	if inputs.Composition != "" {
		r.debug.Printf("  - Composition: %s\n", inputs.Composition)
	}

	if inputs.Functions != "" {
		r.debug.Printf("  - Functions: %s\n", inputs.Functions)
	}

	if len(inputs.CRDs) > 0 {
		r.debug.Printf("  - CRDs:\n")

		for _, crd := range inputs.CRDs {
			r.debug.Printf("    - %s\n", crd)
		}
	}

	if len(inputs.ContextFiles) > 0 {
		r.debug.Printf("  - Context Files:\n")

		for contextKey, contextFile := range inputs.ContextFiles {
			r.debug.Printf("      %s: %s\n", contextKey, contextFile)
		}
	}

	if len(inputs.ContextValues) > 0 {
		r.debug.Printf("  - Context Values:\n")

		for contextKey, contextValue := range inputs.ContextValues {
			r.debug.Printf("      %s: %s\n", contextKey, contextValue)
		}
	}

	if inputs.ObservedResources != "" {
		r.debug.Printf("  - Observed Resources: %s\n", inputs.ObservedResources)
	}

	if inputs.ExtraResources != "" {
		r.debug.Printf("  - Extra Resources: %s\n", inputs.ExtraResources)
	}

	if inputs.FunctionCredentials != "" {
		r.debug.Printf("  - Function Credentials: %s\n", inputs.FunctionCredentials)
	}
}

// debugPrintHooks prints hooks in a consistent format.
func (r *Runner) debugPrintHooks(hooks api.Hooks) {
	if hooks.HasHooks() {
		r.debug.Printf("  Hooks:\n")

		if hooks.HasPreTestHooks() {
			r.debug.Printf("    Pre-Test:\n")

			for _, hook := range hooks.PreTest {
				if hook.Name != "" {
					r.debug.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				r.debug.Printf("      run: %s\n", processedCommand)
			}
		}

		if hooks.HasPostTestHooks() {
			r.debug.Printf("    Post-Test:\n")

			for _, hook := range hooks.PostTest {
				if hook.Name != "" {
					r.debug.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				r.debug.Printf("      run: %s\n", processedCommand)
			}
		}
	}
}

// debugPrintCommon prints debug information for common configuration.
func (r *Runner) debugPrintCommon(common api.Common, header string) {
	r.debug.Printf("%s\n", header)
	r.debugPrintPatches(common.Patches)
	r.debugPrintInputs(common.Inputs)
	r.debugPrintHooks(common.Hooks)
}

// debugPrintTestCase prints debug information for a test case.
func (r *Runner) debugPrintTestCase(testCase api.TestCase, header string) {
	r.debug.Printf("%s\n", header)
	r.debugPrintPatches(testCase.Patches)
	r.debugPrintInputs(testCase.Inputs)
	r.debugPrintHooks(testCase.Hooks)
}
