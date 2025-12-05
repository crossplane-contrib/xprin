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

package processor

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/testexecution/runner"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gertd/go-pluralize"
	"github.com/spf13/afero"
)

// runnerInterface allows dependency injection for test runners (for production and testing).
type runnerInterface interface {
	RunTests() error
}

// Mockable functions
//
//nolint:gochecknoglobals // Global variables for dependency injection in tests
var (
	newRunnerFunc = func(options *testexecutionUtils.Options, testSuiteFile string, testSuiteSpec *api.TestSuiteSpec) runnerInterface {
		return runner.NewRunner(options, testSuiteFile, testSuiteSpec)
	}
)

// ProcessTargets processes the targets and runs the tests
//
//nolint:gocognit // Complex target processing with multiple validation and execution phases
func ProcessTargets(fs afero.Fs, targets []string, options *testexecutionUtils.Options) error {
	var hasErrors bool

	for _, path := range targets {
		if strings.HasSuffix(path, "...") {
			root := strings.TrimSuffix(path, "...")
			if strings.HasSuffix(root, string(filepath.Separator)) {
				root = strings.TrimSuffix(root, string(filepath.Separator))
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

				if err := processDirectory(fs, dir, options); err != nil {
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
			if err := processDirectory(fs, path, options); err != nil {
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

		if err := processTestSuiteFile(fs, path, options); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		utils.OutputPrintf("FAIL\n")
		return fmt.Errorf("processing completed with errors")
	}

	return nil
}

// processDirectory handles finding testsuite files in a directory, printing the go test-style message if none are found.
// Optionally runs tests from each found testsuite file after loading and validating the configuration.
func processDirectory(fs afero.Fs, dir string, options *testexecutionUtils.Options) error {
	if options.Debug {
		utils.DebugPrintf("Processing directory %s\n", dir)
	}

	files, err := findTestSuiteFiles(fs, dir)
	if err != nil {
		// Special case: if the error is just that no files were found, handle it as an info message
		if strings.HasPrefix(err.Error(), "no test files found matching pattern") {
			fmt.Fprintf(os.Stderr, "?   \t%s\t[no testsuite files]\n", dir)
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
		if err := processTestSuiteFile(fs, testSuiteFile, options); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("errors occurred processing files in directory %s", dir)
	}

	return nil
}

// processTestSuiteFile processes a single test file, loading the configuration and running tests if applicable.
func processTestSuiteFile(fs afero.Fs, testSuiteFile string, options *testexecutionUtils.Options) error {
	if options.Debug {
		utils.DebugPrintf("Processing testsuite file %s\n", testSuiteFile)
	}

	// Load and validate test configuration
	testSuiteSpec, err := load(fs, testSuiteFile)
	if err != nil {
		if strings.HasPrefix(err.Error(), ("no test cases found")) {
			fmt.Fprintf(os.Stderr, "?   \t%s\t[no test cases found]\n", testSuiteFile)
			return nil
		}

		return reportTestSuiteError(testSuiteFile, err, "invalid testsuite file")
	}

	// Now that we know we have tests to run, check for empty names and duplicate IDs
	if err := testSuiteSpec.CheckValidTestSuiteFile(); err != nil {
		return reportTestSuiteError(testSuiteFile, err, "invalid testsuite file")
	}

	testRunner := newRunnerFunc(options, testSuiteFile, testSuiteSpec)

	fileErr := testRunner.RunTests()
	if fileErr != nil {
		errMsg := fileErr.Error()
		if !strings.Contains(errMsg, "tests failed in testsuite") {
			return reportTestSuiteError(testSuiteFile, fileErr, "testsuite file execution error")
		}

		return fmt.Errorf("test execution failed for %s: %w", testSuiteFile, fileErr)
	}

	return nil
}
