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

package testsuiteneeds

import "slices"

// testCaseCheckResult is what the check finds in a test case of the file.
type testCaseCheckResult struct {
	name string
	// usesTests is set when the test case uses other test cases through `.Tests`, in a way that can be traced.
	usesTests bool
	// skipReason is why the test case can't be checked.
	skipReason string
	// missing are the IDs it uses through `.Tests` but doesn't list in `needs`, in the order of the file.
	missing []string
	// problems are what a person has to fix.
	problems []string
}

// testSuiteCheckResult is what checking a testsuite file finds: its test cases, in its order.
type testSuiteCheckResult []testCaseCheckResult

// missesNeeds tells if any test case misses needs.
func (r testSuiteCheckResult) missesNeeds() bool {
	return slices.ContainsFunc(r, func(tc testCaseCheckResult) bool { return len(tc.missing) > 0 })
}

// requiresAPerson tells if any test case has a problem that a person has to fix.
func (r testSuiteCheckResult) requiresAPerson() bool {
	return slices.ContainsFunc(r, func(tc testCaseCheckResult) bool { return len(tc.problems) > 0 })
}
