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

package graph

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

// chainedSuite is the testsuite file of examples/mytests/5_chained_tests/example5_xprin.yaml, reduced to what the graph looks at.
const chainedSuite = `tests:
- name: "Base layer final loop"
  id: base_final
- name: "GCP layer first loop"
  id: gcp_first
  needs:
  - base_final
- name: "GCP layer second loop"
  id: gcp_second
  needs:
  - gcp_first
- name: "GCP layer third loop"
  needs:
  - gcp_second
- name: "AWS layer first loop"
  id: aws_first
  needs:
  - base_final
- name: "AWS layer second loop"
  id: aws_second
  needs:
  - aws_first
- name: "AWS layer third loop"
  needs:
  - aws_second
`

// testCase returns a test case with the given name, ID and needs.
func testCase(name, id string, needs ...string) api.TestCase {
	return api.TestCase{Name: name, ID: id, Needs: needs}
}

func TestBuild_ANeedWithAnEmptyIDIsNotADependency(t *testing.T) {
	// "b" has no ID, and "c" lists the empty ID: that is no testcase, not even "b"
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", ""), testCase("c", "c", "")}}

	assert.Equal(t, [][]int{nil, nil, nil}, deps(t, spec))
}

func TestDependenciesOf(t *testing.T) {
	// "c" needs "b" and "a", listed in that order, and "b" twice
	g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", "b"), testCase("c", "c", "b", "a", "b")}})

	assert.Empty(t, g.DependenciesOf(0))
	assert.Empty(t, g.DependenciesOf(1))
	assert.Equal(t, []int{0, 1}, g.DependenciesOf(2), "in ascending order, and once each")
}

// goldenCase is a testcase with golden-file assertions.
func goldenCase(name, id string, needs ...string) api.TestCase {
	testCase := testCase(name, id, needs...)
	testCase.Assertions.Diff = []api.AssertionGoldenFile{{Name: "g", Expected: "g.yaml"}}

	return testCase
}

func TestNodes_NeedsAndNeededBy(t *testing.T) {
	// first <- second <- third: second needs first, third needs second, and so third needs first, but doesn't list it
	g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("first", "first"),
		testCase("second", "second", "first"),
		testCase("third", "third", "second"),
	}})

	nodes := g.Nodes()
	require.Len(t, nodes, 3)

	assert.Equal(t, Node{Position: 0, Name: "first", ID: "first", NeededBy: []int{1}, NeededByIndirectly: []int{2}}, withoutGoldens(nodes[0]))
	assert.Equal(t, Node{Position: 1, Name: "second", ID: "second", Needs: []int{0}, NeededBy: []int{2}}, withoutGoldens(nodes[1]))
	assert.Equal(t, Node{Position: 2, Name: "third", ID: "third", Needs: []int{1}, NeedsIndirectly: []int{0}}, withoutGoldens(nodes[2]))
}

// withoutGoldens returns the node without the facts about golden files, which other tests look at.
func withoutGoldens(node Node) Node {
	node.HasGoldens, node.NeededForGoldens = false, false

	return node
}

