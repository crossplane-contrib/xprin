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
	"fmt"
	"regexp"
	"strings"
)

// Constants for template variable placeholders.
const (
	PlaceholderOpen  = "__OPEN__"
	PlaceholderClose = "__CLOSE__"
)

// CreatePlaceholder creates a template variable placeholder for testing.
func CreatePlaceholder(templateVar string) string {
	return fmt.Sprintf("%s%s%s", PlaceholderOpen, templateVar, PlaceholderClose)
}

// ReplaceTemplateVarsWithPlaceholders replaces template variables with placeholders.
func ReplaceTemplateVarsWithPlaceholders(content string) string {
	re := regexp.MustCompile(`\{\{\s*(.*?)\s*\}\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the content inside the curly brackets
		innerContent := re.FindStringSubmatch(match)[1]
		// Remove any remaining whitespace
		cleanContent := strings.TrimSpace(innerContent)

		return fmt.Sprintf("%s%s%s", PlaceholderOpen, cleanContent, PlaceholderClose)
	})
}

// RestoreTemplateVars restores template variables from placeholders.
func RestoreTemplateVars(content string) string {
	content = strings.ReplaceAll(content, PlaceholderOpen, "{{")
	content = strings.ReplaceAll(content, PlaceholderClose, "}}")

	return content
}
