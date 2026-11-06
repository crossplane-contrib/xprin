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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gertd/go-pluralize"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
)

// ErrTestsFailed is what completeTestSuite wraps when some test cases failed.
var ErrTestsFailed = errors.New("tests failed")

// Runner handles test execution.
type Runner struct {
	*testexecutionOptions.Options

	fs        afero.Fs
	testSuite *testsuite.TestSuite // the test suite that is run: its file, its spec and its dependency graph
	output    io.Writer
	mode      runnerMode
	env       *testSuiteEnv
	// Mockable function fields
	runTestCaseFunc                   func(api.TestCase) *engine.TestCaseResult
	expandPathRelativeToTestSuiteFile func(base, path string) (string, error)
	verifyPathExists                  func(path string) error
	runCommand                        func(name string, args ...string) ([]byte, error)
	copy                              func(src, dest string, opts ...cp.Options) error
	convertClaimToXRFunc              func(r *Runner, claimPath, xrdPath, goldenPath, outputPath string) (string, error)
	patchXRFunc                       func(r *Runner, xrPath, outputPath string, patches api.Patches) (string, error)
}

// testSuiteEnv is the state of one run of a testsuite file, set by SetupTestSuite.
type testSuiteEnv struct {
	savedOutputsDir string                   // the internal artifacts, the outputs that test cases share
	testSuiteResult *engine.TestSuiteResult  // the result of the file, which the result of each test case is added to
	testCaseResults []*engine.TestCaseResult // the result of each test case, by its positionInTestSuite, once it ran
}

// testCaseEnv is the state of one test case run.
type testCaseEnv struct {
	tmpDir     string
	inputsDir  string
	outputsDir string
	index      int
}

// NewRunner creates a new test runner.
func NewRunner(options *testexecutionOptions.Options, testSuite *testsuite.TestSuite) *Runner {
	r := &Runner{
		fs:        afero.NewOsFs(),
		output:    os.Stdout, // Default output to stdout
		Options:   options,
		testSuite: testSuite,
		// Initialize mockable function fields with default implementations
		runTestCaseFunc:                   nil, // will set default below
		expandPathRelativeToTestSuiteFile: testsuite.ExpandPath,
		verifyPathExists:                  utils.VerifyPathExists,
		runCommand: func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...) //nolint:gosec // G204 - Used for Crossplane CLI and hooks, caller is trusted
			cmd.Dir = testSuite.GetDir()

			var combined bytes.Buffer

			cmd.Stdout = &combined
			cmd.Stderr = &combined
			err := cmd.Run()

			return combined.Bytes(), err
		},
		copy:                 cp.Copy,
		convertClaimToXRFunc: (*Runner).convertClaimToXR,
		patchXRFunc:          (*Runner).patchXR,
	}
	r.mode = newRunnerMode(r)

	return r
}

// GetTestSuiteFilePath returns the path of the testsuite file that the runner runs.
func (r *Runner) GetTestSuiteFilePath() string {
	return r.testSuite.File
}

// SetupTestSuite checks the testsuite file and makes the directories that the test cases use.
func (r *Runner) SetupTestSuite() (err error) {
	// Validate that testSuiteFile and testSuiteSpec are set (they should be set in NewRunner)
	if r.testSuite.File == "" {
		return fmt.Errorf("testsuite file path is required")
	}

	if r.testSuite.Spec == nil {
		return fmt.Errorf("testsuite specification is required")
	}

	// Create the directory of the internal artifacts, the outputs that test cases share (always created, cleaned up when testsuite finishes)
	savedOutputsDir, err := afero.TempDir(r.fs, "", "xprin-internal-artifacts-")
	if err != nil {
		return fmt.Errorf("failed to create internal artifacts directory: %w", err)
	}

	r.env = &testSuiteEnv{savedOutputsDir: savedOutputsDir}

	// If it can't go on, nothing will call completeTestSuite to remove it
	defer func() {
		if err != nil {
			r.removeSavedOutputsDir()
		}
	}()

	if r.Debug {
		utils.DebugPrintf("Created internal artifacts directory: %s\n", savedOutputsDir)
	}

	if r.Debug {
		utils.DebugPrintf("Using testsuite file directory for relative path resolution: %s\n", r.testSuite.GetDir())

		// We do not expand or verify common fields here. They will be expanded and verified by the individual test cases that do not override them.
		if r.testSuite.Spec.HasCommon() {
			r.debugPrintCommon(r.testSuite.Spec.Common, "Found common configuration:")
		}

		plural := pluralize.NewClient()
		utils.DebugPrintf("Found %s\n", plural.Pluralize("test case", len(r.testSuite.Spec.Tests), true))
	}

	// Create test suite result
	r.env.testSuiteResult = engine.NewTestSuiteResult(r.testSuite.File, r.Verbose)
	r.env.testSuiteResult.SetWorkingDir(r.WorkingDir)

	totalTestCases := len(r.testSuite.Spec.Tests)
	r.env.testCaseResults = make([]*engine.TestCaseResult, totalTestCases)

	if r.ArtifactsBaseDir != "" && totalTestCases > 0 {
		if err := r.initArtifactsRunDir(); err != nil {
			return err
		}
	}

	return nil
}

