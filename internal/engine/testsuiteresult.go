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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TestSuiteResult is the result of running a testsuite file: its test case results, overall status and duration.
// Its methods are safe for concurrent use, so test cases running at the same time can add their results.
// What is set before the test cases run (the file, when it was created, the formatting flag and the working directory) is
// not guarded, and not changed after that.
type TestSuiteResult struct {
	// What the constructor sets, and what the methods keep, so that nothing else changes it: the file, when the result
	// was created, the formatting flag, the test case results in the order they were added, the overall status
	// (StatusPass() or StatusFail()), when the first test case started and the last one ended, and the duration that
	// Complete or CompleteSinceCreated sets.
	filePath  string
	createdAt time.Time
	verbose   bool

	// mu guards results, status, firstStart, lastEnd and duration.
	mu         sync.Mutex
	results    []TestCaseResult
	status     Status
	firstStart time.Time // when the first test case of the file started
	lastEnd    time.Time // when the last test case of the file ended
	duration   time.Duration

	workingDir string // Set by SetWorkingDir; used to relativise the file path for display.
}

// NewTestSuiteResult creates a new test suite result.
func NewTestSuiteResult(filePath string, verbose bool) *TestSuiteResult {
	return &TestSuiteResult{
		filePath:  filePath,
		status:    StatusPass(), // Default to pass
		createdAt: time.Now(),
		verbose:   verbose,
	}
}

// SetWorkingDir sets the directory that the file path is shown relative to.
func (tsr *TestSuiteResult) SetWorkingDir(workingDir string) {
	tsr.workingDir = workingDir
}

// AddResult adds a test case result to the test suite.
func (tsr *TestSuiteResult) AddResult(result *TestCaseResult) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	tsr.results = append(tsr.results, *result)

	// The file takes from when its first test case started to when its last one ended
	if !result.StartTime.IsZero() {
		if tsr.firstStart.IsZero() || result.StartTime.Before(tsr.firstStart) {
			tsr.firstStart = result.StartTime
		}

		if end := result.StartTime.Add(result.Duration); end.After(tsr.lastEnd) {
			tsr.lastEnd = end
		}
	}

	// Update overall status if any test failed
	if result.Status == StatusFail() {
		tsr.status = StatusFail()
	}
}

// Complete finalizes the test suite result with total duration and returns the result for chaining.
func (tsr *TestSuiteResult) Complete() *TestSuiteResult {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	tsr.duration = 0
	if !tsr.firstStart.IsZero() {
		tsr.duration = tsr.lastEnd.Sub(tsr.firstStart)
	}

	return tsr
}

// CompleteSinceCreated finalizes the test suite result like Complete, with the time since the result was created as its
// duration, whatever the test cases took.
func (tsr *TestSuiteResult) CompleteSinceCreated() *TestSuiteResult {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	tsr.duration = time.Since(tsr.createdAt)

	return tsr
}

// Print the file summary in Go test format.
func (tsr *TestSuiteResult) Print(w io.Writer) {
	// Copy what the lock guards and release it, so the printing below doesn't hold it while others add results.
	tsr.mu.Lock()
	status, duration := tsr.status, tsr.duration
	tsr.mu.Unlock()

	// Convert absolute paths to relative paths when possible (matches Go's testing package behavior).
	// Prefer the stored working directory; fall back to os.Getwd() when not set (e.g. in tests).
	displayPath := tsr.filePath

	pwd := tsr.workingDir
	if pwd == "" {
		pwd, _ = os.Getwd()
	}

	if pwd != "" {
		if rel, err := filepath.Rel(pwd, tsr.filePath); err == nil && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
	}

	if status == StatusFail() {
		fmt.Fprintf(w, "%s\n%s\t%s\t%.3fs\n", StatusFail().Value, StatusFail().Value, displayPath, duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	} else {
		if tsr.verbose {
			fmt.Fprintln(w, StatusPass().Value) //nolint:errcheck // output function, error handling not practical
		}

		fmt.Fprintf(w, "ok\t%s\t%.3fs\n", displayPath, duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	}
}

// HasFailures returns true if any test failed.
func (tsr *TestSuiteResult) HasFailures() bool {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	return tsr.status == StatusFail()
}

// GetCompletedTests returns a map of test ID to test case result for completed tests.
// It is a snapshot: results added later are not in it.
// A result is never changed after AddResult, so the returned pointers are safe to read while others are added.
func (tsr *TestSuiteResult) GetCompletedTests() map[string]*TestCaseResult {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	completed := make(map[string]*TestCaseResult)

	for i := range tsr.results {
		result := &tsr.results[i]
		if result.ID != "" {
			completed[result.ID] = result
		}
	}

	return completed
}
