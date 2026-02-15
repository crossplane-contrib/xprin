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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/pmezard/go-difflib/difflib"
)

// ANSI SGR codes for terminal output (unified diff).
// Matches GNU diff's default palette (see https://www.gnu.org/software/diffutils/manual/html_node/diff-Options.html):
//
//	--palette default: rs=0:hd=1:ad=32:de=31:ln=36
//	de=31 red (deleted/removed lines), ad=32 green (added lines), hd=1 bold (header), ln=36 cyan (line numbers / hunk @@).
const (
	ansiReset = "\033[0m"
	ansiRed   = "\033[31m" // deleted lines (GNU diff de=31)
	ansiGreen = "\033[32m" // added lines (GNU diff ad=32)
	ansiBold  = "\033[1m"  // file headers --- / +++ (GNU diff hd=1)
	ansiCyan  = "\033[36m" // hunk headers @@ (GNU diff ln=36)
)

// executeAssertionsDiff runs diff assertions: compares actual output (full render or one resource) to expected (golden) file.
// Uses shared resolve+read from the executor; compares with bytes.Equal. When colorize is true, the failure message is a colored unified diff.
func (e *assertionExecutor) executeAssertionsDiff(assertions []api.AssertionGoldenFile) []engine.AssertionResult {
	results := make([]engine.AssertionResult, 0, len(assertions))

	for _, a := range assertions {
		expectedPath, actualPath, expectedBytes, actualBytes, failResult := e.resolveAndReadGoldenFile(a)
		if failResult != nil {
			results = append(results, *failResult)
			continue
		}

		if bytes.Equal(expectedBytes, actualBytes) {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusPass(), "files match"))
			continue
		}

		msg := formatDiffMessageUnified(expectedPath, actualPath, expectedBytes, actualBytes, e.colorize)
		results = append(results, engine.NewAssertionResult(a.Name, engine.StatusFail(), msg))
	}

	return results
}

// formatDiffMessageUnified returns a unified diff (like diff -u) between expected and actual.
// When colorize is true, adds ANSI color codes (red for removed, green for added, cyan for file headers, dim for hunk headers).
func formatDiffMessageUnified(expectedPath, actualPath string, expected, actual []byte, colorize bool) string {
	fromLabel := filepath.Base(expectedPath)
	if fromLabel == "" {
		fromLabel = "expected"
	}

	toLabel := filepath.Base(actualPath)
	if toLabel == "" {
		toLabel = "actual"
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(string(actual)),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  3,
	}

	out, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Sprintf("files differ (unified diff failed: %v)", err)
	}

	if colorize {
		out = colorizeUnifiedDiff(out)
	}

	return out
}

// colorizeUnifiedDiff wraps unified diff lines with ANSI SGR codes matching GNU diff's default palette.
func colorizeUnifiedDiff(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			lines[i] = ansiBold + line + ansiReset
		case strings.HasPrefix(line, "@@"):
			lines[i] = ansiCyan + line + ansiReset
		case len(line) > 0 && line[0] == '-':
			lines[i] = ansiRed + line + ansiReset
		case len(line) > 0 && line[0] == '+':
			lines[i] = ansiGreen + line + ansiReset
		}
	}

	return strings.Join(lines, "\n")
}
