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

// Package discovery finds the testsuite files that targets resolve to, and loads them.
package discovery

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gertd/go-pluralize"
	"github.com/spf13/afero"
)

// Options controls what Walk and Load report.
type Options struct {
	// Quiet suppresses the "[no testsuite files]" and "[no test cases found]" messages.
	Quiet bool
	// Debug prints how targets and files are processed.
	Debug bool
}

// Visitor is called for every testsuite file that the targets resolve to.
type Visitor func(testSuiteFile string) error

// Walk resolves the targets (files, directories, or recursive directories) to testsuite files and
// calls visit for each of them. It reports the problems it finds, and returns an error if any of
// them, or any visit, failed.
//
//nolint:gocognit // Complex target processing with multiple validation phases
func Walk(fs afero.Fs, targets []string, options Options, visit Visitor) error {
	var hasErrors bool

	for _, path := range targets {
		if before, ok := strings.CutSuffix(path, "..."); ok {
			root := before
			if before, ok := strings.CutSuffix(root, string(filepath.Separator)); ok {
				root = before
			}

			dirs, err := recursiveDirs(fs, root)
			if err != nil {
				_ = reportError(root, "failed to find testsuite files", err)
				hasErrors = true

				continue
			}

			for _, dir := range dirs {
				info, err := fs.Stat(dir)
				if err != nil || !info.IsDir() {
					continue
				}

				if err := walkDirectory(fs, dir, options, visit); err != nil {
					hasErrors = true
				}
			}

			continue
		}

		info, err := fs.Stat(path)
		if errors.Is(err, iofs.ErrNotExist) {
			if options.Debug {
				utils.DebugPrintf("Skipping test path %s because it does not exist\n", path)
			}

			continue
		}

		if err != nil {
			_ = reportError(path, "failed to access test path", err)
			hasErrors = true

			continue
		}

		if info.IsDir() {
			if err := walkDirectory(fs, path, options, visit); err != nil {
				hasErrors = true
			}

			continue
		}

		// Direct file - check if it's a valid test file
		if !isValidTestSuiteFileName(path) {
			if options.Debug {
				utils.DebugPrintf("Skipping file %s because it is not a valid test file. It should be named 'xprin.yaml' or end with '_xprin.yaml' with at least one character before the underscore\n", path)
			}

			continue
		}

		if err := visit(path); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		utils.OutputPrintf("FAIL\n")
		return fmt.Errorf("processing completed with errors")
	}

	return nil
}

// walkDirectory handles finding testsuite files in a directory, printing the go test-style message if none are found,
// and calls visit for each of them.
func walkDirectory(fs afero.Fs, dir string, options Options, visit Visitor) error {
	if options.Debug {
		utils.DebugPrintf("Processing directory %s\n", dir)
	}

	files, err := findTestSuiteFiles(fs, dir)
	if err != nil {
		// Special case: if the error is just that no files were found, handle it as an info message
		if strings.HasPrefix(err.Error(), "no test files found matching pattern") {
			if !options.Quiet {
				fmt.Fprintf(os.Stderr, "?   \t%s\t[no testsuite files]\n", dir)
			}

			return nil
		}
		// For other errors, report them as real errors
		return reportError(dir, "failed to find testsuite files", err)
	}
	// Note: No need to check len(files) == 0 here because:
	// 1. findTestSuiteFiles guarantees it will return an error if no files are found
	// 2. If we get here, we already know there's no error, so files must be non-empty
	if options.Debug {
		plural := pluralize.NewClient()
		utils.DebugPrintf("Found %s in directory %s\n", plural.Pluralize("testsuite file", len(files), true), dir)
	}

	var hasErrors bool

	for _, testSuiteFile := range files {
		if err := visit(testSuiteFile); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("errors occurred processing files in directory %s", dir)
	}

	return nil
}
