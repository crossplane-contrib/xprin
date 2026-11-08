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

// Package processor runs the tests of the testsuite files that the targets resolve to.
package processor

import (
	"errors"
	"fmt"

	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testexecution/runner"
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
)

// runnerInterface is what the processor needs from the runner of a testsuite file, and what its tests replace.
type runnerInterface interface {
	GetTestSuiteFilePath() string
	SetupTestSuite() error
	Job() scheduler.Job
}

// Mockable functions
//
//nolint:gochecknoglobals // Global variables for dependency injection in tests
var (
	newRunnerFunc = func(options *testexecutionOptions.Options, testSuite *testsuite.TestSuite) runnerInterface {
		return runner.NewRunner(options, testSuite)
	}
	newCoordinatorFunc = func() (*containers.Coordinator, error) {
		docker, err := containers.NewDocker()
		if err != nil {
			return nil, err
		}

		return containers.NewCoordinator(docker), nil
	}
)

// ProcessTargets processes the targets and runs the tests.
func ProcessTargets(fs afero.Fs, targets []string, options *testexecutionOptions.Options) error {
	// Containers tracks reuse of Docker containers for composition functions for this whole
	// invocation. Initialized here rather than required of callers, so every entry point into
	// test execution gets reuse enabled for free, unless the user opted out via
	// --no-container-reuse.
	if !options.NoContainerReuse {
		coordinator, err := newCoordinatorFunc()
		if err != nil {
			return fmt.Errorf("failed to set up container reuse: %w", err)
		}

		options.Containers = coordinator

		defer options.Containers.RemoveAll()
	}

	// The run has three steps:
	// 1. Find, load and set up every testsuite file. The scheduler needs all of them before it can start, to order the
	//    test cases and their results across the files, so this is done for all the files before any test case runs.
	//    What is wrong with the files is also reported first.
	// 2. Run the test cases of all the files, with the scheduler.
	// 3. Report what failed in the files, and print one FAIL.
	var (
		runners []runnerInterface
		jobs    []scheduler.Job
	)

	// A file that can't be loaded or set up is reported, left out, and makes the run fail
	walkErr := discovery.Walk(fs, targets, discoveryOptions(options), func(testSuiteFile string) error {
		testRunner, err := createRunner(fs, testSuiteFile, options)
		if err != nil || testRunner == nil {
			return err
		}

		if err := testRunner.SetupTestSuite(); err != nil {
			return runError(testSuiteFile, err)
		}

		runners = append(runners, testRunner)
		jobs = append(jobs, testRunner.Job())

		return nil
	})

	failed := walkErr != nil

	finishErrs, err := scheduler.Run(jobs, max(1, options.Parallel))
	if err != nil {
		utils.WarningPrintf("%v\n", err)

		failed = true
	}

	// finishErrs[i] is the result of the file of runners[i]
	for i, finishErr := range finishErrs {
		if runError(runners[i].GetTestSuiteFilePath(), finishErr) != nil {
			failed = true
		}
	}

	if failed {
		return discovery.FinalFail()
	}

	return nil
}

// discoveryOptions returns what the discovery package needs to know from the options.
func discoveryOptions(options *testexecutionOptions.Options) discovery.Options {
	return discovery.Options{Quiet: options.Quiet, Debug: options.Debug, DebugLog: options.DebugLog}
}

// createRunner loads the testsuite file, and creates its runner, which is not set up yet. It returns no runner, and
// no error, for a file that has no test cases.
func createRunner(fs afero.Fs, testSuiteFile string, options *testexecutionOptions.Options) (runnerInterface, error) {
	testSuite, err := discovery.Load(fs, testSuiteFile, discoveryOptions(options))
	if err != nil || testSuite == nil {
		return nil, err
	}

	return newRunnerFunc(options, testSuite), nil
}

// runError returns the error of running the test cases of a testsuite file, after reporting it when it is not about
// test cases that failed (they were reported as they ran).
func runError(testSuiteFile string, runErr error) error {
	if runErr == nil {
		return nil
	}

	if !errors.Is(runErr, runner.ErrTestsFailed) {
		return discovery.ReportTestSuiteError(testSuiteFile, runErr, "testsuite file execution error")
	}

	return fmt.Errorf("test execution failed for %s: %w", testSuiteFile, runErr)
}
