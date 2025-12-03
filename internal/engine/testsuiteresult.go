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
	FilePath  string
	Results   []TestCaseResult
	Duration  time.Duration
	Status    Status // StatusPass or StatusFail - overall status
	StartTime time.Time
	Verbose   bool // Formatting flag for output
}

// NewTestSuiteResult creates a new test suite result.
func NewTestSuiteResult(filePath string, verbose bool) *TestSuiteResult {
	return &TestSuiteResult{
		FilePath:  filePath,
		Status:    StatusPass, // Default to pass
		StartTime: time.Now(),
		Verbose:   verbose,
	}
}

// AddResult adds a test case result to the test suite.
func (tsr *TestSuiteResult) AddResult(result *TestCaseResult) {
	tsr.Results = append(tsr.Results, *result)

	// Update overall status if any test failed
	if result.Status == StatusFail {
		tsr.Status = StatusFail
	}
}

// Complete finalizes the test suite result with total duration and returns the result for chaining.
func (tsr *TestSuiteResult) Complete() *TestSuiteResult {
	tsr.Duration = time.Since(tsr.StartTime)
	return tsr
}

// Print the file summary in Go test format.
func (tsr *TestSuiteResult) Print(w io.Writer) {
	// Convert absolute paths to relative paths when possible (matches Go's testing package behavior)
	displayPath := tsr.FilePath
	if pwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(pwd, tsr.FilePath); err == nil && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
	}

	if tsr.Status == StatusFail {
		fmt.Fprintf(w, "%s\n%s\t%s\t%.3fs\n", string(StatusFail), string(StatusFail), displayPath, tsr.Duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	} else {
		if tsr.Verbose {
			fmt.Fprintln(w, string(StatusPass)) //nolint:errcheck // output function, error handling not practical
		}

		fmt.Fprintf(w, "ok\t%s\t%.3fs\n", displayPath, tsr.Duration.Seconds()) //nolint:errcheck // output function, error handling not practical
	}
}

// HasFailures returns true if any test failed.
func (tsr *TestSuiteResult) HasFailures() bool {
	return tsr.Status == StatusFail
}

// GetCompletedTests returns a map of test ID to test case result for completed tests.
func (tsr *TestSuiteResult) GetCompletedTests() map[string]*TestCaseResult {
	completed := make(map[string]*TestCaseResult)

	for i := range tsr.Results {
		result := &tsr.Results[i]
		if result.ID != "" {
			completed[result.ID] = result
		}
	}

	return completed
}
