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

// Package templates parses the Go templates of testsuite files, and tells what of a testcase is rendered as one.
package templates

import (
	"fmt"
	"text/template"

	"github.com/crossplane-contrib/xprin/internal/api"
	"sigs.k8s.io/yaml"
)

// Parse parses text (with {{ ... }}, not placeholders) as a template that errors on missing map keys.
func Parse(name, text string) (*template.Template, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return t, nil
}

// MarshalWithoutHooks returns the YAML of the testcase without its hooks: the part that is rendered as
// one template (each hook is rendered when it runs, as it can use the testcase's outputs).
func MarshalWithoutHooks(testCase api.TestCase) ([]byte, error) {
	testCase.Hooks = api.Hooks{} // a copy: the caller keeps its hooks

	data, err := yaml.Marshal(testCase)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test case to YAML: %w", err)
	}

	return data, nil
}
