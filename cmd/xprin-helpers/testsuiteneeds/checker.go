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

import (
	"fmt"
	"slices"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/graph"
)

// untraceable says that a use of `.Tests` can't be traced to the test cases it uses, the start of both of its messages.
const untraceable = "uses .Tests in a way that can't be traced to the test cases it uses (for example `range .Tests`)"

// checkFile finds, for each test case, the test cases it uses through `.Tests` without listing them in `needs`.
func checkFile(spec *api.TestSuiteSpec) testSuiteCheckResult {
	res := findMissing(spec)

	// Listing the missing needs would create a cycle: a person has to look at it
	if cycles := cyclesWith(spec, res); len(cycles) > 0 {
		for i, tc := range res {
			if len(tc.missing) == 0 {
				continue
			}

			res[i].problems = append(res[i].problems, fmt.Sprintf("listing the needs it misses (%s) would make the test cases need each other (%s)", strings.Join(tc.missing, ", "), strings.Join(cycles, "; ")))
			res[i].missing = nil
		}
	}

	return res
}

// findMissing returns what is known about each test case: the needs it misses, and its problems.
func findMissing(spec *api.TestSuiteSpec) testSuiteCheckResult {
	// The position of each test case that has an ID, which are the only ones that can be needed
	positions := make(map[string]int)

	for i, testCase := range spec.Tests {
		if _, ok := positions[testCase.ID]; !ok && testCase.ID != "" {
			positions[testCase.ID] = i
		}
	}

	res := make(testSuiteCheckResult, len(spec.Tests))

	for i, tc := range spec.Tests {
		merged := tc
		merged.MergeCommon(spec.Common)

		detected, needsAll := findImplicitNeeds(merged)

		check := &res[i]
		check.name = tc.Name

		if needsAll {
			// If it lists some needs, a person took care of it: that is not a failure
			if len(tc.Needs) > 0 {
				check.skipReason = untraceable + ": the needs that it lists can't be checked"
			} else {
				check.problems = append(check.problems, untraceable+": list the needs it uses")
			}

			continue
		}

		check.usesTests = len(detected) > 0

		for _, id := range detected {
			position, exists := positions[id]

			switch {
			case !exists:
				check.problems = append(check.problems, fmt.Sprintf("uses .Tests.%s, but no test case has that ID", id))
			case position == i:
				check.problems = append(check.problems, fmt.Sprintf("uses its own outputs (.Tests.%s)", id))
			case !slices.Contains(tc.Needs, id) && !slices.Contains(check.missing, id):
				check.missing = append(check.missing, id)
			}
		}

		// In the order of the file, so that the result doesn't depend on how the templates are written
		slices.SortFunc(check.missing, func(a, b string) int { return positions[a] - positions[b] })
	}

	return res
}

// cyclesWith returns the cycles that the test cases would have with the missing needs listed.
func cyclesWith(spec *api.TestSuiteSpec, res testSuiteCheckResult) []string {
	listed := *spec
	listed.Tests = slices.Clone(spec.Tests)

	for i, tc := range res {
		listed.Tests[i].Needs = append(slices.Clone(listed.Tests[i].Needs), tc.missing...)
	}

	return graph.Build(&listed).CheckCycles()
}
