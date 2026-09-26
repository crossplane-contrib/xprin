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
	"fmt"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
)

// runUpdateGoldens writes render output to the `expected:` paths of assertions.diff/dyff entries.
// Called instead of validate/assertions/post-test hooks when UpdateGoldens is set.
// Test cases without assertions.diff/dyff are skipped (result marked SKIP).
func (r *Runner) runUpdateGoldens(testCase api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	allAssertions := append(testCase.Assertions.Diff, testCase.Assertions.Dyff...) //nolint:gocritic // intentional append to new slice

	for _, a := range allAssertions {
		written, err := r.writeGoldenFile(a, &result.Outputs)
		if err != nil {
			return result.Fail(fmt.Errorf("update-goldens: %w", err))
		}

		result.WrittenGoldenFiles = append(result.WrittenGoldenFiles, written)
	}

	return result.Complete()
}

// writeGoldenFile resolves the expected path and actual path for a golden-file assertion,
// then writes the actual content to the expected path (creating or overwriting).
// Returns the absolute path that was written.
func (r *Runner) writeGoldenFile(a api.AssertionGoldenFile, outputs *engine.Outputs) (string, error) {
	expectedPath, err := r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, a.Expected)
	if err != nil {
		return "", fmt.Errorf("invalid expected path %q: %w", a.Expected, err)
	}

	var actualPath string
	if a.Resource == "" {
		actualPath = outputs.Render
	} else {
		var ok bool

		actualPath, ok = outputs.Rendered[a.Resource]
		if !ok {
			return "", fmt.Errorf("resource %q not found in render output", a.Resource)
		}
	}

	actualBytes, err := afero.ReadFile(r.fs, actualPath)
	if err != nil {
		return "", fmt.Errorf("read actual file %q: %w", actualPath, err)
	}

	if err := afero.WriteFile(r.fs, expectedPath, actualBytes, 0o600); err != nil {
		return "", fmt.Errorf("write expected file %q: %w", expectedPath, err)
	}

	return expectedPath, nil
}
