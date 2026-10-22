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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// failedFile is how the engine ends the output of a file that failed: the FAIL of the suite, and the line of the file,
// which starts with a FAIL too.
const failedFile = "FAIL\nFAIL\ttmp/a_xprin.yaml\t0.002s\n" //nolint:dupword // The engine writes FAIL twice

func TestReport(t *testing.T) {
	// base uses nothing, one uses base and lists it, two misses base, three misses base and one, four has a problem,
	// five misses base and has a problem too
	base := testCaseCheckResult{name: "base"}
	one := testCaseCheckResult{name: "one", usesTests: true}
	two := testCaseCheckResult{name: "two", usesTests: true, missing: []string{"base"}}
	three := testCaseCheckResult{name: "three", usesTests: true, missing: []string{"base", "one"}}
	four := testCaseCheckResult{name: "four", problems: []string{"uses its own outputs (.Tests.four)"}}
	five := testCaseCheckResult{name: "five", usesTests: true, missing: []string{"base"}, problems: []string{"uses .Tests.nope, but no test case has that ID"}}

	// a person took care of it: the tool can't check it
	skippedForAReason := testCaseCheckResult{name: "six", skipReason: "uses .Tests in a way that can't be traced: its needs can't be checked"}

	all := testSuiteCheckResult{base, one, two, three, four, five}

	const elapsed = 1500 * time.Microsecond

	tests := map[string]struct {
		res     testSuiteCheckResult
		verbose bool
		want    string
	}{
		"only what fails": {
			res:  all,
			want: "--- FAIL: two (0.00s)\n    Missing Needs:\n        [x] base\n--- FAIL: three (0.00s)\n    Missing Needs:\n        [x] base\n        [x] one\n--- FAIL: four (0.00s)\n    [!] uses its own outputs (.Tests.four)\n--- FAIL: five (0.00s)\n    Missing Needs:\n        [x] base\n    [!] uses .Tests.nope, but no test case has that ID\n" + failedFile,
		},
		"verbose: all of them, with their results": {
			res:     all,
			verbose: true,
			want:    "=== RUN   base\n--- SKIP: base (0.00s)\n=== RUN   one\n--- PASS: one (0.00s)\n=== RUN   two\n--- FAIL: two (0.00s)\n    Missing Needs:\n        [x] base\n=== RUN   three\n--- FAIL: three (0.00s)\n    Missing Needs:\n        [x] base\n        [x] one\n=== RUN   four\n--- FAIL: four (0.00s)\n    [!] uses its own outputs (.Tests.four)\n=== RUN   five\n--- FAIL: five (0.00s)\n    Missing Needs:\n        [x] base\n    [!] uses .Tests.nope, but no test case has that ID\n" + failedFile,
		},
		"nothing to say": {
			res:  testSuiteCheckResult{base, one},
			want: "ok\ttmp/a_xprin.yaml\t0.002s\n",
		},
		"nothing to say, verbose: a pass for the test case that uses needs, a skip for the one that uses none": {
			res:     testSuiteCheckResult{base, one},
			verbose: true,
			want: `=== RUN   base
--- SKIP: base (0.00s)
=== RUN   one
--- PASS: one (0.00s)
PASS
ok	tmp/a_xprin.yaml	0.002s
`,
		},
		"a skip with a reason is shown without verbose, and does not fail the file": {
			res: testSuiteCheckResult{base, skippedForAReason},
			want: `--- SKIP: six (0.00s)
    [!] uses .Tests in a way that can't be traced: its needs can't be checked
ok	tmp/a_xprin.yaml	0.002s
`,
		},
		"a skip with a reason, verbose": {
			res:     testSuiteCheckResult{base, skippedForAReason},
			verbose: true,
			want: `=== RUN   base
--- SKIP: base (0.00s)
=== RUN   six
--- SKIP: six (0.00s)
    [!] uses .Tests in a way that can't be traced: its needs can't be checked
PASS
ok	tmp/a_xprin.yaml	0.002s
`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer

			buffer := bufio.NewWriter(&out)
			report(buffer, "tmp/a_xprin.yaml", tt.res, tt.verbose, elapsed)

			require.NoError(t, buffer.Flush())
			assert.Equal(t, tt.want, out.String())
		})
	}
}

// failingWriter is a writer that never works, to see that the errors of the writes of report are not lost.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestReport_AWriteErrorComesOutWhenTheBufferIsFlushed(t *testing.T) {
	// A small buffer, so that the writes reach the writer that fails before the end of the report
	buffer := bufio.NewWriterSize(failingWriter{}, 16)

	report(buffer, "tmp/a_xprin.yaml", testSuiteCheckResult{
		{name: "a", missing: []string{"base", "one"}, problems: []string{"something to fix"}},
	}, true, 0)

	require.ErrorContains(t, buffer.Flush(), "disk full")
}

func TestWriteSummary(t *testing.T) {
	tests := map[string]struct {
		missNeeds, needAPerson int
		want                   string
	}{
		"nothing is left: nothing is written": {0, 0, ""},
		"one file that misses needs": {1, 0, `Total:
    [!] 1 testsuite file misses needs: see Missing Needs above
`},
		"several files that miss needs": {2, 0, `Total:
    [!] 2 testsuite files miss needs: see Missing Needs above
`},
		"one file that requires a person": {0, 1, `Total:
    [!] 1 testsuite file requires a person
`},
		"several of each": {20, 7, `Total:
    [!] 20 testsuite files miss needs: see Missing Needs above
    [!] 7 testsuite files require a person
`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer

			buffer := bufio.NewWriter(&out)
			writeSummary(buffer, tt.missNeeds, tt.needAPerson)

			require.NoError(t, buffer.Flush())
			assert.Equal(t, tt.want, out.String())
		})
	}
}
