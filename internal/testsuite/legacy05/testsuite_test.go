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
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

// parse returns the testsuite file with the content, as the loader reads it (with placeholders).
func parse(t *testing.T, content string) *api.TestSuiteSpec {
	t.Helper()

	var spec api.TestSuiteSpec
	require.NoError(t, yaml.Unmarshal([]byte(placeholder.Replace(content)), &spec))

	return &spec
}

func TestIsTestSuite(t *testing.T) {
	const base = `tests:
- name: "base"
  id: base
  inputs:
    xr: xr.yaml
- name: "other"
  id: other
  inputs:
    xr: xr.yaml
`

	tests := map[string]struct {
		content      string
		wantLegacy05 bool
	}{
		"no testcase uses .Tests": {content: base},
		"a testcase that uses an output and does not need it": {
			content: base + `- name: "uses base"
  inputs:
    xr: '{{ .Tests.base.Outputs.XR }}'
`,
			wantLegacy05: true,
		},
		"the same, with the need declared": {
			content: base + `- name: "uses base"
  needs:
  - base
  inputs:
    xr: '{{ .Tests.base.Outputs.XR }}'
`,
		},
		"a use without needs, and a testcase that lists needs later in the file: the file is current": {
			content: base + `- name: "uses base, without needs"
  inputs:
    xr: '{{ .Tests.base.Outputs.XR }}'
- name: "lists needs, after it"
  needs:
  - base
`,
		},
		"one of two declared: the file is migrated, the runner fails the testcase that misses one": {
			content: base + `- name: "uses both"
  needs:
  - base
  inputs:
    xr: '{{ .Tests.base.Outputs.XR }}'
  hooks:
    pre-test:
    - run: echo {{ .Tests.other.Outputs.XR }}
`,
		},
		"both declared": {
			content: base + `- name: "uses both"
  needs:
  - other
  - base
  inputs:
    xr: '{{ .Tests.base.Outputs.XR }}'
  hooks:
    pre-test:
    - run: echo {{ .Tests.other.Outputs.XR }}
`,
		},
		"index with a hyphenated ID, not declared": {
			content: `tests:
- name: "base"
  id: test-case-1
  inputs:
    xr: xr.yaml
- name: "uses it"
  inputs:
    xr: '{{ (index .Tests "test-case-1").Outputs.XR }}'
`,
			wantLegacy05: true,
		},
		"a use that can't be traced, with only some of the earlier testcases declared: the file is migrated": {
			content: base + `- name: "uses all"
  needs:
  - base
  hooks:
    pre-test:
    - run: echo {{ range .Tests }}x{{ end }}
`,
		},
		"a use that can't be traced, with all the earlier testcases declared": {
			content: base + `- name: "uses all"
  needs:
  - base
  - other
  hooks:
    pre-test:
    - run: echo {{ range .Tests }}x{{ end }}
`,
		},
		"a use that can't be traced, with nothing earlier to declare (the deprecated way all the same)": {
			content: `tests:
- name: "first"
  hooks:
    pre-test:
    - run: echo {{ range .Tests }}x{{ end }}
`,
			wantLegacy05: true,
		},
		"an ID that no testcase has (a typo): the deprecated way all the same, the testcase fails when it runs": {
			content: base + `- name: "typo"
  inputs:
    xr: '{{ .Tests.nope.Outputs.XR }}'
`,
			wantLegacy05: true,
		},
		"an ID of a testcase listed after it: the deprecated way all the same, the testcase fails when it runs": {
			content: `tests:
- name: "first"
  inputs:
    xr: '{{ .Tests.last.Outputs.XR }}'
- name: "last"
  id: last
  inputs:
    xr: xr.yaml
`,
			wantLegacy05: true,
		},
		"used through the common configuration": {
			content: `common:
  hooks:
    post-test:
    - run: echo {{ .Tests.base.Outputs.XR }}
tests:
- name: "base"
  id: base
  inputs:
    xr: xr.yaml
- name: "gets it from common"
  inputs:
    xr: xr.yaml
`,
			wantLegacy05: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.wantLegacy05, IsTestSuite(parse(t, tc.content)))
		})
	}

	t.Run("no testsuite file", func(t *testing.T) {
		assert.False(t, IsTestSuite(nil))
	})
}

func TestHasNeeds(t *testing.T) {
	assert.False(t, hasNeeds(&api.TestSuiteSpec{Tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b"}}}))
	assert.True(t, hasNeeds(&api.TestSuiteSpec{Tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", Needs: []string{"a"}}}}))
}

func TestWarning(t *testing.T) {
	out := Warning("tests/a_xprin.yaml")

	assert.Contains(t, out, "tests/a_xprin.yaml: testcases that use the outputs of other testcases (.Tests) will have to list them in `needs` from xprin 0.7.0")
	assert.Contains(t, out, "/release-0.6/docs/how-it-works.md#dependencies-between-test-cases-needs")
}
