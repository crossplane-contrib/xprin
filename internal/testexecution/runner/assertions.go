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

package runner

import (
	"fmt"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
)

// assertionExecutor runs all assertion kinds (xprin, diff, dyff) for a test case.
// Results are aggregated and printed together regardless of engine.
type assertionExecutor struct {
	fs            afero.Fs
	outputs       *engine.Outputs
	debug         bool
	testSuiteFile string
	expandPath    func(base, path string) (string, error)
	colorize      bool
}

// newAssertionExecutor creates a new assertion executor with context for all assertion kinds.
func newAssertionExecutor(
	fs afero.Fs,
	outputs *engine.Outputs,
	debug bool,
	testSuiteFile string,
	expandPath func(base, path string) (string, error),
	colorize bool,
) *assertionExecutor {
	return &assertionExecutor{
		fs:            fs,
		outputs:       outputs,
		debug:         debug,
		testSuiteFile: testSuiteFile,
		expandPath:    expandPath,
		colorize:      colorize,
	}
}

// resolveAndReadGoldenFile resolves expected/actual paths for a golden-file assertion and reads both files.
// On success returns (expectedPath, actualPath, expectedBytes, actualBytes, nil).
// On operational error (path expansion, missing file, resource not in render) returns (_, _, _, _, result) with StatusError ([!]); caller appends and continues.
func (e *assertionExecutor) resolveAndReadGoldenFile(a api.AssertionGoldenFile) (
	expectedPath, actualPath string,
	expectedBytes, actualBytes []byte,
	failResult *engine.AssertionResult,
) {
	expectedPath, err := e.expandPath(e.testSuiteFile, a.Expected)
	if err != nil {
		ar := engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("invalid expected path: %v", err))
		return "", "", nil, nil, &ar
	}

	if a.Resource == "" {
		actualPath = e.outputs.Render
	} else {
		var ok bool

		actualPath, ok = e.outputs.Rendered[a.Resource]
		if !ok {
			ar := engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("resource %q not found in render output", a.Resource))
			return "", "", nil, nil, &ar
		}
	}

	expectedBytes, err = afero.ReadFile(e.fs, expectedPath)
	if err != nil {
		ar := engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("read expected file: %v", err))
		return "", "", nil, nil, &ar
	}

	actualBytes, err = afero.ReadFile(e.fs, actualPath)
	if err != nil {
		ar := engine.NewAssertionResult(a.Name, engine.StatusError(), fmt.Sprintf("read actual file: %v", err))
		return "", "", nil, nil, &ar
	}

	return expectedPath, actualPath, expectedBytes, actualBytes, nil
}