func TestNodes(t *testing.T) {
	t.Run("positions are the ones of the file, whatever the order they run in", func(t *testing.T) {
		// c needs b, b needs a: listed out of order, they run as a, b, c
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("C", "c", "b"), testCase("B", "b", "a"), testCase("A", "a")}})
		nodes := g.Nodes()

		assert.Equal(t, []int{1}, nodes[0].Needs)
		assert.Equal(t, []int{2}, nodes[0].NeedsIndirectly)
		assert.Equal(t, []int{0}, nodes[1].NeededBy)
		assert.Equal(t, []int{0}, nodes[2].NeededByIndirectly)
		assert.Equal(t, []int{1}, nodes[2].NeededBy)
	})

	t.Run("what is listed is not indirect: a need that is also reached through another stays direct", func(t *testing.T) {
		// d lists a, b and c, and b and c need a
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{
			testCase("a", "a"),
			testCase("b", "b", "a"),
			testCase("c", "c", "a"),
			testCase("d", "d", "a", "b", "c"),
		}})
		nodes := g.Nodes()

		assert.Equal(t, []int{0, 1, 2}, nodes[3].Needs)
		assert.Empty(t, nodes[3].NeedsIndirectly)
		assert.Equal(t, []int{1, 2, 3}, nodes[0].NeededBy)
		assert.Empty(t, nodes[0].NeededByIndirectly)
	})

	t.Run("a diamond: what is needed through two paths is there once", func(t *testing.T) {
		// d needs b and c, which need a
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{
			testCase("a", "a"),
			testCase("b", "b", "a"),
			testCase("c", "c", "a"),
			testCase("d", "d", "b", "c"),
		}})
		nodes := g.Nodes()

		assert.Equal(t, []int{0}, nodes[3].NeedsIndirectly)
		assert.Equal(t, []int{3}, nodes[0].NeededByIndirectly)
	})

	t.Run("a testcase without an ID, and one without needs", func(t *testing.T) {
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("no id", "", "a")}})
		nodes := g.Nodes()

		assert.Equal(t, Node{Position: 1, Name: "no id", Needs: []int{0}}, withoutGoldens(nodes[1]))
		assert.Empty(t, nodes[0].Needs)
		assert.Empty(t, nodes[0].NeedsIndirectly)
	})

	t.Run("DependenciesOf is what the node says it needs", func(t *testing.T) {
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", "b", "a")}})

		assert.Equal(t, g.Nodes()[1].Needs, g.DependenciesOf(1))
	})

	t.Run("testcases that need each other: nothing loops, and none is reached from itself", func(t *testing.T) {
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a", "b"), testCase("b", "b", "c"), testCase("c", "c", "a")}})
		nodes := g.Nodes()

		assert.Equal(t, []int{2}, nodes[0].NeedsIndirectly, "a needs b, which needs c")
		assert.Equal(t, []int{1}, nodes[0].NeededByIndirectly, "a is needed by c, which is needed by b")

		for i, node := range nodes {
			assert.NotContains(t, node.NeedsIndirectly, i)
			assert.NotContains(t, node.NeededByIndirectly, i)
		}
	})
}

func TestNodeByID(t *testing.T) {
	g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a"),
		testCase("no id", ""),
		testCase("b", "b", "a"),
		testCase("b again", "b"),
	}})

	t.Run("the testcase with that ID", func(t *testing.T) {
		node := g.NodeByID("b")

		assert.Equal(t, "b", node.ID)
		assert.Equal(t, 2, node.Position)
		assert.Equal(t, []int{0}, node.Needs)
	})

	t.Run("none has it: the zero Node", func(t *testing.T) {
		assert.Equal(t, Node{}, g.NodeByID("missing"))
	})

	t.Run("a testcase without an ID can't be found", func(t *testing.T) {
		assert.Equal(t, Node{}, g.NodeByID(""))
	})

	t.Run("several have it: the first", func(t *testing.T) {
		assert.Equal(t, "b", g.NodeByID("b").Name)
	})
}

