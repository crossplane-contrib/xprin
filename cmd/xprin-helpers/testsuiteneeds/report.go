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
	"errors"
	"io"
	"strings"
	"time"

	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/gertd/go-pluralize"
)

// report writes the results of a testsuite file as xprin test does.
func report(out io.Writer, testSuiteFile string, res testSuiteCheckResult, verbose bool, elapsed time.Duration) {
	testSuiteResult := engine.NewTestSuiteResult(testSuiteFile, verbose)

	for _, check := range res {
		testCaseResult := newTestCaseResult(check, verbose)
		testCaseResult.Print(out)

		testSuiteResult.AddResult(testCaseResult)
	}

	// The time of everything, from before the file was loaded
	testSuiteResult.Duration = elapsed
	testSuiteResult.Print(out)
}

// newTestCaseResult returns the result of a test case, with the needs that it misses in a section.
func newTestCaseResult(check testCaseCheckResult, verbose bool) *engine.TestCaseResult {
	result := engine.NewTestCaseResult(check.name, "", verbose, false, false, false, false)

	for _, id := range check.missing {
		result.AddToSection("Missing Needs", engine.StatusFail(), id)
	}

	switch {
	case len(check.problems) > 0:
		// The engine writes each line of the error with a [!]
		result.Fail(errors.New(strings.Join(check.problems, "\n")))
	case len(check.missing) > 0:
		// What is missing is in the section: there is no error to show
		result.Fail(nil)
	case check.skipReason != "":
		result.Skip(errors.New(check.skipReason))
	case check.usesTests:
		result.Complete()
	default:
		result.Skip()
	}

	// The tool doesn't time the test cases, only the file
	result.Duration = 0

	return result
}

// writeSummary writes what is left to do after the walk of all the files: a heading, and a line for each thing, with
// the mark of an error so that it is not missed after a long list of results. Nothing is written when there is none.
// The caller flushes the buffer, and gets the first error of the writes there.
func writeSummary(out *bufio.Writer, missNeeds, requireAPerson int) {
	if missNeeds == 0 && requireAPerson == 0 {
		return
	}

	write := func(text string) { _, _ = out.WriteString(text) }
	plural := pluralize.NewClient()

	// files says "1 testsuite file misses ..." and "2 testsuite files miss ...", given the verb in its base form. The
	// library knows nouns only: the plural of the verb is the form for a singular subject
	files := func(n int, verb, what string) string {
		if n == 1 {
			verb = plural.Plural(verb)
		}

		return plural.Pluralize("testsuite file", n, true) + " " + verb + " " + what
	}

	mark := engine.StatusError().Symbol

	write("Total:\n")

	if missNeeds > 0 {
		write("    " + mark + " " + files(missNeeds, "miss", "needs") + "\n")
	}

	if requireAPerson > 0 {
		write("    " + mark + " " + files(requireAPerson, "require", "a person") + "\n")
	}
}
