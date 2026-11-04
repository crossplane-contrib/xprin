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
	"os"
	"path/filepath"

	"github.com/crossplane-contrib/xprin/internal/testexecution/artifacts"
)

// copyTestCaseArtifacts copies inputs/ and outputs/ from the test case tmp dir into
// the run root. Called via defer so it runs on every exit path, including early failures.
// Errors are printed to stderr rather than failing the test.
func (r *Runner) copyTestCaseArtifacts(env *testCaseEnv, testCaseName string) {
	if r.ArtifactsRunDir == "" || env.tmpDir == "" {
		return
	}

	relPath := artifacts.RelPath(env.testSuite.File, r.WorkingDir)
	dirName := artifacts.TestCaseDirName(env.index, len(env.testSuite.Spec.Tests), testCaseName)
	dest := filepath.Join(r.ArtifactsRunDir, relPath, dirName)

	if err := r.fs.MkdirAll(dest, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create artifact dir %s: %v\n", dest, err)
		return
	}

	if err := r.copy(env.inputsDir, filepath.Join(dest, "inputs")); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to copy inputs for %q: %v\n", testCaseName, err)
	}

	if _, err := r.fs.Stat(env.outputsDir); err == nil {
		if err := r.copy(env.outputsDir, filepath.Join(dest, "outputs")); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to copy outputs for %q: %v\n", testCaseName, err)
		}
	}
}
