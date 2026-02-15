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

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
)

// executeAssertionsDyff runs dyff assertions: compares actual output to expected (golden) file using the dyff library.
// Uses shared resolve+read from the executor; builds ytbx.InputFile from bytes, calls dyff.CompareInputFiles; on mismatch uses dyff.HumanReport.
func (e *assertionExecutor) executeAssertionsDyff(assertions []api.AssertionGoldenFile) []engine.AssertionResult {
	results := make([]engine.AssertionResult, 0, len(assertions))

	for _, a := range assertions {
		expectedPath, actualPath, expectedBytes, actualBytes, failResult := e.resolveAndReadGoldenFile(a)
		if failResult != nil {
			results = append(results, *failResult)
			continue
		}

		fromDocs, err := ytbx.LoadDocuments(expectedBytes)
		if err != nil {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("load expected: %v", err)))
			continue
		}

		toDocs, err := ytbx.LoadDocuments(actualBytes)
		if err != nil {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("load actual: %v", err)))
			continue
		}

		fromInput := ytbx.InputFile{Location: expectedPath, Documents: fromDocs}
		toInput := ytbx.InputFile{Location: actualPath, Documents: toDocs}

		report, err := dyff.CompareInputFiles(fromInput, toInput)
		if err != nil {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("dyff compare: %v", err)))
			continue
		}

		if len(report.Diffs) == 0 {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusPass(), "files match"))
			continue
		}

		var buf bytes.Buffer

		human := &dyff.HumanReport{Report: report}
		if err := human.WriteReport(&buf); err != nil {
			results = append(results, engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("dyff report: %v", err)))
			continue
		}

		// Pass raw output to the formatter (same as hooks); no trimming so ASCII art keeps its layout.
		results = append(results, engine.NewAssertionResult(a.Name, engine.StatusFail(), buf.String()))
	}

	return results
}
