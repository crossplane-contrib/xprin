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

// Package graph builds the dependency graph of the testcases of a testsuite file, from their `needs`.
package graph

import (
	"slices"

	"github.com/crossplane-contrib/xprin/internal/api"
)

// Graph records, for each testcase of a testsuite file, the testcases that it needs.
type Graph struct {
	// deps[i] are the testcases that testcase i needs, as their positions in the file (ascending, once each).
	deps [][]int
}

// Build returns the dependency graph of the testcases in spec: a testcase depends on the testcases
// that it lists in `needs`. A `needs` that points to no testcase (an ID that doesn't exist) is not a
// dependency: the testsuite file is rejected when it is loaded, before it gets here.
func Build(spec *api.TestSuiteSpec) *Graph {
	g := &Graph{deps: make([][]int, len(spec.Tests))}

	// The position of each testcase, by ID: only those with an ID can be needed.
	positions := make(map[string]int)

	// First pass: fill positions (the first testcase wins when an ID is repeated)
	for i, testCase := range spec.Tests {
		if _, ok := positions[testCase.ID]; !ok && testCase.ID != "" {
			positions[testCase.ID] = i
		}
	}

	// Second pass: turn the IDs in the `needs` of each testcase into positions
	for i, testCase := range spec.Tests {
		for _, need := range testCase.Needs {
			if j, ok := positions[need]; ok {
				g.deps[i] = append(g.deps[i], j)
			}
		}

		// In order, and once each: the `needs` can list the same testcase twice
		slices.Sort(g.deps[i])
		g.deps[i] = slices.Compact(g.deps[i])
	}

	return g
}
