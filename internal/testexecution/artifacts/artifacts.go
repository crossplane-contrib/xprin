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

// Package artifacts is where the artifacts of a run are kept with `--artifacts`: the directory of the run, and the
// directories of the testsuite files and of the test cases in it.
package artifacts

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/spf13/afero"
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

// RelPath computes the path of the testsuite under the run root, mirroring the
// user's invocation path (relative to cwd). Falls back to sanitised absolute segments
// when the suite file lives outside cwd.
func RelPath(suiteFile, cwd string) string {
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

// padWidth returns the zero-padding width for test case indices in a suite of size n.
func padWidth(n int) int {
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

// TestCaseDirName is the name of the directory of a test case, in the one of its testsuite file: its number in the
// order that the test cases run in, padded for the number of test cases of the file, and its sanitized name.
func TestCaseDirName(index, testCases int, name string) string {
	return fmt.Sprintf("%0*d_%s", padWidth(testCases), index, sanitizePathSegment(name))
}

// InitRunDir creates <ArtifactsBaseDir>/xprin-artifacts-<YYYYMMDDHHMMSS>/ and stores the absolute path in
// ArtifactsRunDir. Idempotent: no-ops when ArtifactsRunDir is already set (shared by all the testsuite files of an
// invocation).
func InitRunDir(fs afero.Fs, options *testexecutionOptions.Options) error {
	if options.ArtifactsBaseDir == "" || options.ArtifactsRunDir != "" {
		return nil
	}

	timestamp := time.Now().Format("20060102150405")

	runRoot := filepath.Join(options.ArtifactsBaseDir, "xprin-artifacts-"+timestamp)
	if err := fs.MkdirAll(runRoot, 0o750); err != nil {
		return fmt.Errorf("failed to create artifacts directory %s: %w", runRoot, err)
	}

	abs, err := filepath.Abs(runRoot)
	if err != nil {
		abs = runRoot
	}

	options.ArtifactsRunDir = abs

	return nil
}
