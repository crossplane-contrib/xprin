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
	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
)

// debugPrintPatches prints patches in a consistent format.
func (s *Session) debugPrintPatches(patches api.Patches) {
	if patches.HasPatches() {
		s.debug.Printf("  Patches:\n")

		if patches.XRD != "" {
			s.debug.Printf("  - XRD: %s\n", patches.XRD)
		}

		if patches.ConnectionSecret != nil {
			s.debug.Printf("  - Connection Secret: %t\n", *patches.ConnectionSecret)
		}

		if patches.ConnectionSecretName != "" {
			s.debug.Printf("  - Connection Secret Name: %s\n", patches.ConnectionSecretName)
		}

		if patches.ConnectionSecretNamespace != "" {
			s.debug.Printf("  - Connection Secret Namespace: %s\n", patches.ConnectionSecretNamespace)
		}
	}
}

// debugPrintInputs prints the content of inputs with the header.
func (s *Session) debugPrintInputs(inputs api.Inputs) {
	s.debug.Printf("  Inputs:\n")

	if inputs.XR != "" {
		s.debug.Printf("  - XR: %s\n", inputs.XR)
	}

	if inputs.Claim != "" {
		s.debug.Printf("  - Claim: %s\n", inputs.Claim)
	}

	if inputs.Composition != "" {
		s.debug.Printf("  - Composition: %s\n", inputs.Composition)
	}

	if inputs.Functions != "" {
		s.debug.Printf("  - Functions: %s\n", inputs.Functions)
	}

	if len(inputs.CRDs) > 0 {
		s.debug.Printf("  - CRDs:\n")

		for _, crd := range inputs.CRDs {
			s.debug.Printf("    - %s\n", crd)
		}
	}

	if len(inputs.ContextFiles) > 0 {
		s.debug.Printf("  - Context Files:\n")

		for contextKey, contextFile := range inputs.ContextFiles {
			s.debug.Printf("      %s: %s\n", contextKey, contextFile)
		}
	}

	if len(inputs.ContextValues) > 0 {
		s.debug.Printf("  - Context Values:\n")

		for contextKey, contextValue := range inputs.ContextValues {
			s.debug.Printf("      %s: %s\n", contextKey, contextValue)
		}
	}

	if inputs.ObservedResources != "" {
		s.debug.Printf("  - Observed Resources: %s\n", inputs.ObservedResources)
	}

	if inputs.ExtraResources != "" {
		s.debug.Printf("  - Extra Resources: %s\n", inputs.ExtraResources)
	}

	if inputs.FunctionCredentials != "" {
		s.debug.Printf("  - Function Credentials: %s\n", inputs.FunctionCredentials)
	}
}

// debugPrintHooks prints hooks in a consistent format.
func (s *Session) debugPrintHooks(hooks api.Hooks) {
	if hooks.HasHooks() {
		s.debug.Printf("  Hooks:\n")

		if hooks.HasPreTestHooks() {
			s.debug.Printf("    Pre-Test:\n")

			for _, hook := range hooks.PreTest {
				if hook.Name != "" {
					s.debug.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				s.debug.Printf("      run: %s\n", processedCommand)
			}
		}

		if hooks.HasPostTestHooks() {
			s.debug.Printf("    Post-Test:\n")

			for _, hook := range hooks.PostTest {
				if hook.Name != "" {
					s.debug.Printf("    - name: %s\n", hook.Name)
				}
				// Show processed hook command instead of raw with placeholders
				processedCommand := placeholder.Restore(hook.Run)
				s.debug.Printf("      run: %s\n", processedCommand)
			}
		}
	}
}

// debugPrintCommon prints debug information for common configuration.
func (s *Session) debugPrintCommon(common api.Common, header string) {
	s.debug.Printf("%s\n", header)
	s.debugPrintPatches(common.Patches)
	s.debugPrintInputs(common.Inputs)
	s.debugPrintHooks(common.Hooks)
}