// runAndStoreTestCase runs the test case at positionInTestSuite and stores its result.
// positionInTestSuite is the position of the test case in the tests of the file.
// Different test cases can run at the same time, once the ones that each needs are done.
func (r *Runner) runAndStoreTestCase(positionInTestSuite int) {
	testCase := r.testSuite.Spec.Tests[positionInTestSuite]

	// Run the test and let the engine handle everything. The test cases are numbered, from 1, in the order they run in.
	result := r.runTestCase(testCase, r.testSuite.Graph.PositionInOrder(positionInTestSuite)+1, r.env.testSuiteResult)

	r.env.testCaseResults[positionInTestSuite] = result
	r.env.testSuiteResult.AddResult(result)
}

// runTestCase executes a single test case and returns a complete TestCaseResult.
func (r *Runner) runTestCase(testCase api.TestCase, index int, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	if r.Debug {
		utils.DebugPrintf("Starting test case '%s'\n", testCase.Name)
	}

	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, r.Verbose, r.ShowRender, r.ShowValidate, r.ShowHooks, r.ShowAssertions)

	// Skip immediately a test case that the mode has no use for, whatever it needs (the modes are in mode.go and
	// update_goldens.go)
	if r.mode.skips(testCase) {
		return result.Skip()
	}

	// Skip the test case if one of the test cases it needs failed or was skipped. Otherwise, the results of
	// the test cases it lists in `needs` are the only ones it can use with `.Tests`.
	needsResults, skipReasons := engine.GetNeedsResults(testCase.Needs, testSuiteResult.GetCompletedTests())
	if len(skipReasons) > 0 {
		return result.Skip(skipReasons...)
	}

	if r.runTestCaseFunc != nil {
		return r.runTestCaseFunc(testCase)
	}

	env := &testCaseEnv{index: index}

	// Create a temporary directory for the test case (with inputs and outputs subdirectories)
	var err error

	env.tmpDir, err = afero.TempDir(r.fs, "", "xprin-testcase-")
	if err != nil {
		return result.Fail(fmt.Errorf("failed to create temporary directory: %w", err))
	}

	defer func() {
		_ = r.fs.RemoveAll(env.tmpDir)
	}()

	if r.ArtifactsRunDir != "" {
		defer r.copyTestCaseArtifacts(env, testCase.Name)
	}

	// Create subdirectories for inputs and outputs
	env.inputsDir = filepath.Join(env.tmpDir, "inputs")

	env.outputsDir = filepath.Join(env.tmpDir, "outputs")
	if err := r.fs.MkdirAll(env.inputsDir, 0o750); err != nil {
		return result.Fail(fmt.Errorf("failed to create inputs directory: %w", err))
	}

	if err := r.fs.MkdirAll(env.outputsDir, 0o750); err != nil {
		return result.Fail(fmt.Errorf("failed to create outputs directory: %w", err))
	}

	if r.Debug {
		utils.DebugPrintf("Created temporary directory for test case: %s\n", env.tmpDir)
		utils.DebugPrintf("- Inputs: %s\n", env.inputsDir)
		utils.DebugPrintf("- Outputs: %s\n", env.outputsDir)
	}

	if r.testSuite.Spec.HasCommon() {
		testCase.MergeCommon(r.testSuite.Spec.Common)
	}

	// Process template variables for this test case
	if err := r.processTemplateVariables(&testCase, needsResults); err != nil {
		return result.Fail(fmt.Errorf("failed to process template variables: %w", err))
	}

	if err := testCase.CheckMandatoryFields(); err != nil {
		return result.Fail(err)
	}

	if r.Debug {
		r.debugPrintTestCase(testCase, "Test specification:")
	}

	if failed := r.expandInputPaths(&testCase, result); failed != nil {
		return failed
	}

	if failed := r.copyInputs(env, &testCase, result); failed != nil {
		return failed
	}

	if failed := r.runPreTestHooks(testCase, needsResults, result); failed != nil {
		return failed
	}

	if failed := r.renderTestCase(env, testCase, result); failed != nil {
		return failed
	}

	return r.mode.runPhasesAfterRender(testCase, env, needsResults, result)
}

// printTestCaseResult prints the stored result of the test case at positionInTestSuite.
// positionInTestSuite is the position of the test case in the tests of the file.
// Not at the same time as another one: the output has the order of the calls.
func (r *Runner) printTestCaseResult(positionInTestSuite int) {
	r.env.testCaseResults[positionInTestSuite].Print(r.output)
}

// completeTestSuite prints the summary of the file, and returns an error if any test case failed.
func (r *Runner) completeTestSuite() error {
	defer r.removeSavedOutputsDir()

	// Complete the test suite result
	r.env.testSuiteResult.Complete()

	// Print only the file summary (not individual test results)
	r.env.testSuiteResult.Print(r.output)

	if r.ArtifactsRunDir != "" {
		suiteArtifactsPath := filepath.Join(r.ArtifactsRunDir, artifactsRelPath(r.testSuite.File, r.WorkingDir))
		fmt.Fprintf(os.Stderr, "artifacts: %s\n", suiteArtifactsPath)
	}

	// Return error if any tests failed
	if r.env.testSuiteResult.HasFailures() {
		return fmt.Errorf("%w in testsuite %s", ErrTestsFailed, filepath.Base(r.testSuite.File))
	}

	return nil
}

// removeSavedOutputsDir removes the directory of the internal artifacts.
// It is safe to call it more than once.
func (r *Runner) removeSavedOutputsDir() {
	if r.env != nil && r.env.savedOutputsDir != "" {
		_ = r.fs.RemoveAll(r.env.savedOutputsDir)
	}
}
