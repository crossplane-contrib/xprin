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
)

// runOrder returns the positions of the test cases in the order they run: the order of the file in a legacy
// testsuite file, which has no dependencies and doesn't need the graph, and the order of the dependency
// graph in any other.
func (r *Runner) runOrder() []int {
	if r.legacy05 {
		order := make([]int, len(r.testSuite.Spec.Tests))
		for i := range order {
			order[i] = i
		}

		return order
	}

	return r.testSuite.Graph.GetOrder()
}

// getNeedsResults returns the results that the test case can use with `.Tests`, and why it can't run, if
// some test case it needs failed or was skipped. The results are the ones of its needs, except in a legacy
// testsuite file, where it is all of the ones that ran before it (it has no needs, so nothing to skip for).
func (r *Runner) getNeedsResults(testCase api.TestCase, testSuiteResult *engine.TestSuiteResult) (needsResults map[string]*engine.TestCaseResult, skipReasons []error) {
	completed := testSuiteResult.GetCompletedTests()
	if r.legacy05 {
		return completed, nil
	}

	return engine.GetNeedsResults(testCase.Needs, completed)
}
