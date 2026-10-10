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

// Package placeholder encodes/decodes the template vars of testsuite files as placeholders
package placeholder

import (
	"fmt"
	"regexp"
	"strings"
)

// Open and Close replace the {{ and }} of a template variable.
const (
	Open  = "__OPEN__"
	Close = "__CLOSE__"
)

// Create returns a placeholder for templateVar, as Replace would create it.
func Create(templateVar string) string {
	return fmt.Sprintf("%s%s%s", Open, templateVar, Close)
}

// Replace replaces the template variables in content with placeholders.
func Replace(content string) string {
	re := regexp.MustCompile(`\{\{\s*(.*?)\s*\}\}`)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the content inside the curly brackets
		innerContent := re.FindStringSubmatch(match)[1]
		// Remove any remaining whitespace
		cleanContent := strings.TrimSpace(innerContent)

		return fmt.Sprintf("%s%s%s", Open, cleanContent, Close)
	})
}

// Restore turns the placeholders in content back into template variables.
func Restore(content string) string {
	content = strings.ReplaceAll(content, Open, "{{")
	content = strings.ReplaceAll(content, Close, "}}")

	return content
}
