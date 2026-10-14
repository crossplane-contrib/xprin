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

package discovery

import (
	"fmt"
	"slices"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
)

// checkTestSuiteFile checks a loaded testsuite file, and returns an error with all the problems it finds:
// - if test case names are non-empty
// - if test case IDs are valid, and unique (only for tests that have IDs)
// - if the `needs` of the test cases are right (see checkNeeds).
func checkTestSuiteFile(spec *api.TestSuiteSpec) error {
	problems := slices.Concat(checkNamesAndIDs(spec), checkNeeds(spec))
	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("invalid testsuite file:\n- %s", strings.Join(problems, "\n- "))
}

// checkNamesAndIDs returns the problems with the names and the IDs of the test cases.
func checkNamesAndIDs(spec *api.TestSuiteSpec) []string {
	var allErrors []string

	// Check if an ID contains only alphanumeric characters, underscores, and hyphens
	hasValidID := func(id string) bool {
		if len(id) == 0 {
			return false
		}

		for _, char := range id {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' && char != '-' {
				return false
			}
		}

		return true
	}

	// Track used IDs to detect duplicates
	usedIDs := make(map[string]bool)

	for i := range spec.Tests {
		test := &spec.Tests[i]

		// Check for empty name
		if test.Name == "" {
			allErrors = append(allErrors, "test case has empty name")
		}

		// Only validate and check uniqueness for IDs that are explicitly provided
		if test.ID != "" {
			// Validate test ID format
			if !hasValidID(test.ID) {
				allErrors = append(allErrors, fmt.Sprintf("test case ID '%s' contains invalid characters (allowed: alphanumeric, underscore, hyphen)", test.ID))
			}

			// Check for duplicate IDs (only among tests that have IDs)
			if usedIDs[test.ID] {
				allErrors = append(allErrors, fmt.Sprintf("duplicate test case ID '%s' found", test.ID))
			} else {
				usedIDs[test.ID] = true
			}
		}
	}

	return allErrors
}

// checkNeeds returns the problems with the `needs` of the testcases.
// A testcase can only need the ones listed before it, thus the followings are errors:
// - an ID that doesn't exist
// - a testcase needs itself
// - a testcase needs one listed afterwards.
func checkNeeds(spec *api.TestSuiteSpec) []string {
	// The position of each ID
	positions := make(map[string]int)

	for i, test := range spec.Tests {
		if _, ok := positions[test.ID]; !ok && test.ID != "" {
			positions[test.ID] = i
		}
	}

	var problems []string

	for i, test := range spec.Tests {
		if test.Needs != nil && len(test.Needs) == 0 {
			problems = append(problems, fmt.Sprintf("test case %q has an empty `needs`: list the IDs of the test cases it needs, or remove it", test.Name))
		}

		for _, need := range test.Needs {
			position, ok := positions[need]

			switch {
			case !ok:
				problems = append(problems, fmt.Sprintf("test case %q needs %q, but no test case has that ID", test.Name, need))
			case position == i:
				problems = append(problems, fmt.Sprintf("test case %q needs itself (%q)", test.Name, need))
			case position > i:
				problems = append(problems, fmt.Sprintf("test case %q needs %q, which is listed after it", test.Name, need))
			}
		}
	}

	return problems
}
