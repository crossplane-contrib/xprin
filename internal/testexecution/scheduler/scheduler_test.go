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

package scheduler

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// wait is how long a test waits for something that has to happen, and blocked is how long it waits before it
// decides that something that must not happen did not.
const (
	wait    = 5 * time.Second
	blocked = 50 * time.Millisecond
)

// log records, in order, what the scheduler did to the Jobs: "A start 2", "A end 2", "A report 2", "A finish".
type log struct {
	mu     sync.Mutex
	events []string
}

func (l *log) addf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, fmt.Sprintf(format, args...))
}

func (l *log) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.events)
}

// with returns the events that contain what, in order.
func (l *log) with(what string) []string {
	var found []string

	for _, e := range l.all() {
		if strings.Contains(e, what) {
			found = append(found, e)
		}
	}

	return found
}

// output returns what the Jobs print: the reports and the ends of the files, in order.
func (l *log) output() []string {
	var found []string

	for _, e := range l.all() {
		if strings.Contains(e, " report ") || strings.HasSuffix(e, " finish") {
			found = append(found, e)
		}
	}

	return found
}

// index returns where the event is, and fails if it is not there.
func (l *log) index(t *testing.T, event string) int {
	t.Helper()

	i := slices.Index(l.all(), event)
	require.GreaterOrEqual(t, i, 0, "no event %q in %v", event, l.all())

	return i
}

// fakeJob is a Job that logs what is done to it.
type fakeJob struct {
	name  string
	log   *log
	needs [][]int // needs[i] are the needs of the task i; its length is the number of tasks
	run   func(i int)
	// finishErr is what Finish returns
	finishErr error
}

// Graph returns the graph of a testsuite file whose test cases are the tasks.
func (f *fakeJob) Graph() *graph.Graph {
	spec := &api.TestSuiteSpec{}

	for i, needs := range f.needs {
		testCase := api.TestCase{Name: fmt.Sprintf("task %d", i), ID: fmt.Sprintf("t%d", i)}
		for _, need := range needs {
			testCase.Needs = append(testCase.Needs, fmt.Sprintf("t%d", need))
		}

		spec.Tests = append(spec.Tests, testCase)
	}

	return graph.Build(spec)
}

func (f *fakeJob) Run(i int) {
	f.log.addf("%s start %d", f.name, i)

	if f.run != nil {
		f.run(i)
	}

	f.log.addf("%s end %d", f.name, i)
}

func (f *fakeJob) Report(i int) { f.log.addf("%s report %d", f.name, i) }

func (f *fakeJob) Finish() error {
	f.log.addf("%s finish", f.name)

	return f.finishErr
}

// independent returns the needs of n tasks that don't need anything.
func independent(n int) [][]int { return make([][]int, n) }

// runInBackground runs the Jobs, and returns a channel that gets the error of Run when it returns.
func runInBackground(jobs []Job, workers int) <-chan error {
	done := make(chan error, 1)

	go func() {
		_, err := Run(jobs, workers)
		done <- err
	}()

	return done
}

// mustRun runs the Jobs, and returns what each Finish returned. Run must not fail.
func mustRun(t *testing.T, jobs []Job, workers int) []error {
	t.Helper()

	finishErrs, err := Run(jobs, workers)
	require.NoError(t, err)

	return finishErrs
}

func awaitRun(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(wait):
		t.Fatal("Run did not return")
	}

	return nil
}

// eventually fails the test unless cond becomes true.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(wait)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("%s did not happen", what)
		}

		time.Sleep(time.Millisecond)
	}
}

func TestRun_OneWorkerRunsInTheSerialOrder(t *testing.T) {
	l := &log{}
	// 0 and 1 need 2, which is listed after them: the order of a serial run is 2, 0, 1
	a := &fakeJob{name: "A", log: l, needs: [][]int{{2}, {2}, {}}}
	b := &fakeJob{name: "B", log: l, needs: independent(2)}

	mustRun(t, []Job{a, b}, 1)

	assert.Equal(t, []string{"A start 2", "A start 0", "A start 1", "B start 0", "B start 1"}, l.with(" start "))
	assert.Equal(t, []string{"A report 2", "A report 0", "A report 1", "B report 0", "B report 1"}, l.with(" report "))
}

