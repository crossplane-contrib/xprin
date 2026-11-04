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
	"os"
	"path/filepath"

	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/gertd/go-pluralize"
	"github.com/spf13/afero"
)

// Session is the execution of one testsuite file: each testsuite file creates a session that runs as a job, and
// its test cases are the tasks of the job. The session holds what is of the file as a whole, which is the results so
// far, the directory of the outputs that test cases share, the order of the output and the summary. The Runner runs
// each test case, and keeps nothing of the file.
type Session struct {
	r         *Runner
	testSuite *testsuite.TestSuite // the file that is run: its path, its spec and its dependency graph
	output    io.Writer
	env       *testSuiteEnv
}

// NewSession creates the session of a testsuite file, which runs its test cases with r. It is not set up yet.
func NewSession(r *Runner, testSuite *testsuite.TestSuite) *Session {
	return &Session{
		r:         r,
		testSuite: testSuite,
		output:    os.Stdout, // Default output to stdout
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
	savedOutputsDir, err := afero.TempDir(s.r.fs, "", "xprin-internal-artifacts-")
	if err != nil {
		return fmt.Errorf("failed to create internal artifacts directory: %w", err)
	}

	s.env = &testSuiteEnv{savedOutputsDir: savedOutputsDir}

	// If it can't go on, nothing will call completeTestSuite to remove it
	defer func() {
		if err != nil {
			s.removeSavedOutputsDir()
		}
	}()

	if s.r.Debug {
		s.r.debug.Printf("Created internal artifacts directory: %s\n", savedOutputsDir)
	}

	if s.r.Debug {
		s.r.debug.Printf("Using testsuite file directory for relative path resolution: %s\n", s.testSuite.GetDir())

		// We do not expand or verify common fields here. They will be expanded and verified by the individual test cases that do not override them.
		if s.testSuite.Spec.HasCommon() {
			s.r.debugPrintCommon(s.testSuite.Spec.Common, "Found common configuration:")
		}

		plural := pluralize.NewClient()
		s.r.debug.Printf("Found %s\n", plural.Pluralize("test case", len(s.testSuite.Spec.Tests), true))
	}

	// Create test suite result
	s.env.testSuiteResult = engine.NewTestSuiteResult(s.testSuite.File, s.r.Verbose)
	s.env.testSuiteResult.SetWorkingDir(s.r.WorkingDir)

	totalTestCases := len(s.testSuite.Spec.Tests)
	s.env.testCaseResults = make([]*engine.TestCaseResult, totalTestCases)

	if s.r.ArtifactsBaseDir != "" && totalTestCases > 0 {
		if err := s.r.initArtifactsRunDir(); err != nil {
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
	result := s.r.runTestCase(testCase, env, s.env.testSuiteResult)

	s.env.testCaseResults[positionInTestSuite] = result
	s.env.testSuiteResult.AddResult(result)
}

// printTestCaseResult prints the stored result of the test case at positionInTestSuite.
// positionInTestSuite is the position of the test case in the tests of the file.
// Not at the same time as another one: the output has the order of the calls.
func (s *Session) printTestCaseResult(positionInTestSuite int) {
	result := s.env.testCaseResults[positionInTestSuite]

	// The --debug lines of a test case that ran at the same time as others come with its result, not mixed with theirs
	if result.DebugLog != "" {
		fmt.Fprint(s.r.debug.Writer(), result.DebugLog) //nolint:errcheck // output function, error handling not practical
	}

	result.Print(s.output)
}

// completeTestSuite prints the summary of the file, and returns an error if any test case failed.
func (s *Session) completeTestSuite() error {
	defer s.removeSavedOutputsDir()

	// Complete the test suite result
	s.env.testSuiteResult.Complete()

	// Print only the file summary (not individual test results)
	s.env.testSuiteResult.Print(s.output)

	if s.r.ArtifactsRunDir != "" {
		suiteArtifactsPath := filepath.Join(s.r.ArtifactsRunDir, artifactsRelPath(s.testSuite.File, s.r.WorkingDir))
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
		_ = s.r.fs.RemoveAll(s.env.savedOutputsDir)
	}
}
