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
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestCmd_Parse(t *testing.T) {
	var cli struct {
		Graph Cmd `cmd:""`
	}

	parser, err := kong.New(&cli)
	require.NoError(t, err)

	_, err = parser.Parse([]string{"graph", "-q", "tests/...", "other_xprin.yaml"})
	require.NoError(t, err)

	assert.Equal(t, []string{"tests/...", "other_xprin.yaml"}, cli.Graph.Targets)
	assert.True(t, cli.Graph.Quiet)
}

func TestGraphTargets(t *testing.T) {
	const chained = `tests:
- name: "first"
  id: first
  inputs:
    xr: xr.yaml
- name: "second"
  needs:
  - first
  inputs:
    xr: {{ .Tests.first.Outputs.XR }}
`

	const independent = `tests:
- name: "alone"
  inputs:
    xr: xr.yaml
`

	t.Run("writes a tree per testsuite file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/tests/a_xprin.yaml", []byte(chained), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/tests/b_xprin.yaml", []byte(independent), 0o644))

		var out bytes.Buffer

		require.NoError(t, graphTargets(fs, &out, []string{"/tests/"}, false))

		want := `/tests/a_xprin.yaml
└── first [first]
    └── second

/tests/b_xprin.yaml
└── alone
`

		assert.Equal(t, want, out.String())
	})

	t.Run("a reference to a test case that is not there is not a dependency", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/tests/a_xprin.yaml", []byte(`tests:
- name: "first"
  id: first
  inputs:
    xr: xr.yaml
- name: "second"
  inputs:
    xr: {{ .Tests.nope.Outputs.XR }}
`), 0o644))

		var out bytes.Buffer

		// It fails by itself when it runs, so the graph has nothing to say about it.
		require.NoError(t, graphTargets(fs, &out, []string{"/tests/a_xprin.yaml"}, false))
		assert.Equal(t, "/tests/a_xprin.yaml\n├── first [first]\n└── second\n", out.String())
	})

	t.Run("a file without test cases is skipped", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/tests/a_xprin.yaml", []byte("tests: []\n"), 0o644))

		var out bytes.Buffer

		require.NoError(t, graphTargets(fs, &out, []string{"/tests/a_xprin.yaml"}, true))
		assert.Empty(t, out.String())
	})
}
