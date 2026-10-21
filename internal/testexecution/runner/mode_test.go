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

package runner

import (
	"bytes"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestNewRunnerMode(t *testing.T) {
	suite := testsuite.New("/tests/a_xprin.yaml", &api.TestSuiteSpec{})

	t.Run("xprin test", func(t *testing.T) {
		assert.IsType(t, testMode{}, newRunnerMode(NewRunner(&testexecutionOptions.Options{}, suite)))
	})

	t.Run("xprin update-goldens", func(t *testing.T) {
		assert.IsType(t, updateGoldensMode{}, newRunnerMode(NewRunner(&testexecutionOptions.Options{UpdateGoldens: true}, suite)))
	})
}

func TestGetMode(t *testing.T) {
	suite := testsuite.New("/tests/a_xprin.yaml", &api.TestSuiteSpec{})

	t.Run("none set, as when runTestCase is called on its own: xprin test, whatever the options", func(t *testing.T) {
		assert.IsType(t, testMode{}, NewRunner(&testexecutionOptions.Options{UpdateGoldens: true}, suite).getMode())
	})

	t.Run("the one that was set", func(t *testing.T) {
		r := NewRunner(&testexecutionOptions.Options{}, suite)
		r.mode = updateGoldensMode{r}

		assert.IsType(t, updateGoldensMode{}, r.getMode())
	})

	t.Run("RunTests sets it, from the options", func(t *testing.T) {
		for options, want := range map[*testexecutionOptions.Options]runnerMode{
			{}:                    testMode{},
			{UpdateGoldens: true}: updateGoldensMode{},
		} {
			r := NewRunner(options, suite)
			r.output = &bytes.Buffer{}

			require.NoError(t, r.RunTests())
			assert.IsType(t, want, r.mode)
		}
	})
}

func TestModeSkips(t *testing.T) {
	withDiff := api.TestCase{Assertions: api.Assertions{Diff: []api.AssertionGoldenFile{{Name: "g"}}}}
	withDyff := api.TestCase{Assertions: api.Assertions{Dyff: []api.AssertionGoldenFile{{Name: "g"}}}}
	withXprin := api.TestCase{Assertions: api.Assertions{Xprin: []api.AssertionXprin{{Name: "a"}}}}

	t.Run("xprin test skips nothing", func(t *testing.T) {
		for _, testCase := range []api.TestCase{{}, withDiff, withDyff, withXprin} {
			assert.False(t, testMode{}.skips(testCase))
		}
	})

	t.Run("xprin update-goldens skips what has no golden files to write", func(t *testing.T) {
		assert.True(t, updateGoldensMode{}.skips(api.TestCase{}))
		assert.True(t, updateGoldensMode{}.skips(withXprin), "assertions that write no golden file are no reason to run")
		assert.False(t, updateGoldensMode{}.skips(withDiff))
		assert.False(t, updateGoldensMode{}.skips(withDyff))
	})
}

func TestUpdateGoldensModeFinish_WritesTheGoldenFiles(t *testing.T) {
	r := newUpdateGoldensRunner(afero.NewMemMapFs())
	require.NoError(t, afero.WriteFile(r.fs, "/tmp/rendered.yaml", []byte("kind: X\n"), 0o600))

	testCase := api.TestCase{Assertions: api.Assertions{Diff: []api.AssertionGoldenFile{{Name: "g", Expected: "/golden.yaml"}}}}
	result := engine.NewTestCaseResult("t", "", false, false, false, false, false)
	result.Outputs.Render = "/tmp/rendered.yaml"

	got := updateGoldensMode{r}.runPhasesAfterRender(testCase, &testCaseEnv{}, nil, result)

	assert.Equal(t, engine.StatusPass(), got.Status)
	assert.Equal(t, []string{"/golden.yaml"}, got.WrittenGoldenFiles)

	written, err := afero.ReadFile(r.fs, "/golden.yaml")
	require.NoError(t, err)
	assert.Equal(t, "kind: X\n", string(written))
}