func TestNodes_Goldens(t *testing.T) {
	tests := []struct {
		name  string
		tests []api.TestCase
		// HasGoldens, and NeededForGoldens, of each testcase in the file
		has, neededFor []bool
	}{
		{
			name:      "no golden files anywhere: nothing is needed for them",
			tests:     []api.TestCase{testCase("a", "a"), testCase("b", "b", "a")},
			has:       []bool{false, false},
			neededFor: []bool{false, false},
		},
		{
			name:      "one with golden files needs another",
			tests:     []api.TestCase{testCase("a", "a"), goldenCase("b", "b", "a"), testCase("c", "c")},
			has:       []bool{false, true, false},
			neededFor: []bool{true, true, false},
		},
		{
			name:      "a chain that ends in one with golden files",
			tests:     []api.TestCase{testCase("a", "a"), testCase("b", "b", "a"), goldenCase("c", "c", "b")},
			has:       []bool{false, false, true},
			neededFor: []bool{true, true, true},
		},
		{
			name:      "one with golden files that others need: they are not needed for it",
			tests:     []api.TestCase{goldenCase("a", "a"), testCase("b", "b", "a")},
			has:       []bool{true, false},
			neededFor: []bool{true, false},
		},
		{
			name:      "golden files in the middle: what it needs is needed, what needs it is not",
			tests:     []api.TestCase{testCase("a", "a"), goldenCase("b", "b", "a"), testCase("c", "c", "b")},
			has:       []bool{false, true, false},
			neededFor: []bool{true, true, false},
		},
		{
			name:      "dyff counts too, and so does a testcase without an ID",
			tests:     []api.TestCase{{Name: "d", ID: "d"}, {Name: "x", Needs: nil, Assertions: api.Assertions{Dyff: []api.AssertionGoldenFile{{Name: "g"}}}}},
			has:       []bool{false, true},
			neededFor: []bool{false, true},
		},
		{
			name:      "assertions that write no golden file don't count",
			tests:     []api.TestCase{{Name: "a", ID: "a", Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "x"}}}}},
			has:       []bool{false},
			neededFor: []bool{false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := Build(&api.TestSuiteSpec{Tests: tt.tests}).Nodes()

			for i, node := range nodes {
				assert.Equal(t, tt.has[i], node.HasGoldens, "HasGoldens of %d", i)
				assert.Equal(t, tt.neededFor[i], node.NeededForGoldens, "NeededForGoldens of %d", i)
			}
		})
	}
}

func deps(t *testing.T, spec *api.TestSuiteSpec) [][]int {
	t.Helper()

	g := Build(spec)

	out := make([][]int, len(spec.Tests))
	for i := range spec.Tests {
		out[i] = g.nodes[i].Needs
	}

	return out
}

func TestBuild_NoNeeds(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", "")}}

	assert.Equal(t, [][]int{nil, nil}, deps(t, spec))
}

func TestBuild_TheTestCasesThatAreNeeded(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a"),
		testCase("b", "b", "a"),
		testCase("c", "", "b"),
	}}

	assert.Equal(t, [][]int{nil, {0}, {1}}, deps(t, spec))
}

func TestBuild_SeveralNeedsInOrderAndOnceEach(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a"),
		testCase("b", "b"),
		testCase("c", "c"),
		testCase("join", "", "c", "a", "c", "b"),
	}}

	assert.Equal(t, [][]int{nil, nil, nil, {0, 1, 2}}, deps(t, spec))
}

func TestBuild_ANeedThatIsNoTestCaseIsNotADependency(t *testing.T) {
	// The loader rejects the file before: this is only that Build doesn't fail, or invent a dependency
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", "", "nope")}}

	assert.Equal(t, [][]int{nil, nil}, deps(t, spec))
}

func TestBuild_TemplatesAreNotLookedAt(t *testing.T) {
	// A testcase that uses .Tests without listing it in needs depends on nothing: needs is the only way
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a"),
		{Name: "uses a", Inputs: api.Inputs{XR: "{{ .Tests.a.Outputs.XR }}"}},
	}}

	assert.Equal(t, [][]int{nil, nil}, deps(t, spec))
}

func TestBuild_DuplicateIDsMeanTheFirst(t *testing.T) {
	// Duplicate IDs are rejected when the file is loaded; Build keeps to the first, as the runner's maps do not
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("first", "x"), testCase("second", "x"), testCase("c", "", "x")}}

	assert.Equal(t, [][]int{nil, nil, {0}}, deps(t, spec))
}

func TestBuild_ChainedSuite(t *testing.T) {
	var spec api.TestSuiteSpec
	require.NoError(t, yaml.Unmarshal([]byte(chainedSuite), &spec))

	assert.Equal(t, [][]int{nil, {0}, {1}, {2}, {0}, {4}, {5}}, deps(t, &spec))
}

func order(t *testing.T, spec *api.TestSuiteSpec) []int {
	t.Helper()

	return Build(spec).GetOrder()
}

func TestOrder_WithoutDependenciesIsTheOrderOfTheFile(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a"), testCase("b", ""), testCase("c", "c")}}

	assert.Equal(t, []int{0, 1, 2}, order(t, spec))
}

func TestOrder_NeedsListedBeforeKeepTheOrderOfTheFile(t *testing.T) {
	var spec api.TestSuiteSpec
	require.NoError(t, yaml.Unmarshal([]byte(chainedSuite), &spec))

	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, order(t, &spec))
}

