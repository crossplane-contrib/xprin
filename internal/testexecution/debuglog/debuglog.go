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

// Package debuglog prints, for --debug, the patches, the inputs and the hooks that a test case and the common
// configuration of a testsuite file are both made of. It has no state: the lines go to the logger that it is given.
package debuglog

import (
	"log"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
)

// printPatches prints patches in a consistent format.
func printPatches(logger *log.Logger, patches api.Patches) {
	if patches.HasPatches() {
		logger.Printf("  Patches:\n")

		if patches.XRD != "" {
			logger.Printf("  - XRD: %s\n", patches.XRD)
		}

		if patches.ConnectionSecret != nil {
			logger.Printf("  - Connection Secret: %t\n", *patches.ConnectionSecret)
		}

		if patches.ConnectionSecretName != "" {
			logger.Printf("  - Connection Secret Name: %s\n", patches.ConnectionSecretName)
		}

		if patches.ConnectionSecretNamespace != "" {
			logger.Printf("  - Connection Secret Namespace: %s\n", patches.ConnectionSecretNamespace)
		}
	}
}

// printInputs prints the content of inputs with the header.
func printInputs(logger *log.Logger, inputs api.Inputs) {
	logger.Printf("  Inputs:\n")

	if inputs.XR != "" {
		logger.Printf("  - XR: %s\n", inputs.XR)
	}

	if inputs.Claim != "" {
		logger.Printf("  - Claim: %s\n", inputs.Claim)
	}

	if inputs.Composition != "" {
		logger.Printf("  - Composition: %s\n", inputs.Composition)
	}

	if inputs.Functions != "" {
		logger.Printf("  - Functions: %s\n", inputs.Functions)
	}

	if len(inputs.CRDs) > 0 {
		logger.Printf("  - CRDs:\n")

		for _, crd := range inputs.CRDs {
			logger.Printf("    - %s\n", crd)
		}
	}

	if len(inputs.ContextFiles) > 0 {
		logger.Printf("  - Context Files:\n")

		for contextKey, contextFile := range inputs.ContextFiles {
			logger.Printf("      %s: %s\n", contextKey, contextFile)
		}
	}

	if len(inputs.ContextValues) > 0 {
		logger.Printf("  - Context Values:\n")

		for contextKey, contextValue := range inputs.ContextValues {
			logger.Printf("      %s: %s\n", contextKey, contextValue)
		}
	}

	if inputs.ObservedResources != "" {
		logger.Printf("  - Observed Resources: %s\n", inputs.ObservedResources)
	}

	if inputs.ExtraResources != "" {
		logger.Printf("  - Extra Resources: %s\n", inputs.ExtraResources)
	}

	if inputs.FunctionCredentials != "" {
		logger.Printf("  - Function Credentials: %s\n", inputs.FunctionCredentials)
	}
}

// printHooks prints hooks in a consistent format.
func printHooks(logger *log.Logger, hooks api.Hooks) {
	if hooks.HasHooks() {
		logger.Printf("  Hooks:\n")

		if hooks.HasPreTestHooks() {
			logger.Printf("    Pre-Test:\n")

			for _, hook := range hooks.PreTest {
				if hook.Name != "" {
					logger.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				logger.Printf("      run: %s\n", processedCommand)
			}
		}

		if hooks.HasPostTestHooks() {
			logger.Printf("    Post-Test:\n")

			for _, hook := range hooks.PostTest {
				if hook.Name != "" {
					logger.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				logger.Printf("      run: %s\n", processedCommand)
			}
		}
	}
}

// PrintCommon prints debug information for common configuration.
func PrintCommon(logger *log.Logger, common api.Common, header string) {
	logger.Printf("%s\n", header)
	printPatches(logger, common.Patches)
	printInputs(logger, common.Inputs)
	printHooks(logger, common.Hooks)
}

// PrintTestCase prints debug information for a test case.
func PrintTestCase(logger *log.Logger, testCase api.TestCase, header string) {
	logger.Printf("%s\n", header)
	printPatches(logger, testCase.Patches)
	printInputs(logger, testCase.Inputs)
	printHooks(logger, testCase.Hooks)
}
