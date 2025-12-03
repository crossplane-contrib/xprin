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

package utils

import (
	"strings"
	"testing"
)

func TestCreatePlaceholder(t *testing.T) {
	tests := []struct {
		name        string
		templateVar string
		want        string
	}{
		{
			name:        "simple variable",
			templateVar: ".Inputs.XR",
			want:        PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
		},
		{
			name:        "repository variable",
			templateVar: ".Repositories.myrepo",
			want:        PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose,
		},
		{
			name:        "empty variable",
			templateVar: "",
			want:        PlaceholderOpen + PlaceholderClose,
		},
		{
			name:        "variable with spaces",
			templateVar: "  .Inputs.XR  ",
			want:        PlaceholderOpen + "  .Inputs.XR  " + PlaceholderClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreatePlaceholder(tt.templateVar)
			if got != tt.want {
				t.Errorf("CreatePlaceholder(%q) = %q, want %q", tt.templateVar, got, tt.want)
			}
		})
	}
}

func TestReplaceTemplateVarsWithPlaceholders(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single template variable",
			content: "path: {{ .Inputs.XR }}",
			want:    "path: " + PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
		},
		{
			name:    "multiple template variables",
			content: "{{ .Repositories.myrepo }}/functions and {{ .Inputs.XR }}",
			want:    PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions and " + PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
		},
		{
			name:    "template variable with spaces",
			content: "path: {{  .Inputs.XR  }}",
			want:    "path: " + PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
		},
		{
			name:    "no template variables",
			content: "path: /some/static/path",
			want:    "path: /some/static/path",
		},
		{
			name:    "mixed content",
			content: "static: /path and dynamic: {{ .Repositories.myrepo }}/functions",
			want:    "static: /path and dynamic: " + PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions",
		},
		{
			name:    "nested template variables",
			content: "{{ .Inputs.XR }} and {{ .Outputs.XR }}",
			want:    PlaceholderOpen + ".Inputs.XR" + PlaceholderClose + " and " + PlaceholderOpen + ".Outputs.XR" + PlaceholderClose,
		},
		{
			name:    "template variable in YAML",
			content: "functions: {{ .Repositories.myrepo }}/functions\ncrds:\n  - {{ .Repositories.otherrepo }}/crds",
			want:    "functions: " + PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions\ncrds:\n  - " + PlaceholderOpen + ".Repositories.otherrepo" + PlaceholderClose + "/crds",
		},
		{
			name:    "empty template variable",
			content: "path: {{ }}",
			want:    "path: " + PlaceholderOpen + PlaceholderClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReplaceTemplateVarsWithPlaceholders(tt.content)
			if got != tt.want {
				t.Errorf("ReplaceTemplateVarsWithPlaceholders() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRestoreTemplateVars(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single placeholder",
			content: "path: " + PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
			want:    "path: {{.Inputs.XR}}",
		},
		{
			name:    "multiple placeholders",
			content: PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions and " + PlaceholderOpen + ".Inputs.XR" + PlaceholderClose,
			want:    "{{.Repositories.myrepo}}/functions and {{.Inputs.XR}}",
		},
		{
			name:    "no placeholders",
			content: "path: /some/static/path",
			want:    "path: /some/static/path",
		},
		{
			name:    "mixed content",
			content: "static: /path and dynamic: " + PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions",
			want:    "static: /path and dynamic: {{.Repositories.myrepo}}/functions",
		},
		{
			name:    "empty placeholder",
			content: "path: " + PlaceholderOpen + PlaceholderClose,
			want:    "path: {{}}",
		},
		{
			name:    "placeholder in YAML",
			content: "functions: " + PlaceholderOpen + ".Repositories.myrepo" + PlaceholderClose + "/functions\ncrds:\n  - " + PlaceholderOpen + ".Repositories.otherrepo" + PlaceholderClose + "/crds",
			want:    "functions: {{.Repositories.myrepo}}/functions\ncrds:\n  - {{.Repositories.otherrepo}}/crds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RestoreTemplateVars(tt.content)
			if got != tt.want {
				t.Errorf("RestoreTemplateVars() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReplaceAndRestoreRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "single variable",
			content: "path: {{ .Inputs.XR }}",
		},
		{
			name:    "multiple variables",
			content: "{{ .Repositories.myrepo }}/functions and {{ .Inputs.XR }}",
		},
		{
			name:    "variables with spaces",
			content: "path: {{  .Inputs.XR  }}",
		},
		{
			name:    "mixed content",
			content: "static: /path and dynamic: {{ .Repositories.myrepo }}/functions",
		},
		{
			name:    "YAML with variables",
			content: "functions: {{ .Repositories.myrepo }}/functions\ncrds:\n  - {{ .Repositories.otherrepo }}/crds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace template variables with placeholders
			replaced := ReplaceTemplateVarsWithPlaceholders(tt.content)

			// Verify that placeholders were created (content should be different)
			if replaced == tt.content && strings.Contains(tt.content, "{{") {
				t.Errorf("ReplaceTemplateVarsWithPlaceholders() did not replace template variables")
			}

			// Restore template variables from placeholders
			restored := RestoreTemplateVars(replaced)

			// Verify that template variables were restored (should contain {{ and }})
			if !strings.Contains(restored, "{{") || !strings.Contains(restored, "}}") {
				t.Errorf("RestoreTemplateVars() did not restore template variables: %q", restored)
			}

			// Verify that the variable names are preserved (even if whitespace differs)
			// Extract variable names from original and restored
			originalVars := extractTemplateVarNames(tt.content)
			restoredVars := extractTemplateVarNames(restored)

			if len(originalVars) != len(restoredVars) {
				t.Errorf("Variable count mismatch: original has %d, restored has %d", len(originalVars), len(restoredVars))
			}

			for i, origVar := range originalVars {
				if i >= len(restoredVars) {
					t.Errorf("Missing variable in restored: %q", origVar)
					continue
				}
				// Compare normalized variable names (trimmed)
				normalizedOrig := strings.TrimSpace(origVar)

				normalizedRestored := strings.TrimSpace(restoredVars[i])
				if normalizedOrig != normalizedRestored {
					t.Errorf("Variable mismatch at index %d: original %q, restored %q", i, normalizedOrig, normalizedRestored)
				}
			}
		})
	}
}

// extractTemplateVarNames extracts template variable names from content.
func extractTemplateVarNames(content string) []string {
	var vars []string
	// Simple extraction: find content between {{ and }}
	start := 0
	for {
		openIdx := strings.Index(content[start:], "{{")
		if openIdx == -1 {
			break
		}

		openIdx += start

		closeIdx := strings.Index(content[openIdx:], "}}")
		if closeIdx == -1 {
			break
		}

		closeIdx += openIdx
		varName := strings.TrimSpace(content[openIdx+2 : closeIdx])
		vars = append(vars, varName)
		start = closeIdx + 2
	}

	return vars
}
