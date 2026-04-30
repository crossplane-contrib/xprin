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
	"strings"
	"time"
)

// sanitizePathSegment replaces characters unsafe in filesystem paths with underscores,
// collapses consecutive underscores, and falls back to "unnamed" for empty results.
func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder

	for _, r := range s {
		if r < 0x20 || strings.ContainsRune(`\/: *?"<>|`, r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}

	result := b.String()
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	result = strings.Trim(result, "_")
	if result == "" {
		return "unnamed"
	}

	return result
}

// artifactsRelPath computes the path of the testsuite under the run root, mirroring the
// user's invocation path (relative to cwd). Falls back to sanitised absolute segments
// when the suite file lives outside cwd.
func artifactsRelPath(suiteFile, cwd string) string {
	absSuite, err := filepath.Abs(suiteFile)
	if err != nil {
		absSuite = suiteFile
	}

	base := sanitizePathSegment(strings.TrimSuffix(filepath.Base(absSuite), filepath.Ext(absSuite)))

	rel, err := filepath.Rel(cwd, absSuite)
	if err == nil && !strings.HasPrefix(rel, "..") {
		dir := filepath.Dir(rel)
		if dir == "." {
			return base
		}

		parts := strings.Split(filepath.ToSlash(dir), "/")

		sanitized := make([]string, 0, len(parts)+1)
		for _, p := range parts {
			if p != "" && p != "." {
				sanitized = append(sanitized, sanitizePathSegment(p))
			}
		}

		sanitized = append(sanitized, base)

		return filepath.Join(sanitized...)
	}

	// Fallback: sanitised absolute path segments.
	dir := filepath.Dir(absSuite)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	sanitized := make([]string, 0, len(parts)+1)
	for _, p := range parts {
		if p != "" && p != "." {
			sanitized = append(sanitized, sanitizePathSegment(p))
		}
	}

	sanitized = append(sanitized, base)

	return filepath.Join(sanitized...)
}

// indexPadWidth returns the zero-padding width for test case indices in a suite of size n.
func indexPadWidth(n int) int {
	switch {
	case n < 100:
		return 2
	case n < 1000:
		return 3
	default:
		w := 0
		for x := n; x > 0; x /= 10 {
			w++
		}

		return w
	}
}

// initArtifactsRunDir creates <ArtifactsBaseDir>/xprin-artifacts-<YYYYMMDDHHMMSS>/,
// stores the absolute path in ArtifactsRunDir, and prints it to stderr. Idempotent:
// no-ops when ArtifactsRunDir is already set (shared across runners in one invocation).
func (r *Runner) initArtifactsRunDir() error {
	if r.ArtifactsBaseDir == "" || r.ArtifactsRunDir != "" {
		return nil
	}

	timestamp := time.Now().Format("20060102150405")

	runRoot := filepath.Join(r.ArtifactsBaseDir, "xprin-artifacts-"+timestamp)
	if err := r.fs.MkdirAll(runRoot, 0o750); err != nil {
		return fmt.Errorf("failed to create artifacts directory %s: %w", runRoot, err)
	}

	abs, err := filepath.Abs(runRoot)
	if err != nil {
		abs = runRoot
	}

	r.ArtifactsRunDir = abs

	return nil
}

// copyTestCaseArtifacts copies inputs/ and outputs/ from the test case tmp dir into
// the run root. Called via defer so it runs on every exit path, including early failures.
// Errors are printed to stderr rather than failing the test.
func (r *Runner) copyTestCaseArtifacts(testCaseName string) {
	if r.ArtifactsRunDir == "" || r.testCaseTmpDir == "" {
		return
	}

	relPath := artifactsRelPath(r.testSuiteFile, r.WorkingDir)
	w := indexPadWidth(r.totalTestCases)
	dirName := fmt.Sprintf("%0*d_%s", w, r.currentTestCaseIndex, sanitizePathSegment(testCaseName))
	dest := filepath.Join(r.ArtifactsRunDir, relPath, dirName)

	if err := r.fs.MkdirAll(dest, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create artifact dir %s: %v\n", dest, err)
		return
	}

	if err := r.copy(r.inputsDir, filepath.Join(dest, "inputs")); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to copy inputs for %q: %v\n", testCaseName, err)
	}

	if _, err := r.fs.Stat(r.outputsDir); err == nil {
		if err := r.copy(r.outputsDir, filepath.Join(dest, "outputs")); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to copy outputs for %q: %v\n", testCaseName, err)
		}
	}
}
