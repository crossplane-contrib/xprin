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

import "sync"

// container is a named container, which RemoveAll removes and AnnotateAndAcquireFunctions waits for.
type container struct {
	name string
	// mu is held by the test case that is the first to use the container, until its render ends.
	mu sync.Mutex
	// created is true once a render that used the container succeeded. Only written with mu held.
	created bool
}

// lock waits for the container, and returns whether the caller holds it, which is not so when the container was
// created already: there is nothing to wait for.
func (c *container) lock() bool {
	c.mu.Lock()

	if c.created {
		c.mu.Unlock()

		return false
	}

	return true
}

// unlock releases the container that lock returned true for, and marks it as created when the command succeeded.
func (c *container) unlock(created bool) {
	if created {
		c.created = true
	}

	c.mu.Unlock()
}
