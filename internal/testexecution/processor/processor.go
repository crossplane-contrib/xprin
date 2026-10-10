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
	"fmt"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/discovery"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	"github.com/crossplane-contrib/xprin/internal/testexecution/runner"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
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
	newCoordinatorFunc = func() (*containers.Coordinator, error) {
		docker, err := containers.NewDocker()
		if err != nil {
			return nil, err
		}

		return containers.NewCoordinator(docker), nil
	}
)

// ProcessTargets processes the targets and runs the tests.
func ProcessTargets(fs afero.Fs, targets []string, options *testexecutionUtils.Options) error {
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

	return discovery.Walk(fs, targets, discoveryOptions(options), func(testSuiteFile string) error {
		return processTestSuiteFile(fs, testSuiteFile, options)
	})
}

// discoveryOptions returns what the discovery package needs to know from the options.
func discoveryOptions(options *testexecutionUtils.Options) discovery.Options {
	return discovery.Options{Quiet: options.Quiet, Debug: options.Debug}
}

// processTestSuiteFile processes a single test file, loading the configuration and running tests if applicable.
func processTestSuiteFile(fs afero.Fs, testSuiteFile string, options *testexecutionUtils.Options) error {
	testSuiteSpec, err := discovery.Load(fs, testSuiteFile, discoveryOptions(options))
	if err != nil || testSuiteSpec == nil {
		return err
	}

	testRunner := newRunnerFunc(options, testSuiteFile, testSuiteSpec)

	fileErr := testRunner.RunTests()
	if fileErr != nil {
		errMsg := fileErr.Error()
		if !strings.Contains(errMsg, "tests failed in testsuite") {
			return discovery.ReportTestSuiteError(testSuiteFile, fileErr, "testsuite file execution error")
		}

		return fmt.Errorf("test execution failed for %s: %w", testSuiteFile, fileErr)
	}

	return nil
}
