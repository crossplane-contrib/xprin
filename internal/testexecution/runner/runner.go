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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/testsuite/legacy05"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gertd/go-pluralize"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
)

// Runner handles test execution.
type Runner struct {
	*testexecutionOptions.Options

	fs                    afero.Fs
	testSuite             *testsuite.TestSuite // the test suite that is run: its file, its spec and its dependency graph
	legacy05              bool                 // the file is in the format of xprin up to 0.5 (see legacy05.IsTestSuite)
	output                io.Writer
	testSuiteArtifactsDir string
	totalTestCases        int
	// Mockable function fields
	runTestsFunc                      func() error
	runTestCaseFunc                   func(api.TestCase) *engine.TestCaseResult
	expandPathRelativeToTestSuiteFile func(base, path string) (string, error)
	verifyPathExists                  func(path string) error
	runCommand                        func(name string, args ...string) ([]byte, error)
	copy                              func(src, dest string, opts ...cp.Options) error
	convertClaimToXRFunc              func(r *Runner, claimPath, xrdPath, goldenPath, outputPath string) (string, error)
	patchXRFunc                       func(r *Runner, xrPath, outputPath string, patches api.Patches) (string, error)
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
	return &Runner{
		fs:        afero.NewOsFs(),
		output:    os.Stdout, // Default output to stdout
		Options:   options,
		testSuite: testSuite,
		legacy05:  legacy05.IsTestSuite(testSuite.Spec),
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
}

// RunTests runs all tests in a test suite.
func (r *Runner) RunTests() error {
	if r.runTestsFunc != nil {
		return r.runTestsFunc()
	}

	// Validate that testSuiteFile and testSuiteSpec are set (they should be set in NewRunner)
	if r.testSuite.File == "" {
		return fmt.Errorf("testsuite file path is required")
	}

	if r.testSuite.Spec == nil {
		return fmt.Errorf("testsuite specification is required")
	}

	if r.legacy05 {
		utils.WarningPrintf("%s\n", legacy05.Warning(r.testSuite.File))
	}

	// Create testsuite artifacts directory (always created, cleaned up when testsuite finishes)
	var err error

	r.testSuiteArtifactsDir, err = afero.TempDir(r.fs, "", "xprin-testsuite-artifacts-")
	if err != nil {
		return fmt.Errorf("failed to create testsuite artifacts directory: %w", err)
	}

	defer func() {
		_ = r.fs.RemoveAll(r.testSuiteArtifactsDir)
	}()

	if r.Debug {
		utils.DebugPrintf("Created testsuite artifacts directory: %s\n", r.testSuiteArtifactsDir)
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
	testSuiteResult := engine.NewTestSuiteResult(r.testSuite.File, r.Verbose)
	testSuiteResult.WorkingDir = r.WorkingDir

	r.totalTestCases = len(r.testSuite.Spec.Tests)
	if r.ArtifactsBaseDir != "" && r.totalTestCases > 0 {
		if err := r.initArtifactsRunDir(); err != nil {
			return err
		}
	}

	// Loop through all test cases, in the order of their dependencies, and run them directly
	for positionInOrder, positionInTestSuite := range r.runOrder() {
		testCase := r.testSuite.Spec.Tests[positionInTestSuite]

		// Run the test and let the engine handle everything
		testCaseResult := r.runTestCase(testCase, positionInOrder+1, testSuiteResult)
		testCaseResult.Print(r.output) // Print immediately as test completes
		testSuiteResult.AddResult(testCaseResult)
	}

	// Complete the test suite result
	testSuiteResult.Complete()

	// Print only the file summary (not individual test results)
	testSuiteResult.Print(r.output)

	if r.ArtifactsRunDir != "" {
		suiteArtifactsPath := filepath.Join(r.ArtifactsRunDir, artifactsRelPath(r.testSuite.File, r.WorkingDir))
		fmt.Fprintf(os.Stderr, "artifacts: %s\n", suiteArtifactsPath)
	}

	// Return error if any tests failed
	if testSuiteResult.HasFailures() {
		return fmt.Errorf("tests failed in testsuite %s", filepath.Base(r.testSuite.File))
	}

	return nil
}

// runTestCase executes a single test case and returns a complete TestCaseResult.
func (r *Runner) runTestCase(testCase api.TestCase, index int, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	if r.Debug {
		utils.DebugPrintf("Starting test case '%s'\n", testCase.Name)
	}

	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, r.Verbose, r.ShowRender, r.ShowValidate, r.ShowHooks, r.ShowAssertions)

	// In update-goldens mode, skip immediately if there are no golden-file assertions to write.
	if r.UpdateGoldens && !testCase.HasAssertionsDiff() && !testCase.HasAssertionsDyff() {
		return result.Skip()
	}

	needsResults, skipReasons := r.getNeedsResults(testCase, testSuiteResult)
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

	if r.UpdateGoldens {
		return r.runUpdateGoldens(testCase, result)
	}

	if failed := r.validateTestCase(env, testCase, result); failed != nil {
		return failed
	}

	if failed := r.runAssertions(env, testCase, result); failed != nil {
		return failed
	}

	r.runPostTestHooks(testCase, needsResults, result)

	if err := r.exportOutputs(testCase, env, result); err != nil {
		return result.Fail(err)
	}

	// Fail with nil when the pipeline failed: what failed is in the result
	if result.HasPipelineFailure() {
		return result.Fail(nil)
	}

	// Complete the test case result
	if r.Debug {
		utils.DebugPrintf("Test case '%s' completed with status: %s\n", testCase.Name, result.Status.String())
	}

	return result.Complete()
}
