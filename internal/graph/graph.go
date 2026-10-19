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
	"fmt"
	"slices"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
)

// Graph records, for each testcase of a testsuite file, the testcases that it needs.
type Graph struct {
	// deps[i] are the testcases that testcase i needs, as their positions in the file (ascending, once each).
	deps [][]int
	// ids[i] is the ID of the testcase at position i of the file, empty when it has none.
	ids []string
}

// Build returns the dependency graph of the testcases in spec: a testcase depends on the testcases
// that it lists in `needs`. A `needs` that points to no testcase (an ID that doesn't exist) is not a
// dependency: the testsuite file is rejected when it is loaded, before it gets here.
func Build(spec *api.TestSuiteSpec) *Graph {
	g := &Graph{deps: make([][]int, len(spec.Tests)), ids: make([]string, len(spec.Tests))}

	// Turn the IDs in the `needs` of each testcase into positions
	for i, testCase := range spec.Tests {
		g.ids[i] = testCase.ID

		for _, need := range testCase.Needs {
			if j := spec.PositionOf(need); j >= 0 {
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
// Testcases that need each other, directly or not, have no order and are left out: CheckCycles reports them,
// and a testsuite file that has any is rejected when it is loaded.
func (g *Graph) Order() []int {
	order, _ := g.sortByDependencies(nil)

	return order
}

// CheckCycles describes each cycle of testcases that need each other, one message for each, like
// `needs cycle: a -> b -> a` (a needs b, which needs a). It is empty when the testcases can be ordered.
func (g *Graph) CheckCycles() []string {
	var cycles []string

	// done marks the testcases of the cycles found so far: the next sort leaves them out, so it can't find them again
	done := make([]bool, len(g.deps))

	for {
		// Sort what is not done: what can't run is stuck, either in a cycle or waiting behind one
		order, waiting := g.sortByDependencies(done)

		if g.noCyclesLeft(order, done) {
			return cycles
		}

		// Find one cycle among the stuck testcases, and keep its message
		cycle := g.findCycle(waiting)
		cycles = append(cycles, g.describe(cycle))

		// Leave it out of the next round
		for _, i := range cycle {
			done[i] = true
		}
	}
}

// noCyclesLeft tells if every testcase that is not done is in order: nothing is stuck, so no cycle is left to find.
func (g *Graph) noCyclesLeft(order []int, done []bool) bool {
	return len(order) == len(g.deps)-count(done)
}

// count returns how many of the values are true.
func count(values []bool) int {
	n := 0

	for _, v := range values {
		if v {
			n++
		}
	}

	return n
}

// sortByDependencies runs the testcases that can run, and returns the following slices:
// - order: their positions, in the order they ran
// - waiting: for each testcase, how many of the ones it needs did not run; not zero means it is stuck
// The testcases in done count as already run: they are left out, and nobody waits for them (nil for none).
func (g *Graph) sortByDependencies(done []bool) (order, waiting []int) {
	isDone := func(i int) bool { return done != nil && done[i] }

	// waiting[i] counts the testcases that testcase i still has to wait for: it can run when it reaches 0
	waiting = make([]int, len(g.deps))
	// needed[j] lists the testcases that need testcase j: the ones to unblock a little when j has run
	needed := make([][]int, len(g.deps))

	// Example: if B needs A, waiting[B] starts at 1 and needed[A] is [B]. When A runs, waiting[B]
	// drops to 0 and B can run.

	// the positions that can run, ascending
	var ready []int

	// Setup: count what each testcase waits for, note who needs it, and mark the ones that wait for nothing
	for i, deps := range g.deps {
		// A testcase that is done is left out
		if isDone(i) {
			continue
		}

		for _, j := range deps {
			// A need that is done doesn't make i wait
			if !isDone(j) {
				waiting[i]++

				needed[j] = append(needed[j], i)
			}
		}

		if waiting[i] == 0 {
			ready = append(ready, i)
		}
	}

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

	return order, waiting
}

// findCycle returns the testcases of one cycle, from one of them and back to it (the first and the last are the same),
// among the stuck ones (waiting > 0). It walks from a stuck testcase to the first of its needs that is also stuck.
// Every stuck testcase has one, so the walk can't end: it has to reach a testcase it already visited.
func (g *Graph) findCycle(waiting []int) []int {
	// path is the testcases visited, in order
	var path []int
	// at[i] is where i is in path
	at := make(map[int]int)

	// Start at the first stuck testcase
	for i := slices.IndexFunc(waiting, func(w int) bool { return w > 0 }); ; {
		// Seen before: the walk went around. The cycle is the path from there, closed with i
		// (what comes before only led to the cycle)
		if start, seen := at[i]; seen {
			return append(path[start:], i)
		}

		// First visit: add it to the path
		at[i] = len(path)
		path = append(path, i)

		// Go on to the first of its needs that is also stuck
		i = g.deps[i][slices.IndexFunc(g.deps[i], func(j int) bool { return waiting[j] > 0 })]
	}
}

// describe returns the message for a cycle of testcases, as findCycle returns it.
func (g *Graph) describe(cycle []int) string {
	labels := make([]string, len(cycle))

	for k, i := range cycle {
		labels[k] = g.ids[i]
		if labels[k] == "" {
			labels[k] = fmt.Sprintf("test case %d", i+1)
		}
	}

	return "needs cycle: " + strings.Join(labels, " -> ")
}
