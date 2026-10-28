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

package testsuiteneeds

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// reportOf returns what report writes for a file.
func reportOf(t *testing.T, verbose bool, res testSuiteCheckResult) string {
	t.Helper()

	var out bytes.Buffer

	buffer := bufio.NewWriter(&out)
	report(buffer, engine.NewTestSuiteResult("tmp/a_xprin.yaml", verbose), res, verbose)

	require.NoError(t, buffer.Flush())

	return out.String()
}

// The format of the results is the engine's: here is only that report writes the test cases, and then the file.
func TestReport(t *testing.T) {
	base := testCaseCheckResult{name: "base"}
	one := testCaseCheckResult{name: "one", usesTests: true}
	two := testCaseCheckResult{name: "two", usesTests: true, missing: []string{"base"}}

	t.Run("a file that fails: the test cases that fail, and then the FAIL of the file", func(t *testing.T) {
		assert.Regexp(t, `^--- FAIL: two \(.*\)\n    Missing Needs:\n        \[x\] base\nFAIL\nFAIL\ttmp/a_xprin.yaml\t\d+\.\d{3}s\n$`,
			reportOf(t, false, testSuiteCheckResult{base, one, two}))
	})

	t.Run("a file that passes: only its line", func(t *testing.T) {
		assert.Regexp(t, `^ok\ttmp/a_xprin.yaml\t\d+\.\d{3}s\n$`, reportOf(t, false, testSuiteCheckResult{base, one}))
	})

	t.Run("the duration of the file is the time since its result was created", func(t *testing.T) {
		var out bytes.Buffer

		buffer := bufio.NewWriter(&out)
		suite := engine.NewTestSuiteResult("tmp/a_xprin.yaml", false)

		// What happens before report, like loading the file, counts
		time.Sleep(20 * time.Millisecond)
		report(buffer, suite, testSuiteCheckResult{base}, false)
		require.NoError(t, buffer.Flush())

		match := regexp.MustCompile(`\t(\d+\.\d{3})s\n$`).FindStringSubmatch(out.String())
		require.NotNil(t, match, out.String())

		seconds, err := strconv.ParseFloat(match[1], 64)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, seconds, 0.02)
	})

	t.Run("verbose: all the test cases, in order", func(t *testing.T) {
		assert.Regexp(t, `^=== RUN   base\n--- SKIP: base .*\n=== RUN   one\n--- PASS: one .*\nPASS\nok\ttmp/a_xprin.yaml\t`,
			reportOf(t, true, testSuiteCheckResult{base, one}))
	})
}

func TestNewTestCaseResult(t *testing.T) {
	tests := map[string]struct {
		check       testCaseCheckResult
		wantStatus  engine.Status
		wantError   string // empty: no error
		wantMissing []string
	}{
		"uses nothing: skipped": {
			check:      testCaseCheckResult{name: "base"},
			wantStatus: engine.StatusSkip(),
		},
		"uses tests and lists them: passes": {
			check:      testCaseCheckResult{name: "one", usesTests: true},
			wantStatus: engine.StatusPass(),
		},
		"misses needs: fails, with the needs in a section and no error": {
			check:       testCaseCheckResult{name: "two", usesTests: true, missing: []string{"base", "one"}},
			wantStatus:  engine.StatusFail(),
			wantMissing: []string{"base", "one"},
		},
		"has problems: fails, with the problems as the error": {
			check:      testCaseCheckResult{name: "four", problems: []string{"uses its own outputs", "something else"}},
			wantStatus: engine.StatusFail(),
			wantError:  "uses its own outputs\nsomething else",
		},
		"misses needs and has problems: fails, with both": {
			check:       testCaseCheckResult{name: "five", usesTests: true, missing: []string{"base"}, problems: []string{"no test case has that ID"}},
			wantStatus:  engine.StatusFail(),
			wantError:   "no test case has that ID",
			wantMissing: []string{"base"},
		},
		"a skip reason: skipped, with the reason as the error": {
			check:      testCaseCheckResult{name: "six", skipReason: "its needs can't be checked"},
			wantStatus: engine.StatusSkip(),
			wantError:  "its needs can't be checked",
		},
		"problems win over a skip reason": {
			check:      testCaseCheckResult{name: "seven", problems: []string{"a problem"}, skipReason: "a reason"},
			wantStatus: engine.StatusFail(),
			wantError:  "a problem",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := newTestCaseResult(tt.check, false)

			assert.Equal(t, tt.check.name, result.Name)
			assert.Equal(t, tt.wantStatus, result.Status)

			if tt.wantError == "" {
				require.NoError(t, result.Error)
			} else {
				require.EqualError(t, result.Error, tt.wantError)
			}

			var missing []string

			for _, section := range result.Sections {
				assert.Equal(t, "Missing Needs", section.Header)

				for _, line := range section.Lines {
					assert.Equal(t, engine.StatusFail(), line.Status)

					missing = append(missing, line.Text)
				}
			}

			assert.Equal(t, tt.wantMissing, missing)
			assert.Zero(t, result.Duration, "the tool doesn't time the test cases")
		})
	}
}

// failingWriter is a writer that never works, to see that the errors of the writes of report are not lost.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestReport_AWriteErrorComesOutWhenTheBufferIsFlushed(t *testing.T) {
	// A small buffer, so that the writes reach the writer that fails before the end of the report
	buffer := bufio.NewWriterSize(failingWriter{}, 16)

	report(buffer, engine.NewTestSuiteResult("tmp/a_xprin.yaml", true), testSuiteCheckResult{
		{name: "a", missing: []string{"base", "one"}, problems: []string{"something to fix"}},
	}, true)

	require.ErrorContains(t, buffer.Flush(), "disk full")
}

func TestWriteSummary(t *testing.T) {
	tests := map[string]struct {
		missNeeds, requireAPerson int
		want                      string
	}{
		"nothing is left: nothing is written": {0, 0, ""},
		"one file that misses needs": {1, 0, `Total:
    [!] 1 testsuite file misses needs
`},
		"several files that miss needs": {2, 0, `Total:
    [!] 2 testsuite files miss needs
`},
		"one file that requires a person": {0, 1, `Total:
    [!] 1 testsuite file requires a person
`},
		"several of each": {20, 7, `Total:
    [!] 20 testsuite files miss needs
    [!] 7 testsuite files require a person
`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer

			buffer := bufio.NewWriter(&out)
			writeSummary(buffer, tt.missNeeds, tt.requireAPerson)

			require.NoError(t, buffer.Flush())
			assert.Equal(t, tt.want, out.String())
		})
	}
}
