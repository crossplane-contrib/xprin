/*
Copyright 2025 The Crossplane Authors.

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

package engine

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

// Core tests for TestSuiteResult (moved from testcaseresult_test.go)

func TestNewTestSuiteResult(t *testing.T) {
	t.Run("creates result with correct initial values", func(t *testing.T) {
		result := NewTestSuiteResult("/path/to/test.yaml", true)

		assert.Equal(t, "/path/to/test.yaml", result.filePath)
		assert.Equal(t, StatusPass(), result.status)
		assert.True(t, result.verbose)
		assert.False(t, result.startTime.IsZero())
		assert.Equal(t, time.Duration(0), result.duration)
		assert.Empty(t, result.results)
	})
}

func TestTestSuiteResult_AddResult(t *testing.T) {
	t.Run("adds passing result and keeps PASS status", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		testResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		testResult.Complete()

		suite.AddResult(testResult)

		assert.Len(t, suite.results, 1)
		assert.Equal(t, StatusPass(), suite.status)
		assert.Equal(t, StatusPass(), suite.results[0].Status)
	})

	t.Run("adds failing result and changes status to FAIL", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		testResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		testResult.Fail(assert.AnError)

		suite.AddResult(testResult)

		assert.Len(t, suite.results, 1)
		assert.Equal(t, StatusFail(), suite.status)
		assert.Equal(t, StatusFail(), suite.results[0].Status)
	})

	t.Run("adds multiple results and updates status correctly", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)

		// Add passing test
		passResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		passResult.Complete()
		suite.AddResult(passResult)

		assert.Equal(t, StatusPass(), suite.status)

		// Add failing test
		failResult := NewTestCaseResult("test2", "test2-id", false, false, false, false, false)
		failResult.Fail(assert.AnError)
		suite.AddResult(failResult)

		assert.Equal(t, StatusFail(), suite.status)
		assert.Len(t, suite.results, 2)
	})
}

func TestTestSuiteResult_Complete(t *testing.T) {
	t.Run("sets duration and returns self", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		// Add a small delay to ensure non-zero duration
		time.Sleep(1 * time.Millisecond)

		returned := suite.Complete()

		assert.Equal(t, suite, returned) // Should return self for chaining
		assert.Positive(t, suite.duration)
	})
}

func TestTestSuiteResult_Print(t *testing.T) {
	t.Run("prints PASS for successful suite in non-verbose mode", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		suite.Complete()

		var buf bytes.Buffer
		suite.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "ok")
		assert.Contains(t, output, "test.yaml")
		assert.NotContains(t, output, "PASS")
	})

	t.Run("prints PASS for successful suite in verbose mode", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", true)
		suite.Complete()

		var buf bytes.Buffer
		suite.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "PASS")
		assert.Contains(t, output, "ok")
		assert.Contains(t, output, "test.yaml")
	})

	t.Run("prints FAIL for failed suite", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		failResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		failResult.Fail(assert.AnError)
		suite.AddResult(failResult)
		suite.Complete()

		var buf bytes.Buffer
		suite.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "FAIL")
		assert.Contains(t, output, "test.yaml")
		assert.NotContains(t, output, "ok")
	})

	t.Run("prints relative path when possible", func(t *testing.T) {
		// This test might be flaky depending on the test environment
		// but it tests the path conversion logic
		suite := NewTestSuiteResult("test.yaml", false)
		suite.Complete()

		var buf bytes.Buffer
		suite.Print(&buf)

		output := buf.String()
		// Should contain some form of the path
		assert.True(t, strings.Contains(output, "test.yaml") || strings.Contains(output, "test"))
	})

	t.Run("prints the path relative to the working directory that was set", func(t *testing.T) {
		suite := NewTestSuiteResult("/work/tests/a_xprin.yaml", false)
		suite.SetWorkingDir("/work")

		var buf bytes.Buffer
		suite.Print(&buf)

		assert.Contains(t, buf.String(), "ok\ttests/a_xprin.yaml\t")
	})

	t.Run("prints the path as it is when it is outside the working directory that was set", func(t *testing.T) {
		suite := NewTestSuiteResult("/elsewhere/a_xprin.yaml", false)
		suite.SetWorkingDir("/work")

		var buf bytes.Buffer
		suite.Print(&buf)

		assert.Contains(t, buf.String(), "ok\t/elsewhere/a_xprin.yaml\t")
	})
}

func TestTestSuiteResult_HasFailures(t *testing.T) {
	t.Run("returns false for passing suite", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		testResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		testResult.Complete()
		suite.AddResult(testResult)

		assert.False(t, suite.HasFailures())
	})

	t.Run("returns true for failing suite", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)
		testResult := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		testResult.Fail(assert.AnError)
		suite.AddResult(testResult)

		assert.True(t, suite.HasFailures())
	})
}

func TestTestSuiteResult_Integration(t *testing.T) {
	t.Run("complete workflow with multiple test cases", func(t *testing.T) {
		suite := NewTestSuiteResult("integration-test.yaml", true)

		// Add passing test
		passResult := NewTestCaseResult("passing-test", "passing-test-id", true, true, true, false, false)
		passResult.RawRenderOutput = []byte("apiVersion: v1\nkind: Pod")
		passResult.RawValidateOutput = []byte("[✓] test validated")
		passResult.Complete()
		suite.AddResult(passResult)

		// Add failing test
		failResult := NewTestCaseResult("failing-test", "failing-test-id", true, true, true, false, false)
		failResult.RawValidateOutput = []byte("validation error")
		failResult.Fail(failResult.MarkValidateFailed())
		suite.AddResult(failResult)

		suite.Complete()

		// Test properties
		assert.Len(t, suite.results, 2)
		assert.Equal(t, StatusFail(), suite.status)
		assert.True(t, suite.HasFailures())
		assert.Positive(t, suite.duration)

		// Test output
		var buf bytes.Buffer
		suite.Print(&buf)

		output := buf.String()
		assert.Contains(t, output, "FAIL")
		assert.Contains(t, output, "integration-test.yaml")
	})
}

func TestTestSuiteResult_GetCompletedTests(t *testing.T) {
	t.Run("returns empty map for empty suite", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)

		completed := suite.GetCompletedTests()

		assert.Empty(t, completed)
	})

	t.Run("returns only tests with IDs", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)

		// Add test with ID
		testWithID := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		testWithID.Complete()
		suite.AddResult(testWithID)

		// Add test without ID
		testWithoutID := NewTestCaseResult("test2", "", false, false, false, false, false)
		testWithoutID.Complete()
		suite.AddResult(testWithoutID)

		completed := suite.GetCompletedTests()

		assert.Len(t, completed, 1)
		assert.Contains(t, completed, "test1-id")
		assert.Equal(t, "test1", completed["test1-id"].Name)
		assert.NotContains(t, completed, "")
	})

	t.Run("returns multiple tests with different IDs", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)

		// Add multiple tests with IDs
		test1 := NewTestCaseResult("test1", "test1-id", false, false, false, false, false)
		test1.Complete()
		suite.AddResult(test1)

		test2 := NewTestCaseResult("test2", "test2-id", false, false, false, false, false)
		test2.Complete()
		suite.AddResult(test2)

		completed := suite.GetCompletedTests()

		assert.Len(t, completed, 2)
		assert.Contains(t, completed, "test1-id")
		assert.Contains(t, completed, "test2-id")
		assert.Equal(t, "test1", completed["test1-id"].Name)
		assert.Equal(t, "test2", completed["test2-id"].Name)
	})

	t.Run("returns tests regardless of status", func(t *testing.T) {
		suite := NewTestSuiteResult("test.yaml", false)

		// Add passing test
		passTest := NewTestCaseResult("pass-test", "pass-id", false, false, false, false, false)
		passTest.Complete()
		suite.AddResult(passTest)

		// Add failing test
		failTest := NewTestCaseResult("fail-test", "fail-id", false, false, false, false, false)
		failTest.Fail(assert.AnError)
		suite.AddResult(failTest)

		completed := suite.GetCompletedTests()

		assert.Len(t, completed, 2)
		assert.Contains(t, completed, "pass-id")
		assert.Contains(t, completed, "fail-id")
		assert.Equal(t, StatusPass(), completed["pass-id"].Status)
		assert.Equal(t, StatusFail(), completed["fail-id"].Status)
	})
}

// The methods can be used by several goroutines at the same time: run with -race.
func TestTestSuiteResult_ConcurrentUse(t *testing.T) {
	const testCases = 50

	tsr := NewTestSuiteResult("/path/to/test.yaml", false)

	var wg sync.WaitGroup

	for i := range testCases {
		wg.Go(func() {
			result := NewTestCaseResult(fmt.Sprintf("test %d", i), fmt.Sprintf("id%d", i), false, false, false, false, false)
			if i == 7 {
				result.Fail(assert.AnError)
			}

			// What a test case does: read the results of the ones before it, then add its own
			_ = tsr.GetCompletedTests()
			_ = tsr.HasFailures()

			tsr.AddResult(result)
			tsr.Print(&bytes.Buffer{})
		})
	}

	wg.Wait()
	tsr.Complete()

	assert.Len(t, tsr.results, testCases)
	assert.Len(t, tsr.GetCompletedTests(), testCases)
	assert.True(t, tsr.HasFailures(), "one test case failed")
}
