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

// Package processor provides test suite file discovery, loading, and processing functionality.
package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// recursiveDirs recursively collects all directories under a root.
func recursiveDirs(fs afero.Fs, root string) ([]string, error) {
	var dirs []string

	err := afero.Walk(fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirs = append(dirs, path)
		}

		return nil
	})

	return dirs, err
}

// findTestSuiteFiles finds all test files matching the given pattern.
func findTestSuiteFiles(fs afero.Fs, pattern string) ([]string, error) {
	matches, err := afero.Glob(fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to match pattern %s: %w", pattern, err)
	}

	var files []string

	for _, match := range matches {
		info, err := fs.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", match, err)
		}

		if !info.IsDir() {
			// For files, only include if they are named "xprin.yaml" or end with "_xprin.yaml" with at least one character before the underscore
			if !isValidTestSuiteFileName(match) {
				continue
			}

			files = append(files, match)

			continue
		}

		// Look for all yaml files and then filter by our validation function
		allMatches, err := afero.Glob(fs, filepath.Join(match, "*.yaml"))
		if err != nil {
			return nil, fmt.Errorf("failed to match pattern in directory %s: %w", match, err)
		}

		// Filter results using isValidTestSuiteFileName
		for _, fileMatch := range allMatches {
			if isValidTestSuiteFileName(fileMatch) {
				files = append(files, fileMatch)
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no test files found matching pattern %s", pattern)
	}

	return files, nil
}

// isValidTestSuiteFileName checks if a filename matches the patterns for valid test suite files.
// Valid patterns are:
// - Exactly "xprin.yaml"
// - Ending with "_xprin.yaml" with at least one character before the underscore.
func isValidTestSuiteFileName(filename string) bool {
	base := filepath.Base(filename)
	return base == "xprin.yaml" || (strings.HasSuffix(base, "_xprin.yaml") && len(base) > len("_xprin.yaml"))
}
