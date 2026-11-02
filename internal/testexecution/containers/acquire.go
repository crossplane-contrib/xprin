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

package containers

import (
	"slices"
	"sync"
)

// acquire waits for the containers with the names, which are the ones that annotateFunctions returned, and keeps the ones
// that are not created yet until the function it returns is called (see AnnotateAndAcquireFunctions).
//
// The containers are locked in the order of their names, so that commands that use several of them can't deadlock.
func (c *Coordinator) acquire(names []string) (release func(created bool)) {
	if c == nil {
		return func(bool) {}
	}

	var held []*container

	for _, name := range sortedUnique(names) {
		if ctr := c.containerOf(name); ctr.lock() {
			held = append(held, ctr)
		}
	}

	var once sync.Once

	return func(created bool) {
		once.Do(func() {
			for _, ctr := range held {
				ctr.unlock(created)
			}
		})
	}
}

// sortedUnique returns the names in order, each once, and leaves names as it is.
func sortedUnique(names []string) []string {
	sorted := slices.Clone(names)
	slices.Sort(sorted)

	return slices.Compact(sorted)
}
