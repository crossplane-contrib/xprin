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

package processor

import (
	"fmt"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// load loads and validates a single testsuite file.
func load(fs afero.Fs, path string) (*api.TestSuiteSpec, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read testsuite file %s: %w", path, err)
	}

	content := string(data)

	if strings.Contains(content, "{{") {
		content = utils.ReplaceTemplateVarsWithPlaceholders(content)
	}

	var testSuiteSpec api.TestSuiteSpec
	if err := yaml.Unmarshal([]byte(content), &testSuiteSpec); err != nil {
		return nil, fmt.Errorf("failed to parse testsuite file %s: %w", path, err)
	}

	if len(testSuiteSpec.Tests) == 0 {
		return nil, fmt.Errorf("no test cases found in testsuite file %s", path)
	}

	return &testSuiteSpec, nil
}
