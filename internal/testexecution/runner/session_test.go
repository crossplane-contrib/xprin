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
	"io"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestJob_APanicFailsOnlyThatTestCase(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "A", ID: "a"},
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "C", ID: "c", Needs: []string{"b"}},
		{Name: "D", ID: "d"},
	}}

	r, tcRunner := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var out bytes.Buffer

	r.output = &out
	tcRunner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		if testCase.ID == "b" {
			panic("boom")
		}

		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
	}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r}, 4)
	require.NoError(t, err)

	require.Len(t, finishErrs, 1)
	require.ErrorIs(t, finishErrs[0], ErrTestsFailed)
	assert.Contains(t, out.String(), "--- FAIL: B")
	assert.Contains(t, out.String(), "panic: boom")
	assert.Contains(t, out.String(), "--- SKIP: C", "what needs it is skipped, as after any failure")
	assert.NotContains(t, out.String(), "--- FAIL: A")
	assert.NotContains(t, out.String(), "--- FAIL: D", "the others are not affected")
	assert.NotContains(t, out.String(), "goroutine", "the stack is for --debug")
}

// mockTestCaseRunner is a testCaseRunner that passes every test case, and records what the session gave it.
type mockTestCaseRunner struct {
	contexts  map[string]TestSuiteContext // by the ID of the test case
	completed map[string][]string         // the IDs of the results that the test case was given
}

func (m *mockTestCaseRunner) RunTestCase(testSuiteContext TestSuiteContext, testCase api.TestCase, completed map[string]*engine.TestCaseResult) *engine.TestCaseResult {
	m.contexts[testCase.ID] = testSuiteContext

	for id := range completed {
		m.completed[testCase.ID] = append(m.completed[testCase.ID], id)
	}

	return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false).Complete()
}

func newMockTestCaseRunner() *mockTestCaseRunner {
	return &mockTestCaseRunner{contexts: map[string]TestSuiteContext{}, completed: map[string][]string{}}
}

func TestSession_GivesTheRunnerTheTestSuiteContextAndTheResultsThatAreDone(t *testing.T) {
	// B is first in the file, and runs second: it needs A
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "B", ID: "b", Needs: []string{"a"}},
		{Name: "A", ID: "a"},
	}}
	suite := testsuite.New("/tests/a_xprin.yaml", spec)
	mock := newMockTestCaseRunner()
	s := NewSession(&testexecutionOptions.Options{}, suite, mock)
	s.output = io.Discard
	s.fs = afero.NewMemMapFs()

	require.NoError(t, s.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{s}, 1)
	require.NoError(t, err)
	require.NoError(t, finishErrs[0])

	require.Len(t, mock.contexts, 2)

	for id, testSuiteContext := range mock.contexts {
		assert.Same(t, suite, testSuiteContext.TestSuite, "%s gets the file", id)
		assert.NotEmpty(t, testSuiteContext.SavedOutputsDir, "%s gets the directory of the saved outputs", id)
	}

	assert.Equal(t, 1, mock.contexts["a"].Index, "the numbers are the ones of the order that they run in, which starts at 1")
	assert.Equal(t, 2, mock.contexts["b"].Index)
	assert.Empty(t, mock.completed["a"], "nothing is done when A runs")
	assert.Equal(t, []string{"a"}, mock.completed["b"], "A is done when B runs")
}

func TestJob_FinishReturnsNothingWhenNothingFailed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a"}}}

	r := NewSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec), newMockTestCaseRunner())
	r.output = &bytes.Buffer{}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r}, 2)
	require.NoError(t, err)

	require.Len(t, finishErrs, 1)
	require.NoError(t, finishErrs[0], "nothing failed")
}
