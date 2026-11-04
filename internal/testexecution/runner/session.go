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

package runner

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/graph"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/gertd/go-pluralize"
	"github.com/spf13/afero"
)

// Session is the execution of one testsuite file: each testsuite file creates a session that runs as a job, and
// its test cases are the tasks of the job. The session holds what is of the file as a whole, which is the results so
// far, the directory of the outputs that test cases share, the order of the output and the summary. The runner runs
// each test case, and keeps nothing of the file.
// Setup has to be called before the scheduler runs it.
type Session struct {
	options   *testexecutionOptions.Options
	fs        afero.Fs
	runner    testCaseRunner
	testSuite *testsuite.TestSuite // the file that is run: its path, its spec and its dependency graph
	output    io.Writer
	debug     *log.Logger
	env       *testSuiteEnv
}

// testCaseRunner is what the session needs from the runner: to run one test case. The session makes the env of the test
// case, and gives it the results of the ones that already ran, for it to find the ones that the test case needs.
// It is the Runner, and the tests of the session replace it.
type testCaseRunner interface {
	RunTestCase(env *testCaseEnv, testCase api.TestCase, completed map[string]*engine.TestCaseResult) *engine.TestCaseResult
}

// NewSession creates the session of a testsuite file, which runs its test cases with runner. It is not set up yet.
func NewSession(options *testexecutionOptions.Options, testSuite *testsuite.TestSuite, runner testCaseRunner) *Session {
	return &Session{
		options:   options,
		fs:        afero.NewOsFs(),
		runner:    runner,
		testSuite: testSuite,
		output:    os.Stdout, // Default output to stdout
		debug:     options.DebugLog,
	}
}

// testSuiteEnv is the state of one run of a testsuite file, set by Setup.
type testSuiteEnv struct {
	savedOutputsDir string                   // the internal artifacts, the outputs that test cases share
	testSuiteResult *engine.TestSuiteResult  // the result of the file, which the result of each test case is added to
	testCaseResults []*engine.TestCaseResult // the result of each test case, by its positionInTestSuite, once it ran
}

// GetTestSuiteFilePath returns the path of the testsuite file of the session.
func (s *Session) GetTestSuiteFilePath() string {
	return s.testSuite.File
}

// Setup checks the testsuite file and makes the directories that the test cases use.
func (s *Session) Setup() (err error) {
	// Validate that testSuiteFile and testSuiteSpec are set (they should be set in NewSession)
	if s.testSuite.File == "" {
		return fmt.Errorf("testsuite file path is required")
	}

	if s.testSuite.Spec == nil {
		return fmt.Errorf("testsuite specification is required")
	}

	// Create the directory of the internal artifacts, the outputs that test cases share (always created, cleaned up when testsuite finishes)
	savedOutputsDir, err := afero.TempDir(s.fs, "", "xprin-internal-artifacts-")
	if err != nil {
		return fmt.Errorf("failed to create internal artifacts directory: %w", err)
	}

	s.env = &testSuiteEnv{savedOutputsDir: savedOutputsDir}

	// If it can't go on, nothing will call Finish to remove it
	defer func() {
		if err != nil {
			s.removeSavedOutputsDir()
		}
	}()

	if s.options.Debug {
		s.debug.Printf("Created internal artifacts directory: %s\n", savedOutputsDir)
	}

	if s.options.Debug {
		s.debug.Printf("Using testsuite file directory for relative path resolution: %s\n", s.testSuite.GetDir())

		// We do not expand or verify common fields here. They will be expanded and verified by the individual test cases that do not override them.
		if s.testSuite.Spec.HasCommon() {
			s.debugPrintCommon(s.testSuite.Spec.Common, "Found common configuration:")
		}

		plural := pluralize.NewClient()
		s.debug.Printf("Found %s\n", plural.Pluralize("test case", len(s.testSuite.Spec.Tests), true))
	}

	// Create test suite result
	s.env.testSuiteResult = engine.NewTestSuiteResult(s.testSuite.File, s.options.Verbose)
	s.env.testSuiteResult.SetWorkingDir(s.options.WorkingDir)

	totalTestCases := len(s.testSuite.Spec.Tests)
	s.env.testCaseResults = make([]*engine.TestCaseResult, totalTestCases)

	if s.options.ArtifactsBaseDir != "" && totalTestCases > 0 {
		if err := s.initArtifactsRunDir(); err != nil {
			return err
		}
	}

	return nil
}

