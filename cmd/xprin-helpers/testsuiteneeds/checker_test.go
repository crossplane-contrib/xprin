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
	"testing"

	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// check loads the testsuite file with the content, as the command does, and checks it.
func check(t *testing.T, content string) testSuiteCheckResult {
	t.Helper()

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/tests/a_xprin.yaml", []byte(content), 0o644))

	testSuite, err := discovery.Load(fs, "/tests/a_xprin.yaml", discovery.Options{})
	require.NoError(t, err)
	require.NotNil(t, testSuite)

	return checkFile(testSuite.Spec)
}

func TestCheckFile_FindsTheMissingNeeds(t *testing.T) {
	tests := map[string]struct {
		content string
		want    map[int][]string // the test cases that miss needs, by position
	}{
		"a test case that uses another one and lists no needs": {
			content: `tests:
- name: "first"
  id: first
- name: "second"
  inputs:
    xr: {{ .Tests.first.Outputs.XR }}
`,
			want: map[int][]string{1: {"first"}},
		},
		"several, in the order of the file, found in a hook as well": {
			content: `tests:
- name: "a"
  id: a
- name: "b"
  id: b
- name: "join"
  inputs:
    xr: {{ .Tests.b.Outputs.XR }}
  hooks:
    pre-test:
    - run: echo {{ .Tests.a.Outputs.XR }}
`,
			want: map[int][]string{2: {"a", "b"}},
		},
		"only what is not listed": {
			content: `tests:
- name: "a"
  id: a
- name: "b"
  id: b
- name: "join"
  needs: [a]
  inputs:
    xr: {{ .Tests.b.Outputs.XR }}
    other: {{ .Tests.a.Outputs.XR }}
`,
			want: map[int][]string{2: {"b"}},
		},
		"found in the common configuration": {
			content: `common:
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
tests:
- name: "a"
  id: a
  inputs:
    xr: x.yaml
- name: "b"
`,
			want: map[int][]string{1: {"a"}},
		},
		"a test case that is listed before the one it uses": {
			content: `tests:
- name: "b"
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
- name: "a"
  id: a
`,
			want: map[int][]string{0: {"a"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res := check(t, tc.content)

			assert.False(t, res.requiresAPerson())

			got := map[int][]string{}

			for i, c := range res {
				if len(c.missing) > 0 {
					got[i] = c.missing
				}
			}

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCheckFile_NothingMissing(t *testing.T) {
	tests := map[string]string{
		"no test case uses .Tests": `tests:
- name: "a"
  id: a
- name: "b"
  inputs:
    xr: xr.yaml
`,
		"a file that is complete": `tests:
- name: "a"
  id: a
- name: "b"
  needs:
  - a
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
`,
		"needs that are not used through .Tests are left alone": `tests:
- name: "a"
  id: a
- name: "b"
  needs:
  - a
  inputs:
    xr: xr.yaml
`,
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			res := check(t, content)

			assert.False(t, res.missesNeeds())
			assert.False(t, res.requiresAPerson())
		})
	}
}

func TestCheckFile_NeedsAPerson(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantReview string
	}{
		"a use that can't be traced": {
			content: `tests:
- name: "a"
  id: a
- name: "lists them"
  hooks:
    pre-test:
    - run: echo {{ range .Tests }}x{{ end }}
`,
			wantReview: `uses .Tests in a way that can't be traced`,
		},
		"an id that no test case has": {
			content: `tests:
- name: "a"
  id: a
- name: "typo"
  inputs:
    xr: {{ .Tests.nope.Outputs.XR }}
`,
			wantReview: `uses .Tests.nope, but no test case has that ID`,
		},
		"its own outputs": {
			content: `tests:
- name: "a"
  id: a
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
`,
			wantReview: `uses its own outputs (.Tests.a)`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res := check(t, tc.content)

			var problems []string

			for _, c := range res {
				problems = append(problems, c.problems...)
			}

			require.Len(t, problems, 1)
			assert.Contains(t, problems[0], tc.wantReview)
		})
	}
}

func TestCheckFile_TestCases(t *testing.T) {
	// a uses nothing, b lists what it uses, c misses something, and d uses what can't be found
	const content = `tests:
- name: "a"
  id: a
- name: "b"
  id: b
  needs:
  - a
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
- name: "c"
  inputs:
    xr: {{ .Tests.b.Outputs.XR }}
- name: "d"
  inputs:
    xr: {{ .Tests.nope.Outputs.XR }}
`

	res := check(t, content)

	require.Len(t, res, 4)

	assert.Equal(t, testCaseCheckResult{name: "a"}, res[0], "it uses nothing")
	assert.Equal(t, testCaseCheckResult{name: "b", usesTests: true}, res[1], "it lists what it uses")
	assert.Equal(t, testCaseCheckResult{name: "c", usesTests: true, missing: []string{"b"}}, res[2])
	assert.Equal(t, testCaseCheckResult{name: "d", usesTests: true, problems: []string{"uses .Tests.nope, but no test case has that ID"}}, res[3])
}

func TestCheckFile_AUseThatCantBeTracedAndNeedsThatAPersonListed(t *testing.T) {
	// It can't tell which test cases it uses, but a person listed some: that is not a failure, as it can't be checked
	const content = `tests:
- name: "a"
  id: a
- name: "lists them"
  needs:
  - a
  hooks:
    pre-test:
    - run: echo {{ range .Tests }}x{{ end }}
`

	res := check(t, content)

	assert.False(t, res.requiresAPerson())
	assert.False(t, res.missesNeeds())
	assert.Empty(t, res[0].skipReason, "a uses nothing: there is nothing to say")
	assert.Contains(t, res[1].skipReason, "uses .Tests in a way that can't be traced")
	assert.Contains(t, res[1].skipReason, "the needs that it lists can't be checked")
}

func TestCheckFile_NeedsThatWouldMakeACycle(t *testing.T) {
	// "a" needs "b" already, and "b" uses the outputs of "a": listing it would make them need each other
	const content = `tests:
- name: "a"
  id: a
  needs:
  - b
- name: "b"
  id: b
  inputs:
    xr: {{ .Tests.a.Outputs.XR }}
`

	res := check(t, content)

	assert.Empty(t, res[0].problems, "a")
	require.Len(t, res[1].problems, 1, "b, which misses the needs")
	assert.Contains(t, res[1].problems[0], "would make the test cases need each other (needs cycle: a -> b -> a)")
	assert.False(t, res.missesNeeds())
}