func TestOrder_ATestCaseRunsAfterTheOnesItNeeds(t *testing.T) {
	// "a" is first in the file, but needs "c" and "b": b, c, a. The independent "d" keeps its place.
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a", "c", "b"),
		testCase("b", "b"),
		testCase("c", "c"),
		testCase("d", "d"),
	}}

	assert.Equal(t, []int{1, 2, 0, 3}, order(t, spec))
}

func TestOrder_TheOrderOfTheNeedsDoesNotMatter(t *testing.T) {
	first := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a", "c", "b"), testCase("b", "b"), testCase("c", "c")}}
	second := &api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a", "b", "c"), testCase("b", "b"), testCase("c", "c")}}

	assert.Equal(t, order(t, first), order(t, second))
}

func TestOrder_WhenSeveralCanRunTheOneListedFirstDoes(t *testing.T) {
	// "x" needs "z" and is listed first: "y" (listed second) goes before "z", and "x" goes last
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("x", "x", "z"),
		testCase("y", "y"),
		testCase("z", "z"),
	}}

	assert.Equal(t, []int{1, 2, 0}, order(t, spec))
}

func TestOrder_ADependencyUnlocksTheOnesThatWaitForIt(t *testing.T) {
	// "late" is listed first and needs "mid", which needs "base", listed last
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("late", "late", "mid"),
		testCase("mid", "mid", "base"),
		testCase("other", "other"),
		testCase("base", "base"),
	}}

	assert.Equal(t, []int{2, 3, 1, 0}, order(t, spec))
}

func TestCheckCycles(t *testing.T) {
	tests := map[string]struct {
		tests []api.TestCase
		want  []string
	}{
		"none": {
			tests: []api.TestCase{testCase("a", "a"), testCase("b", "b", "a")},
		},
		"none, with a testcase listed after the one that needs it": {
			tests: []api.TestCase{testCase("a", "a", "b"), testCase("b", "b")},
		},
		"itself": {
			tests: []api.TestCase{testCase("a", "a", "a")},
			want:  []string{"needs cycle: a -> a"},
		},
		"two that need each other": {
			tests: []api.TestCase{testCase("a", "a", "b"), testCase("b", "b", "a")},
			want:  []string{"needs cycle: a -> b -> a"},
		},
		"every cycle, in the order they are listed": {
			tests: []api.TestCase{
				testCase("ok", "ok"),
				testCase("self", "self", "self"),
				testCase("a", "a", "b"),
				testCase("b", "b", "a"),
				testCase("downstream", "downstream", "a", "ok"),
			},
			want: []string{"needs cycle: self -> self", "needs cycle: a -> b -> a"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, Build(&api.TestSuiteSpec{Tests: tc.tests}).CheckCycles())
		})
	}
}

func TestCheckCycles_KeepsTheOrderOnlyWhenThereAreNoCycles(t *testing.T) {
	t.Run("no cycles: the order is kept, and GetOrder returns it", func(t *testing.T) {
		// "a" is listed first, but needs "b"
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a", "b"), testCase("b", "b"), testCase("c", "c")}})

		assert.Nil(t, g.order, "it is not sorted until the graph is checked")
		assert.Empty(t, g.CheckCycles())
		assert.Equal(t, []int{1, 0, 2}, g.order)
		assert.Equal(t, []int{1, 0, 2}, g.GetOrder())
	})

	t.Run("a graph that was not checked sorts in GetOrder", func(t *testing.T) {
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("a", "a", "b"), testCase("b", "b")}})

		assert.Equal(t, []int{1, 0}, g.GetOrder())
		assert.Nil(t, g.order, "GetOrder doesn't keep what it sorts")
	})

	t.Run("cycles: nothing is kept, because what was sorted is only the part that is not in a cycle", func(t *testing.T) {
		g := Build(&api.TestSuiteSpec{Tests: []api.TestCase{testCase("ok", "ok"), testCase("a", "a", "b"), testCase("b", "b", "a")}})

		assert.Len(t, g.CheckCycles(), 1)
		assert.Nil(t, g.order)
	})
}
