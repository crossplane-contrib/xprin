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
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/crossplane-contrib/xprin/internal/utils"
)

// debugPrintPatches prints patches in a consistent format.
func (r *Runner) debugPrintPatches(patches api.Patches) {
	if patches.HasPatches() {
		utils.DebugPrintf("  Patches:\n")

		if patches.XRD != "" {
			utils.DebugPrintf("  - XRD: %s\n", patches.XRD)
		}

		if patches.ConnectionSecret != nil {
			utils.DebugPrintf("  - Connection Secret: %t\n", *patches.ConnectionSecret)
		}

		if patches.ConnectionSecretName != "" {
			utils.DebugPrintf("  - Connection Secret Name: %s\n", patches.ConnectionSecretName)
		}

		if patches.ConnectionSecretNamespace != "" {
			utils.DebugPrintf("  - Connection Secret Namespace: %s\n", patches.ConnectionSecretNamespace)
		}
	}
}

// debugPrintInputs prints the content of inputs with the header.
func (r *Runner) debugPrintInputs(inputs api.Inputs) {
	utils.DebugPrintf("  Inputs:\n")

	if inputs.XR != "" {
		utils.DebugPrintf("  - XR: %s\n", inputs.XR)
	}

	if inputs.Claim != "" {
		utils.DebugPrintf("  - Claim: %s\n", inputs.Claim)
	}

	if inputs.Composition != "" {
		utils.DebugPrintf("  - Composition: %s\n", inputs.Composition)
	}

	if inputs.Functions != "" {
		utils.DebugPrintf("  - Functions: %s\n", inputs.Functions)
	}

	if len(inputs.CRDs) > 0 {
		utils.DebugPrintf("  - CRDs:\n")

		for _, crd := range inputs.CRDs {
			utils.DebugPrintf("    - %s\n", crd)
		}
	}

	if len(inputs.ContextFiles) > 0 {
		utils.DebugPrintf("  - Context Files:\n")

		for contextKey, contextFile := range inputs.ContextFiles {
			utils.DebugPrintf("      %s: %s\n", contextKey, contextFile)
		}
	}

	if len(inputs.ContextValues) > 0 {
		utils.DebugPrintf("  - Context Values:\n")

		for contextKey, contextValue := range inputs.ContextValues {
			utils.DebugPrintf("      %s: %s\n", contextKey, contextValue)
		}
	}

	if inputs.ObservedResources != "" {
		utils.DebugPrintf("  - Observed Resources: %s\n", inputs.ObservedResources)
	}

	if inputs.ExtraResources != "" {
		utils.DebugPrintf("  - Extra Resources: %s\n", inputs.ExtraResources)
	}

	if inputs.FunctionCredentials != "" {
		utils.DebugPrintf("  - Function Credentials: %s\n", inputs.FunctionCredentials)
	}
}

// debugPrintHooks prints hooks in a consistent format.
func (r *Runner) debugPrintHooks(hooks api.Hooks) {
	if hooks.HasHooks() {
		utils.DebugPrintf("  Hooks:\n")

		if hooks.HasPreTestHooks() {
			utils.DebugPrintf("    Pre-Test:\n")

			for _, hook := range hooks.PreTest {
				if hook.Name != "" {
					utils.DebugPrintf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := testexecutionUtils.RestoreTemplateVars(hook.Run)
				utils.DebugPrintf("      run: %s\n", processedCommand)
			}
		}

		if hooks.HasPostTestHooks() {
			utils.DebugPrintf("    Post-Test:\n")

			for _, hook := range hooks.PostTest {
				if hook.Name != "" {
					utils.DebugPrintf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := testexecutionUtils.RestoreTemplateVars(hook.Run)
				utils.DebugPrintf("      run: %s\n", processedCommand)
			}
		}
	}
}

// debugPrintCommon prints debug information for common configuration.
func (r *Runner) debugPrintCommon(common api.Common, header string) {
	utils.DebugPrintf("%s\n", header)
	r.debugPrintPatches(common.Patches)
	r.debugPrintInputs(common.Inputs)
	r.debugPrintHooks(common.Hooks)
}

// debugPrintTestCase prints debug information for a test case.
func (r *Runner) debugPrintTestCase(testCase api.TestCase, header string) {
	utils.DebugPrintf("%s\n", header)
	r.debugPrintPatches(testCase.Patches)
	r.debugPrintInputs(testCase.Inputs)
	r.debugPrintHooks(testCase.Hooks)
}
