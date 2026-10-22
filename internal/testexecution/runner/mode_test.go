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
	cp "github.com/otiai10/copy"
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

func TestTestModeSkipsNothing(t *testing.T) {
	assert.False(t, testMode{}.skips(api.TestCase{}))
	assert.False(t, testMode{}.skips(goldenCase("a")))
}

func TestUpdateGoldensModeSkips(t *testing.T) {
	tests := []struct {
		name  string
		tests []api.TestCase
		// whether each test case of the file is skipped
		want []bool
	}{
		{
			name:  "no golden files: everything is skipped",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a")},
			want:  []bool{true, true},
		},
		{
			name:  "one with golden files, and what it needs",
			tests: []api.TestCase{plainCase("a"), goldenCase("b", "a"), plainCase("c")},
			want:  []bool{false, false, true},
		},
		{
			name:  "what needs one with golden files is not needed for it",
			tests: []api.TestCase{goldenCase("a"), plainCase("b", "a")},
			want:  []bool{false, true},
		},
		{
			name:  "a chain that ends in one with golden files runs all of it",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a"), goldenCase("c", "b")},
			want:  []bool{false, false, false},
		},
		{
			name:  "needed only by one that is skipped: skipped too",
			tests: []api.TestCase{plainCase("a"), plainCase("b", "a")},
			want:  []bool{true, true},
		},
		{
			name:  "listed after the one that needs it",
			tests: []api.TestCase{goldenCase("x", "y"), plainCase("z"), plainCase("y")},
			want:  []bool{false, true, false},
		},
		{
			name:  "without an ID: only its own golden files matter",
			tests: []api.TestCase{{Name: "no id"}, {Name: "no id, golden files", Assertions: goldenCase("g").Assertions}},
			want:  []bool{true, false},
		},
		{
			name:  "a legacy file has no needs: only the ones with golden files run",
			tests: append(legacySpec().Tests, goldenCase("d")),
			want:  []bool{true, true, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRunner(&testexecutionOptions.Options{UpdateGoldens: true}, testsuite.New("/tests/a_xprin.yaml", &api.TestSuiteSpec{Tests: tt.tests}))
			got := make([]bool, len(tt.tests))

			for i, testCase := range tt.tests {
				got[i] = updateGoldensMode{r}.skips(testCase)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateGoldensModeRunPhasesAfterRender(t *testing.T) {
	newRunner := func() (*Runner, *[]string) {
		r := newUpdateGoldensRunner(afero.NewMemMapFs())
		r.testSuiteSavedOutputsDir = "/saved-outputs"

		var copied []string

		r.copy = func(src, dest string, _ ...cp.Options) error {
			copied = append(copied, src+" -> "+dest)

			return nil
		}

		return r, &copied
	}

	t.Run("it writes the golden files, and saves the outputs for the ones that need this one", func(t *testing.T) {
		r, copied := newRunner()
		require.NoError(t, afero.WriteFile(r.fs, "/tmp/outputs/rendered.yaml", []byte("kind: X\n"), 0o600))

		testCase := goldenCase("a")
		result := engine.NewTestCaseResult("a", "a", false, false, false, false, false)
		result.Outputs.Render = "/tmp/outputs/rendered.yaml"

		got := updateGoldensMode{r}.runPhasesAfterRender(testCase, &testCaseEnv{outputsDir: "/tmp/outputs"}, nil, result)

		assert.Equal(t, engine.StatusPass(), got.Status)
		assert.Equal(t, []string{"a.yaml"}, writtenGoldenFiles(got))
		assert.Equal(t, []string{"/tmp/outputs -> /saved-outputs/a"}, *copied)
		assert.Equal(t, "/saved-outputs/a/rendered.yaml", got.Outputs.Render)

		written, err := afero.ReadFile(r.fs, "a.yaml")
		require.NoError(t, err)
		assert.Equal(t, "kind: X\n", string(written))
	})

	t.Run("a test case that writes nothing and has no ID saves nothing", func(t *testing.T) {
		r, copied := newRunner()
		result := engine.NewTestCaseResult("n", "", false, false, false, false, false)

		got := updateGoldensMode{r}.runPhasesAfterRender(api.TestCase{Name: "n"}, &testCaseEnv{outputsDir: "/tmp/outputs"}, nil, result)

		assert.Equal(t, engine.StatusPass(), got.Status)
		assert.Empty(t, *copied)
	})

	t.Run("a failure to write a golden file is the result, and nothing is saved", func(t *testing.T) {
		r, copied := newRunner()

		testCase := goldenCase("a")
		testCase.Assertions.Diff[0].Resource = "Kind/missing"
		result := engine.NewTestCaseResult("a", "a", false, false, false, false, false)

		got := updateGoldensMode{r}.runPhasesAfterRender(testCase, &testCaseEnv{outputsDir: "/tmp/outputs"}, nil, result)

		assert.Equal(t, engine.StatusFail(), got.Status)
		assert.Empty(t, *copied)
	})
}
