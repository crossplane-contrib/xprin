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
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/utils"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
)

// ErrTestsFailed is what Session.Finish wraps when some test cases failed.
var ErrTestsFailed = errors.New("tests failed")

// Runner runs test cases. It is created once for an invocation, and keeps nothing of the test cases that it runs.
type Runner struct {
	*testexecutionOptions.Options

	fs    afero.Fs
	mode  runnerMode
	debug *log.Logger
	// Mockable function fields
	runTestCaseFunc                   func(api.TestCase) *engine.TestCaseResult
	expandPathRelativeToTestSuiteFile func(base, path string) (string, error)
	verifyPathExists                  func(path string) error
	runCommand                        func(dir, name string, args ...string) ([]byte, error)
	copy                              func(src, dest string, opts ...cp.Options) error
	convertClaimToXRFunc              func(r *Runner, claimPath, xrdPath, goldenPath, outputPath string) (string, error)
	patchXRFunc                       func(r *Runner, xrPath, outputPath string, patches api.Patches) (string, error)
}

// testCaseEnv is what one test case run works with: the directories that are its own, and what it takes from the
// testsuite file that it is a part of. The phases of a test case read the file only from here.
type testCaseEnv struct {
	tmpDir     string
	inputsDir  string
	outputsDir string
	index      int

	testSuite       *testsuite.TestSuite // the file: where its paths are relative to, its spec and its dependency graph
	savedOutputsDir string               // where the outputs of the test cases that others use are saved
}

// NewRunner creates the runner of test cases, for one invocation. It keeps nothing of the test cases that it runs:
// what they share with their testsuite file is in the testCaseEnv that each one is run with.
func NewRunner(options *testexecutionOptions.Options) *Runner {
	r := &Runner{
		fs:      afero.NewOsFs(),
		Options: options,
		debug:   options.DebugLog,
		// Initialize mockable function fields with default implementations
		runTestCaseFunc:                   nil, // will set default below
		expandPathRelativeToTestSuiteFile: testsuite.ExpandPath,
		verifyPathExists:                  utils.VerifyPathExists,
		runCommand: func(dir, name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...) //nolint:gosec // G204 - Used for Crossplane CLI and hooks, caller is trusted
			cmd.Dir = dir

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

// runTestCase executes a single test case and returns a complete TestCaseResult.
//
// With --debug, the lines of the test case are printed as they happen when only one test case runs at a time. With
// --parallel they would be mixed with those of the others, so they are kept in the DebugLog of the result, and its
// session prints them with the result.
func (r *Runner) runTestCase(testCase api.TestCase, env *testCaseEnv, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	if !r.Debug || r.Parallel <= 1 {
		return r.executeTestCase(testCase, env, testSuiteResult)
	}

	// This copy of the runner is the one of this test case: it has its own debug log, and the mode that refers to it
	var lines strings.Builder

	own := *r
	own.debug = log.New(&lines, r.debug.Prefix(), r.debug.Flags())
	own.mode = newRunnerMode(&own)

	result := own.executeTestCase(testCase, env, testSuiteResult)
	result.DebugLog = lines.String()

	return result
}

func (r *Runner) executeTestCase(testCase api.TestCase, env *testCaseEnv, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	if r.Debug {
		r.debug.Printf("Starting test case '%s'\n", testCase.Name)
	}

	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, r.Verbose, r.ShowRender, r.ShowValidate, r.ShowHooks, r.ShowAssertions)

	// Skip immediately a test case that the mode has no use for, whatever it needs (the modes are in mode.go and
	// update_goldens.go)
	if r.mode.skips(env, testCase) {
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
		r.debug.Printf("Created temporary directory for test case: %s\n", env.tmpDir)
		r.debug.Printf("- Inputs: %s\n", env.inputsDir)
		r.debug.Printf("- Outputs: %s\n", env.outputsDir)
	}

	if env.testSuite.Spec.HasCommon() {
		testCase.MergeCommon(env.testSuite.Spec.Common)
	}

	// Process template variables for this test case
	if err := r.processTemplateVariables(env, &testCase, needsResults); err != nil {
		return result.Fail(fmt.Errorf("failed to process template variables: %w", err))
	}

	if err := testCase.CheckMandatoryFields(); err != nil {
		return result.Fail(err)
	}

	if r.Debug {
		r.debugPrintTestCase(testCase, "Test specification:")
	}

	if failed := r.expandInputPaths(env, &testCase, result); failed != nil {
		return failed
	}

	if failed := r.copyInputs(env, &testCase, result); failed != nil {
		return failed
	}

	if failed := r.runPreTestHooks(env, testCase, needsResults, result); failed != nil {
		return failed
	}

	if failed := r.renderTestCase(env, testCase, result); failed != nil {
		return failed
	}

	return r.mode.runPhasesAfterRender(testCase, env, needsResults, result)
}
