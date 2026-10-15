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

func deps(t *testing.T, spec *api.TestSuiteSpec) [][]int {
	t.Helper()

	g := Build(spec)

	out := make([][]int, len(spec.Tests))
	for i := range spec.Tests {
		out[i] = g.deps[i]
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

	return Build(spec).Order()
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
