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

// Package scheduler runs the tasks of several Jobs with a number of workers, each task after the ones it needs, and
// reports the results in a fixed order. In xprin a Job is a testsuite file, and a task is one of its test cases.
//
// Tasks of different Jobs can run at the same time, so a free worker takes the first task of the next Job while the
// last task of the previous one is still running.
package scheduler

import (
	"container/heap"
	"fmt"
	"sync"

	"github.com/crossplane-contrib/xprin/internal/graph"
)

// Job is a set of tasks that depend on each other: in xprin, the test cases of one testsuite file.
// A task is identified by its position in the Job, which is its position in the graph.
// The Job keeps the results of Run, for Report.
type Job interface {
	// Graph returns the dependency graph of the tasks: what each one needs, and the order of a serial run.
	// That order is the order of the reports, and of the tasks that can start.
	Graph() *graph.Graph

	// Run runs task i.
	// It is called from the workers, so for several tasks of the Job at the same time.
	Run(i int)
	// Report is called once for each task after its Run, in the order of the graph.
	// It waits until all the tasks before it were reported, in this Job and in the Jobs before it.
	// It is never called at the same time as another Report or Finish.
	Report(i int)
	// Finish is called once after the last Report of the Job (at once, if it has no tasks), and before anything of
	// the next Job is reported.
	// The package-level Run returns what it returns, in finishErrs.
	Finish() error
}

// Run runs the tasks of the Jobs, with at most workers at the same time, and returns when all were run and reported.
// A task starts once the tasks it needs have run, passed or not: what to do about that is the Job's to decide. Of the
// tasks that can start, the first goes first: the first Job, then its graph order. With one worker, exactly in that order.
//
// finishErrs[i] is what Finish returned for jobs[i]. err is set when tasks wait for each other and can never start
// (a bug: a file with a cycle of `needs` is rejected when it is loaded). A panic in a Job is not recovered.
func Run(jobs []Job, workers int) (finishErrs []error, err error) {
	s := newScheduler(jobs)

	// A Job without tasks finishes before anything runs
	s.flush()

	workers = max(1, min(workers, s.remaining))

	var wg sync.WaitGroup

	for range workers {
		wg.Go(s.work)
	}

	wg.Wait()

	if s.remaining > 0 {
		return s.finishErrs, fmt.Errorf("%d tasks wait for each other and can never start", s.remaining)
	}

	return s.finishErrs, nil
}

// scheduler holds the state of one Run.
type scheduler struct {
	jobs []Job
	// graphs[i] is the graph of jobs[i], order[i] its order: the positions of its tasks, in the order they run, and
	// taskCount[i] its number of tasks.
	graphs    []*graph.Graph
	order     [][]int
	taskCount []int

	// mu guards the fields up to flushMu.
	mu         sync.Mutex
	cond       *sync.Cond
	ready      queue
	waitingFor map[task]int    // how many of its needs a task still waits for
	dependents map[task][]task // the tasks that wait for a task
	finished   [][]bool        // finished[i][rank] tells whether the task at that place in order[i] ran
	running    int
	remaining  int // tasks that did not run yet

	// flushMu keeps the reports in order, as only one flush runs at a time. The lock order is flushMu, then mu.
	flushMu sync.Mutex
	// cursor is the next report: the task at rank of jobs[cursor.jobIndex], or the end of that Job when rank is its
	// number of tasks.
	cursor     struct{ jobIndex, rank int }
	finishErrs []error // what Finish returned, by jobIndex
}

func newScheduler(jobs []Job) *scheduler {
	s := &scheduler{
		jobs:       jobs,
		graphs:     make([]*graph.Graph, len(jobs)),
		order:      make([][]int, len(jobs)),
		taskCount:  make([]int, len(jobs)),
		finished:   make([][]bool, len(jobs)),
		finishErrs: make([]error, len(jobs)),
		waitingFor: make(map[task]int),
		dependents: make(map[task][]task),
	}
	s.cond = sync.NewCond(&s.mu)

	for jobIndex, job := range jobs {
		s.add(jobIndex, job)
	}

	return s
}

// add reads the graph of jobs[jobIndex]: the order of its tasks, and what each one waits for.
// The needs of a task are positions of its own Job, so each Job is read on its own.
// A task in a cycle of needs is not in the order and never starts: Run reports it.
func (s *scheduler) add(jobIndex int, job Job) {
	g := job.Graph()
	nodes := g.Nodes()

	s.graphs[jobIndex] = g
	s.order[jobIndex] = g.GetOrder()
	s.taskCount[jobIndex] = len(nodes)
	s.finished[jobIndex] = make([]bool, len(nodes))
	s.remaining += len(nodes)

	for position, node := range nodes {
		t := s.taskAt(jobIndex, position)

		for _, need := range node.Needs {
			s.waitingFor[t]++
			dependency := s.taskAt(jobIndex, need)
			s.dependents[dependency] = append(s.dependents[dependency], t)
		}

		if s.waitingFor[t] == 0 {
			heap.Push(&s.ready, t)
		}
	}
}

// taskAt returns the task at position of jobs[jobIndex].
func (s *scheduler) taskAt(jobIndex, position int) task {
	return task{jobIndex, position, s.graphs[jobIndex].PositionInOrder(position)}
}

// work runs the tasks that can start, until there are none left.
func (s *scheduler) work() {
	for {
		t, ok := s.next()
		if !ok {
			return
		}

		s.jobs[t.jobIndex].Run(t.position)
		s.finish(t)
	}
}

// next waits for a task that can start.
// It returns false when there is none and there never will be: all ran, or what is left waits for itself.
func (s *scheduler) next() (task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if len(s.ready) > 0 {
			s.running++

			return heap.Pop(&s.ready).(task), true //nolint:forcetypeassert // only tasks are pushed
		}

		// Nothing can start, and nothing running can change that
		if s.running == 0 {
			return task{}, false
		}

		s.cond.Wait()
	}
}

// finish records that t ran, makes ready what waited only for it, and reports what can be reported.
func (s *scheduler) finish(t task) {
	s.mu.Lock()

	s.running--
	s.remaining--
	s.finished[t.jobIndex][t.rank] = true

	for _, dependent := range s.dependents[t] {
		if s.waitingFor[dependent]--; s.waitingFor[dependent] == 0 {
			heap.Push(&s.ready, dependent)
		}
	}

	s.cond.Broadcast()
	s.mu.Unlock()

	s.flush()
}

// flush reports, from the cursor on, the tasks that ran and the end of each Job that is done.
func (s *scheduler) flush() {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	for s.cursor.jobIndex < len(s.jobs) {
		job := s.jobs[s.cursor.jobIndex]

		if s.cursor.rank == s.taskCount[s.cursor.jobIndex] {
			s.finishErrs[s.cursor.jobIndex] = job.Finish()

			s.cursor.jobIndex++
			s.cursor.rank = 0

			continue
		}

		if !s.isFinished(s.cursor.jobIndex, s.cursor.rank) {
			return
		}

		job.Report(s.order[s.cursor.jobIndex][s.cursor.rank])

		s.cursor.rank++
	}
}

func (s *scheduler) isFinished(jobIndex, rank int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.finished[jobIndex][rank]
}
