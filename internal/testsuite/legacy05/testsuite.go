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

package legacy05

import (
	"fmt"

	"github.com/crossplane-contrib/xprin/internal/api"
)

const docs = "https://github.com/crossplane-contrib/xprin/blob/release-0.6/docs/how-it-works.md#dependencies-between-test-cases-needs"

// Warning is the deprecation notice for a testsuite file in this format.
func Warning(file string) string {
	return fmt.Sprintf("%[1]s: testcases that use the outputs of other testcases (.Tests) will have to list them in `needs` from xprin 0.7.0. See %[2]s, and run `xprin-helpers check-testsuite-needs %[1]s` to see what to add.", file, docs)
}

// IsTestSuite tells if the testsuite file is in the format of xprin up to 0.5, which means that:
// - no testcase lists `needs`
// - some testcase uses `.Tests`, also in a way that can't work (a typo in the ID, a testcase listed below)
// Any other file is in the current format.
func IsTestSuite(spec *api.TestSuiteSpec) bool {
	if spec == nil || hasNeeds(spec) {
		return false
	}

	for _, testCase := range spec.Tests {
		merged := testCase
		merged.MergeCommon(spec.Common)

		if hasImplicitNeeds(merged) {
			return true
		}
	}

	return false
}

// hasNeeds tells if some testcase of the file lists something in `needs`.
func hasNeeds(spec *api.TestSuiteSpec) bool {
	for _, testCase := range spec.Tests {
		if len(testCase.Needs) > 0 {
			return true
		}
	}

	return false
}

// hasImplicitNeeds tells if a testcase uses `.Tests` in the actions of its templates. It is also true for:
//   - a mention that is not a use (a comment or a string in an action), so that a use is never missed
//   - a testcase that can't be marshalled, which the runner fails anyway
func hasImplicitNeeds(testCase api.TestCase) bool {
	ids, needsAll := FindImplicitNeeds(testCase)

	return len(ids) > 0 || needsAll
}