func TestRun_WorkersLessThanOneIsOne(t *testing.T) {
	l := &log{}
	a := &fakeJob{name: "A", log: l, needs: independent(3)}

	mustRun(t, []Job{a}, 0)

	assert.Equal(t, []string{"A report 0", "A report 1", "A report 2"}, l.with(" report "))
}

func TestRun_ATestCaseStartsAfterTheOnesItNeedsFinished(t *testing.T) {
	l := &log{}
	// 1 needs 0, 2 needs 1 and 3, and 3 is listed after the one that needs it
	a := &fakeJob{
		name:  "A",
		log:   l,
		needs: [][]int{{}, {0}, {1, 3}, {}},
		run:   func(int) { time.Sleep(5 * time.Millisecond) },
	}

	mustRun(t, []Job{a}, 4)

	assert.Less(t, l.index(t, "A end 0"), l.index(t, "A start 1"))
	assert.Less(t, l.index(t, "A end 1"), l.index(t, "A start 2"))
	assert.Less(t, l.index(t, "A end 3"), l.index(t, "A start 2"))
}

func TestRun_RunsAtTheSameTimeUpToTheWorkers(t *testing.T) {
	const workers = 3

	var running, most atomic.Int32

	release := make(chan struct{})
	started := make(chan struct{}, 10)

	// The first tasks don't return until the workers are all busy: it only returns if they run together
	a := &fakeJob{name: "A", log: &log{}, needs: independent(8)}
	a.run = func(i int) {
		now := running.Add(1)
		defer running.Add(-1)

		for {
			old := most.Load()
			if now <= old || most.CompareAndSwap(old, now) {
				break
			}
		}

		if i < workers {
			started <- struct{}{}

			<-release
		}
	}

	done := runInBackground([]Job{a}, workers)

	for range workers {
		select {
		case <-started:
		case <-time.After(wait):
			t.Fatal("the tasks did not run at the same time")
		}
	}

	close(release)
	require.NoError(t, awaitRun(t, done))

	assert.Equal(t, int32(workers), most.Load(), "never more than the workers at the same time")
}

func TestRun_ReportsInOrderWhateverTheOrderTheyFinishIn(t *testing.T) {
	l := &log{}
	release := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	a := &fakeJob{name: "A", log: l, needs: independent(3), run: func(i int) { <-release[i] }}

	done := runInBackground([]Job{a}, 3)

	eventually(t, "the three tasks starting", func() bool { return len(l.with(" start ")) == 3 })

	// The last one finishes first, then the middle one: nothing is reported, the first one is not done
	close(release[2])
	close(release[1])
	eventually(t, "the two tasks ending", func() bool { return len(l.with(" end ")) == 2 })
	time.Sleep(blocked)
	assert.Empty(t, l.with(" report "), "nothing is reported before the task that comes first")

	close(release[0])
	require.NoError(t, awaitRun(t, done))

	assert.Equal(t, []string{"A report 0", "A report 1", "A report 2", "A finish"}, l.output(), "in order, and then the end of the file")
}

func TestRun_ReportsAfterTheRunAndNeverAtTheSameTime(t *testing.T) {
	const tests = 40

	var reporting atomic.Int32

	l := &log{}
	ran := [2][tests]atomic.Bool{}
	jobs := make([]Job, 2)

	for s := range jobs {
		f := &fakeJob{name: fmt.Sprintf("S%d", s), log: l, needs: independent(tests)}
		f.run = func(i int) {
			time.Sleep(time.Duration(i%3) * time.Millisecond)
			ran[s][i].Store(true)
		}
		jobs[s] = &checkedSuite{fakeJob: f, check: func(i int) {
			assert.EqualValues(t, 1, reporting.Add(1), "two reports at the same time")
			assert.True(t, ran[s][i].Load(), "reported before it ran")
			time.Sleep(100 * time.Microsecond)
			reporting.Add(-1)
		}}
	}

	mustRun(t, jobs, 8)

	assert.Len(t, l.with(" report "), 2*tests)
}

