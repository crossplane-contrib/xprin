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
	"time"
)

// TestSuiteResult represents the result of running a test suite file.
type TestSuiteResult struct {
	WorkingDir string // Set by the runner from Options.WorkingDir; used to relativise the file path for display.

	// What the constructor sets, and what the methods keep, so that nothing else changes it: the file, when the work
	// on it started, the formatting flag, the test case results in the order they were added, the overall status
	// (StatusPass() or StatusFail()), and the duration that Complete sets.
	filePath  string
	startTime time.Time
	verbose   bool
	results   []TestCaseResult
	status    Status
	duration  time.Duration
}

// NewTestSuiteResult creates a new test suite result.
func NewTestSuiteResult(filePath string, verbose bool) *TestSuiteResult {
	return &TestSuiteResult{
		filePath:  filePath,
		status:    StatusPass(), // Default to pass
		startTime: time.Now(),
		verbose:   verbose,
	}
}

// AddResult adds a test case result to the test suite.
func (tsr *TestSuiteResult) AddResult(result *TestCaseResult) {
	tsr.results = append(tsr.results, *result)

	// Update overall status if any test failed
	if result.Status == StatusFail() {
		tsr.status = StatusFail()
	}
}

// Complete finalizes the test suite result with total duration and returns the result for chaining.
func (tsr *TestSuiteResult) Complete() *TestSuiteResult {
	tsr.duration = time.Since(tsr.startTime)
	return tsr
}

// Print the file summary in Go test format.
func (tsr *TestSuiteResult) Print(w io.Writer) {
	// Convert absolute paths to relative paths when possible (matches Go's testing package behavior).
	// Prefer the stored WorkingDir; fall back to os.Getwd() when not set (e.g. in tests).
	displayPath := tsr.filePath

	pwd := tsr.WorkingDir
	if pwd == "" {
		pwd, _ = os.Getwd()
	}

	if pwd != "" {
		if rel, err := filepath.Rel(pwd, tsr.filePath); err == nil && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
	}

	if tsr.status == StatusFail() {
		fmt.Fprintf(w, "%s\n%s\t%s\t%.3fs\n", StatusFail().Value, StatusFail().Value, displayPath, tsr.duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	} else {
		if tsr.verbose {
			fmt.Fprintln(w, StatusPass().Value) //nolint:errcheck // output function, error handling not practical
		}

		fmt.Fprintf(w, "ok\t%s\t%.3fs\n", displayPath, tsr.duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	}
}

// HasFailures returns true if any test failed.
func (tsr *TestSuiteResult) HasFailures() bool {
	return tsr.status == StatusFail()
}

// GetCompletedTests returns a map of test ID to test case result for completed tests.
func (tsr *TestSuiteResult) GetCompletedTests() map[string]*TestCaseResult {
	completed := make(map[string]*TestCaseResult)

	for i := range tsr.results {
		result := &tsr.results[i]
		if result.ID != "" {
			completed[result.ID] = result
		}
	}

	return completed
}