// runAndStoreTestCase runs the test case at positionInTestSuite and stores its result.
// positionInTestSuite is the position of the test case in the tests of the file.
// Different test cases can run at the same time, once the ones that each needs are done.
func (s *Session) runAndStoreTestCase(positionInTestSuite int) {
	testCase := s.testSuite.Spec.Tests[positionInTestSuite]

	// Run the test and let the engine handle everything
	env := &testCaseEnv{
		index:           s.testSuite.Graph.PositionInOrder(positionInTestSuite) + 1,
		testSuite:       s.testSuite,
		savedOutputsDir: s.env.savedOutputsDir,
	}
	result := s.runner.RunTestCase(env, testCase, s.env.testSuiteResult.GetCompletedTests())

	s.env.testCaseResults[positionInTestSuite] = result
	s.env.testSuiteResult.AddResult(result)
}

// Report prints the stored result of the test case at positionInTestSuite.
// positionInTestSuite is the position of the test case in the tests of the file.
// Not at the same time as another one: the output has the order of the calls.
func (s *Session) Report(positionInTestSuite int) {
	result := s.env.testCaseResults[positionInTestSuite]

	// The --debug lines of a test case that ran at the same time as others come with its result, not mixed with theirs
	if result.DebugLog != "" {
		fmt.Fprint(s.debug.Writer(), result.DebugLog) //nolint:errcheck // output function, error handling not practical
	}

	result.Print(s.output)
}

// Finish prints the summary of the file, and returns nil, or ErrTestsFailed when some test case failed.
func (s *Session) Finish() error {
	defer s.removeSavedOutputsDir()

	// Complete the test suite result
	s.env.testSuiteResult.Complete()

	// Print only the file summary (not individual test results)
	s.env.testSuiteResult.Print(s.output)

	if s.options.ArtifactsRunDir != "" {
		suiteArtifactsPath := filepath.Join(s.options.ArtifactsRunDir, artifactsRelPath(s.testSuite.File, s.options.WorkingDir))
		fmt.Fprintf(os.Stderr, "artifacts: %s\n", suiteArtifactsPath)
	}

	// Return error if any tests failed
	if s.env.testSuiteResult.HasFailures() {
		return fmt.Errorf("%w in testsuite %s", ErrTestsFailed, filepath.Base(s.testSuite.File))
	}

	return nil
}

// removeSavedOutputsDir removes the directory of the internal artifacts.
// It is safe to call it more than once.
func (s *Session) removeSavedOutputsDir() {
	if s.env != nil && s.env.savedOutputsDir != "" {
		_ = s.fs.RemoveAll(s.env.savedOutputsDir)
	}
}

// Graph returns the dependency graph of the test cases of the file.
func (s *Session) Graph() *graph.Graph { return s.testSuite.Graph }

// Run runs the test case. A panic fails that test case only, so that the others, which run at the same time, finish.
func (s *Session) Run(positionInTestSuite int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.failPanicked(positionInTestSuite, recovered, debug.Stack())
		}
	}()

	s.runAndStoreTestCase(positionInTestSuite)
}

// failPanicked records that the test case at positionInTestSuite failed because of a panic, as its result.
func (s *Session) failPanicked(positionInTestSuite int, recovered any, stack []byte) {
	o := s.options
	testCase := s.testSuite.Spec.Tests[positionInTestSuite]
	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, o.Verbose, o.ShowRender, o.ShowValidate, o.ShowHooks, o.ShowAssertions)

	err := fmt.Errorf("panic: %v", recovered)
	if o.Debug {
		err = fmt.Errorf("%w\n%s", err, stack)
	}

	result.Fail(err)

	s.env.testCaseResults[positionInTestSuite] = result
	s.env.testSuiteResult.AddResult(result)
}
