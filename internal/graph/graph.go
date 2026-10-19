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

// Order returns the positions of the testcases in the order they run: each one after the testcases it
// needs. Whenever several can run, the one listed first in the file does, so the order depends only
// on the file and the dependencies (never on the order of the IDs in a `needs`), and a file without
// dependencies runs in the order it is written. It is the same for every command that uses it.
// Testcases that need each other, directly or not, have no order: such a testsuite file is rejected
// when it is loaded, so there are none here.
func (g *Graph) Order() []int {
	// waiting[i] counts the testcases that testcase i still has to wait for: it can run when it reaches 0
	waiting := make([]int, len(g.deps))
	// needed[j] lists the testcases that need testcase j: the ones to unblock a little when j has run
	needed := make([][]int, len(g.deps))

	// Example: if B needs A, waiting[B] starts at 1 and needed[A] is [B]. When A runs, waiting[B]
	// drops to 0 and B can run.

	// the positions that can run, ascending
	var ready []int

	// Setup: count what each testcase waits for, note who needs it, and mark the ones that wait for nothing
	for i, deps := range g.deps {
		waiting[i] = len(deps)

		for _, j := range deps {
			needed[j] = append(needed[j], i)
		}

		if len(deps) == 0 {
			ready = append(ready, i)
		}
	}

	order := make([]int, 0, len(g.deps))

	// Run the first ready testcase (the earliest in the file), then unblock the ones that waited for it
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]

		order = append(order, i)

		// Each testcase that needs it now waits for one less; at zero it is ready, in its sorted place
		for _, dependent := range needed[i] {
			if waiting[dependent]--; waiting[dependent] == 0 {
				at, _ := slices.BinarySearch(ready, dependent)
				ready = slices.Insert(ready, at, dependent)
			}
		}
	}

	return order
}
