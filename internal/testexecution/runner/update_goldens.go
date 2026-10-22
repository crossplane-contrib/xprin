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

// updateGoldensMode is the mode of `xprin update-goldens`. The other one is testMode, in mode.go, which says what a mode is.
type updateGoldensMode struct {
	r *Runner
}

// skips tells that update-goldens has nothing to do with a test case that writes no golden file, and that no golden
// file depends on: the graph knows which those are. One without an ID is not in the graph by it, but no test case
// can need it, so only its own golden-file assertions matter.
func (m updateGoldensMode) skips(testCase api.TestCase) bool {
	return !testCase.HasAssertionsGoldenFiles() && !m.r.testSuite.Graph.NodeByID(testCase.ID).NeededForGoldens
}

// runPhasesAfterRender is the one phase of update-goldens that follows the render: it writes the golden files of the
// test case from what was rendered. Validate, the assertions and the post-test hooks are not run. The outputs are
// exported, as in xprin test, for the test cases that need this one.
func (m updateGoldensMode) runPhasesAfterRender(testCase api.TestCase, env *testCaseEnv, _ map[string]*engine.TestCaseResult, result *engine.TestCaseResult) *engine.TestCaseResult {
	result = m.r.runUpdateGoldens(testCase, result)
	if result.Status == engine.StatusFail() {
		return result
	}

	if err := m.r.exportOutputs(testCase, env, result); err != nil {
		return result.Fail(err)
	}

	return result
}

// goldenFilesHeader is the header of the section where a test case lists its golden files: the ones that it
// wrote, and the one that it could not.
const goldenFilesHeader = "Golden Files"

// runUpdateGoldens writes render output to the `expected:` paths of assertions.diff/dyff entries.
// Called instead of validate/assertions/post-test hooks when UpdateGoldens is set.
func (r *Runner) runUpdateGoldens(testCase api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	allAssertions := append(testCase.Assertions.Diff, testCase.Assertions.Dyff...) //nolint:gocritic // intentional append to new slice

	for _, a := range allAssertions {
		path, err := r.writeGoldenFile(a, &result.Outputs)
		if err != nil {
			// It stops at the first one that fails. The failure is a line of the section, after the files that were
			// written, so there is no other error to show.
			result.AddToSection(goldenFilesHeader, engine.StatusError(), path+": "+err.Error())

			return result.Fail(nil)
		}

		result.AddToSection(goldenFilesHeader, engine.StatusPass(), path)
	}

	return result.Complete()
}

// writeGoldenFile resolves the expected path and actual path for a golden-file assertion,
// then writes the actual content to the expected path (creating or overwriting).
// Returns the path of the golden file, also when it fails, to say which one it was: the absolute path that was
// written, or the one of the assertion when it can't be resolved.
func (r *Runner) writeGoldenFile(a api.AssertionGoldenFile, outputs *engine.Outputs) (string, error) {
	expectedPath, err := r.expandPathRelativeToTestSuiteFile(r.testSuite.File, a.Expected)
	if err != nil {
		return a.Expected, fmt.Errorf("invalid expected path: %w", err)
	}

	var actualPath string
	if a.Resource == "" {
		actualPath = outputs.Render
	} else {
		var ok bool

		actualPath, ok = outputs.Rendered[a.Resource]
		if !ok {
			return expectedPath, fmt.Errorf("resource %q not found in render output", a.Resource)
		}
	}

	actualBytes, err := afero.ReadFile(r.fs, actualPath)
	if err != nil {
		return expectedPath, fmt.Errorf("read actual file %q: %w", actualPath, err)
	}

	if err := afero.WriteFile(r.fs, expectedPath, actualBytes, 0o600); err != nil {
		return expectedPath, fmt.Errorf("write expected file: %w", err)
	}

	return expectedPath, nil
}
