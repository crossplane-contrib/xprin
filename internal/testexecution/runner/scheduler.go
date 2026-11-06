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
	"runtime/debug"

	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/crossplane-contrib/xprin/internal/testexecution/scheduler"
)

// Job returns the runner as the scheduler sees it: the steps of the runner, and the dependency graph of its file.
// SetupTestSuite has to be called before the scheduler runs it.
func (r *Runner) Job() scheduler.Job {
	return &schedulerJob{r: r}
}

// schedulerJob gives the steps of a runner, and the dependency graph of its file, to the scheduler.
type schedulerJob struct {
	r *Runner
}

func (s *schedulerJob) Graph() *graph.Graph { return s.r.testSuite.Graph }

// Run runs the test case. A panic fails that test case only, so that the others, which run at the same time, finish.
func (s *schedulerJob) Run(positionInTestSuite int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.r.failPanicked(positionInTestSuite, recovered, debug.Stack())
		}
	}()

	s.r.runAndStoreTestCase(positionInTestSuite)
}

func (s *schedulerJob) Report(positionInTestSuite int) {
	s.r.printTestCaseResult(positionInTestSuite)
}

// Finish returns what completeTestSuite returned: nil, or ErrTestsFailed when some test case failed.
func (s *schedulerJob) Finish() error { return s.r.completeTestSuite() }

// failPanicked records that the test case at positionInTestSuite failed because of a panic, as its result.
func (r *Runner) failPanicked(positionInTestSuite int, recovered any, stack []byte) {
	testCase := r.testSuite.Spec.Tests[positionInTestSuite]
	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, r.Verbose, r.ShowRender, r.ShowValidate, r.ShowHooks, r.ShowAssertions)

	err := fmt.Errorf("panic: %v", recovered)
	if r.Debug {
		err = fmt.Errorf("%w\n%s", err, stack)
	}

	result.Fail(err)

	r.env.testCaseResults[positionInTestSuite] = result
	r.env.testSuiteResult.AddResult(result)
}