// checkedSuite runs check at every Report.
type checkedSuite struct {
	*fakeJob

	check func(i int)
}

func (c *checkedSuite) Report(i int) {
	c.check(i)
	c.fakeJob.Report(i)
}

func TestRun_AFileEndsAfterItsLastReportAndBeforeTheNextOne(t *testing.T) {
	l := &log{}
	a := &fakeJob{name: "A", log: l, needs: independent(2)}
	empty := &fakeJob{name: "E", log: l}
	b := &fakeJob{name: "B", log: l, needs: independent(2)}

	mustRun(t, []Job{a, empty, b}, 4)

	assert.Equal(t, []string{"A report 0", "A report 1", "A finish", "E finish", "B report 0", "B report 1", "B finish"}, l.output())
}

func TestRun_JobsWithoutTestCases(t *testing.T) {
	l := &log{}

	mustRun(t, []Job{&fakeJob{name: "A", log: l}, &fakeJob{name: "B", log: l}}, 4)
	assert.Equal(t, []string{"A finish", "B finish"}, l.output())
}

func TestRun_NothingToRun(t *testing.T) {
	finishErrs := mustRun(t, nil, 4)

	assert.Empty(t, finishErrs)
}

func TestRun_ReturnsWhatEachFinishReturned(t *testing.T) {
	l := &log{}
	failed := errors.New("it failed")
	a := &fakeJob{name: "A", log: l, needs: independent(2)}
	b := &fakeJob{name: "B", log: l, needs: independent(1), finishErr: failed}
	empty := &fakeJob{name: "empty", log: l, finishErr: failed}
	c := &fakeJob{name: "C", log: l, needs: independent(1)}

	finishErrs := mustRun(t, []Job{a, b, empty, c}, 4)

	require.Len(t, finishErrs, 4)
	require.NoError(t, finishErrs[0])
	assert.Equal(t, failed, finishErrs[1], "in the position of its job")
	assert.Equal(t, failed, finishErrs[2], "also a job without tasks")
	require.NoError(t, finishErrs[3])
}

func TestRun_TheErrorOfAFinishIsNotAnErrorOfRun(t *testing.T) {
	l := &log{}
	a := &fakeJob{name: "A", log: l, needs: independent(1), finishErr: errors.New("it failed")}
	b := &fakeJob{name: "B", log: l, needs: independent(1)}

	finishErrs, err := Run([]Job{a, b}, 1)

	require.NoError(t, err)
	assert.Equal(t, []string{"A report 0", "A finish", "B report 0", "B finish"}, l.output(), "the next job is not affected")
	assert.Len(t, finishErrs, 2)
}

func TestRun_TheTestCasesOfTheNextFileUseTheFreeWorkers(t *testing.T) {
	l := &log{}
	release := make(chan struct{})
	slow := &fakeJob{name: "A", log: l, needs: independent(1), run: func(int) { <-release }}
	quick := &fakeJob{name: "B", log: l, needs: independent(3)}

	done := runInBackground([]Job{slow, quick}, 2)

	// The first file has one task, busy: the other worker runs the ones of the second file
	eventually(t, "the second file running", func() bool { return len(l.with("B end ")) == 3 })
	assert.Empty(t, l.output(), "but nothing is reported before the first file")

	close(release)
	require.NoError(t, awaitRun(t, done))

	assert.Equal(t, []string{"A report 0", "A finish", "B report 0", "B report 1", "B report 2", "B finish"}, l.output())
}

func TestRun_TestCasesThatWaitForEachOtherAreAnError(t *testing.T) {
	l := &log{}
	// 0 and 1 need each other, 2 needs 1, and 3 is free
	a := &fakeJob{name: "A", log: l, needs: [][]int{{1}, {0}, {1}, {}}}

	err := awaitRun(t, runInBackground([]Job{a}, 2))

	require.ErrorContains(t, err, "wait for each other")
	assert.Equal(t, []string{"A start 3", "A end 3", "A report 3"}, l.with(" 3"), "what is not stuck runs, and is reported")
	assert.Empty(t, l.with(" start 0"))
	assert.Empty(t, l.with(" finish"), "the file does not end, as some of its tasks never ran")
}
