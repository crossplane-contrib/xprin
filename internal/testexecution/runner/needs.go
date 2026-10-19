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
	"errors"
	"regexp"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/graph"
)

// runOrder returns the positions of the test cases in the order they run: the order of the file in a legacy
// testsuite file, which has no dependencies and doesn't need the graph, and the order of the dependency
// graph in any other.
func (r *Runner) runOrder() []int {
	if r.legacy05 {
		order := make([]int, len(r.testSuiteSpec.Tests))
		for i := range order {
			order[i] = i
		}

		return order
	}

	return graph.Build(r.testSuiteSpec).Order()
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

// missingKey finds the key in the error of a template that uses `.Tests` for a test case that is not there:
// `.Tests.<id>` fails with "map has no entry for key", and `(index .Tests "<id>")`, the way to use an ID that
// is not a field name (a hyphen), gives nothing for it, so using its outputs fails with a nil pointer.
var missingKey = regexp.MustCompile(`<[^>]*\.Tests[^>]*>: map has no entry for key "([^"]*)"|at <"([^"]*)">: nil pointer evaluating \*engine\.TestCaseResult\.`)

// explainMissingNeeds replaces the error of a template that uses `.Tests` for a test case of the file that
// is not in `needs` (the usual reason for it not being there) with one that says so, in a testsuite file
// that is not legacy. Any other error is returned as it is.
func (r *Runner) explainMissingNeeds(err error) error {
	if r.legacy05 {
		return err
	}

	match := missingKey.FindStringSubmatch(err.Error())
	if match == nil {
		return err
	}

	id := match[1] + match[2] // one of them is set, depending on how the test case is used

	for _, testCase := range r.testSuiteSpec.Tests {
		if testCase.ID == id {
			return errors.New("one or more test cases are used but are not listed in needs")
		}
	}

	return err
}
