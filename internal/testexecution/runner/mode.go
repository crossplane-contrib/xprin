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
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/utils"
)

// runnerMode is what differs between the commands that run the test cases of a testsuite file: `xprin test` runs
// them and checks them, and `xprin update-goldens` runs them to write their golden files. Everything else, from
// the temporary directory to the render, is the same. A runner has one, for the whole run of its testsuite file.
type runnerMode interface {
	// skips tells if the test case is skipped without a word, whatever the test cases it needs did.
	skips(testCase api.TestCase) bool
	// runPhasesAfterRender runs the phases that follow the render of a test case, and returns its result.
	runPhasesAfterRender(testCase api.TestCase, env *testCaseEnv, needsResults map[string]*engine.TestCaseResult, result *engine.TestCaseResult) *engine.TestCaseResult
}

// newRunnerMode returns the mode of the command that the runner is for, from its options.
func newRunnerMode(r *Runner) runnerMode {
	if r.UpdateGoldens {
		return updateGoldensMode{r}
	}

	return testMode{r}
}

// getMode returns the mode that RunTests set. Without one, which is how the tests call runTestCase on its own,
// it is the one of xprin test.
func (r *Runner) getMode() runnerMode {
	if r.mode == nil {
		return testMode{r}
	}

	return r.mode
}

// testMode is the mode of `xprin test`.
type testMode struct {
	r *Runner
}

// skips tells that xprin test runs every test case.
func (testMode) skips(api.TestCase) bool {
	return false
}

// runPhasesAfterRender runs the phases of xprin test that follow the render: Validate, Assert and Finish. They
// validate the rendered resources, run the assertions and the post-test hooks, and export the outputs.
func (m testMode) runPhasesAfterRender(testCase api.TestCase, env *testCaseEnv, needsResults map[string]*engine.TestCaseResult, result *engine.TestCaseResult) *engine.TestCaseResult {
	r := m.r

	if failed := r.validateTestCase(env, testCase, result); failed != nil {
		return failed
	}

	if failed := r.runAssertions(env, testCase, result); failed != nil {
		return failed
	}

	r.runPostTestHooks(testCase, needsResults, result)

	if err := r.exportOutputs(testCase, env, result); err != nil {
		return result.Fail(err)
	}

	// Fail with nil when the pipeline failed: what failed is in the result
	if result.HasPipelineFailure() {
		return result.Fail(nil)
	}

	// Complete the test case result
	if r.Debug {
		utils.DebugPrintf("Test case '%s' completed with status: %s\n", testCase.Name, result.Status.String())
	}

	return result.Complete()
}
