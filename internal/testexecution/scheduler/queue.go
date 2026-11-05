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

// task is the task at position of jobs[jobIndex]. rank is where position is in the Order of that Job.
type task struct {
	jobIndex, position, rank int
}

func (t task) before(other task) bool {
	return t.jobIndex < other.jobIndex || (t.jobIndex == other.jobIndex && t.rank < other.rank)
}

// queue holds the tasks that can start, the first first.
//
//nolint:recvcheck // heap.Interface: Push and Pop change the slice, so they need the pointer
type queue []task

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].before(q[j]) }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(task)) } //nolint:forcetypeassert // only tasks are pushed
func (q *queue) Pop() any {
	old := *q
	last := old[len(old)-1]
	*q = old[:len(old)-1]

	return last
}
