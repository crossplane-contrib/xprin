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

package placeholder

import (
	"strings"
	"testing"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name        string
		templateVar string
		want        string
	}{
		{
			name:        "simple variable",
			templateVar: ".Inputs.XR",
			want:        Open + ".Inputs.XR" + Close,
		},
		{
			name:        "repository variable",
			templateVar: ".Repositories.myrepo",
			want:        Open + ".Repositories.myrepo" + Close,
		},
		{
			name:        "empty variable",
			templateVar: "",
			want:        Open + Close,
		},
		{
			name:        "variable with spaces",
			templateVar: "  .Inputs.XR  ",
			want:        Open + "  .Inputs.XR  " + Close,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Create(tt.templateVar)
			if got != tt.want {
				t.Errorf("Create(%q) = %q, want %q", tt.templateVar, got, tt.want)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single template variable",
			content: "path: {{ .Inputs.XR }}",
			want:    "path: " + Open + ".Inputs.XR" + Close,
		},
		{
			name:    "multiple template variables",
			content: "{{ .Repositories.myrepo }}/functions and {{ .Inputs.XR }}",
			want:    Open + ".Repositories.myrepo" + Close + "/functions and " + Open + ".Inputs.XR" + Close,
		},
		{
			name:    "template variable with spaces",
			content: "path: {{  .Inputs.XR  }}",
			want:    "path: " + Open + ".Inputs.XR" + Close,
		},
		{
			name:    "no template variables",
			content: "path: /some/static/path",
			want:    "path: /some/static/path",
		},
		{
			name:    "mixed content",
			content: "static: /path and dynamic: {{ .Repositories.myrepo }}/functions",
			want:    "static: /path and dynamic: " + Open + ".Repositories.myrepo" + Close + "/functions",
		},
		{
			name:    "nested template variables",
			content: "{{ .Inputs.XR }} and {{ .Outputs.XR }}",
			want:    Open + ".Inputs.XR" + Close + " and " + Open + ".Outputs.XR" + Close,
		},
		{
			name:    "template variable in YAML",
			content: "functions: {{ .Repositories.myrepo }}/functions\ncrds:\n  - {{ .Repositories.otherrepo }}/crds",
			want:    "functions: " + Open + ".Repositories.myrepo" + Close + "/functions\ncrds:\n  - " + Open + ".Repositories.otherrepo" + Close + "/crds",
		},
		{
			name:    "empty template variable",
			content: "path: {{ }}",
			want:    "path: " + Open + Close,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Replace(tt.content)
			if got != tt.want {
				t.Errorf("Replace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRestore(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single placeholder",
			content: "path: " + Open + ".Inputs.XR" + Close,
			want:    "path: {{.Inputs.XR}}",
		},
		{
			name:    "multiple placeholders",
			content: Open + ".Repositories.myrepo" + Close + "/functions and " + Open + ".Inputs.XR" + Close,
			want:    "{{.Repositories.myrepo}}/functions and {{.Inputs.XR}}",
		},
		{
			name:    "no placeholders",
			content: "path: /some/static/path",
			want:    "path: /some/static/path",
		},
		{
			name:    "mixed content",
			content: "static: /path and dynamic: " + Open + ".Repositories.myrepo" + Close + "/functions",
			want:    "static: /path and dynamic: {{.Repositories.myrepo}}/functions",
		},
		{
			name:    "empty placeholder",
			content: "path: " + Open + Close,
			want:    "path: {{}}",
		},
		{
			name:    "placeholder in YAML",
			content: "functions: " + Open + ".Repositories.myrepo" + Close + "/functions\ncrds:\n  - " + Open + ".Repositories.otherrepo" + Close + "/crds",
			want:    "functions: {{.Repositories.myrepo}}/functions\ncrds:\n  - {{.Repositories.otherrepo}}/crds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Restore(tt.content)
			if got != tt.want {
				t.Errorf("Restore() = %q, want %q", got, tt.want)
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
			replaced := Replace(tt.content)

			// Verify that placeholders were created (content should be different)
			if replaced == tt.content && strings.Contains(tt.content, "{{") {
				t.Errorf("Replace() did not replace template variables")
			}

			// Restore template variables from placeholders
			restored := Restore(replaced)

			// Verify that template variables were restored (should contain {{ and }})
			if !strings.Contains(restored, "{{") || !strings.Contains(restored, "}}") {
				t.Errorf("Restore() did not restore template variables: %q", restored)
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
