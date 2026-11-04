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
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
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

	r := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))

	var out bytes.Buffer

	r.output = &out
	r.r.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		if testCase.ID == "b" {
			panic("boom")
		}

		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
	}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r.Job()}, 4)
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

func TestJob_FinishReturnsNothingWhenNothingFailed(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "A", ID: "a"}}}

	r := newTestSession(&testexecutionOptions.Options{}, testsuite.New("/tests/a_xprin.yaml", spec))
	r.output = &bytes.Buffer{}
	r.r.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
		return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false)
	}

	require.NoError(t, r.Setup())

	finishErrs, err := scheduler.Run([]scheduler.Job{r.Job()}, 2)
	require.NoError(t, err)

	require.Len(t, finishErrs, 1)
	require.NoError(t, finishErrs[0], "nothing failed")
}
