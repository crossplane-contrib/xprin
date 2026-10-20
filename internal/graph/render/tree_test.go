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

package render

import (
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
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

func tree(t *testing.T, spec *api.TestSuiteSpec) string {
	t.Helper()

	var b strings.Builder
	require.NoError(t, Tree(&b, testsuite.New("suite_xprin.yaml", spec)))

	return b.String()
}

func TestTree_ChainedSuite(t *testing.T) {
	var spec api.TestSuiteSpec
	require.NoError(t, yaml.Unmarshal([]byte(chainedSuite), &spec))

	want := `suite_xprin.yaml
└── Base layer final loop [base_final]
    ├── GCP layer first loop [gcp_first]
    │   └── GCP layer second loop [gcp_second]
    │       └── GCP layer third loop
    └── AWS layer first loop [aws_first]
        └── AWS layer second loop [aws_second]
            └── AWS layer third loop
`

	assert.Equal(t, want, tree(t, &spec))
}

func TestTree_IndependentTestCasesAreAllAtTheTopLevel(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("one", ""),
		testCase("two", "two"),
		testCase("three", ""),
	}}

	want := `suite_xprin.yaml
├── one
├── two [two]
└── three
`

	assert.Equal(t, want, tree(t, spec))
}

func TestTree_TestCaseWaitingForSeveralIsExpandedOnce(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("a", "a"),
		testCase("b", "b"),
		testCase("join", "join", "a", "b"),
		testCase("after join", "", "join"),
	}}

	want := `suite_xprin.yaml
├── a [a]
│   └── join [join]
│       └── after join
└── b [b]
    └── join [join] (*)

(*) needs more than one test case, listed above
`

	assert.Equal(t, want, tree(t, spec))
}

func TestTree_ATestCaseCanNeedOneListedAfterIt(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		testCase("needs the last", "", "base"),
		testCase("alone", ""),
		testCase("base", "base"),
	}}

	want := `suite_xprin.yaml
├── alone
└── base [base]
    └── needs the last
`

	assert.Equal(t, want, tree(t, spec))
}
