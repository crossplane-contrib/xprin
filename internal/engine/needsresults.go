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

package engine

import "fmt"

// GetNeedsResults returns the results of the test cases in needs, out of the completed ones (by ID), and
// why a test case that needs them can't run: an error for each one that failed or was skipped (it can't work
// without them), in the order of needs. The ones that have not run are in neither. skipReasons is empty when
// the test case can run.
func GetNeedsResults(needs []string, completed map[string]*TestCaseResult) (needsResults map[string]*TestCaseResult, skipReasons []error) {
	needsResults = make(map[string]*TestCaseResult, len(needs))

	for _, need := range needs {
		result, ok := completed[need]
		if !ok {
			continue
		}

		needsResults[need] = result

		switch result.Status {
		case StatusFail():
			skipReasons = append(skipReasons, fmt.Errorf("needs failed test case %q", need))
		case StatusSkip():
			skipReasons = append(skipReasons, fmt.Errorf("needs skipped test case %q", need))
		}
	}

	return needsResults, skipReasons
}
