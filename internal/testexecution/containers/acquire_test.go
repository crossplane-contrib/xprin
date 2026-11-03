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
	"fmt"
	"sync"
	"testing"
	"time"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

// wait is how long a test waits for something that has to happen, and blocked is how long it waits before it
// decides that something that must not happen did not.
const (
	wait    = 5 * time.Second
	blocked = 50 * time.Millisecond
)

// acquireInBackground calls Acquire in a goroutine, and returns a channel that gets its release function once it
// returned.
func acquireInBackground(c *Coordinator, names ...string) <-chan func(bool) {
	got := make(chan func(bool), 1)

	go func() { got <- c.acquire(names) }()

	return got
}

func assertWaits(t *testing.T, got <-chan func(bool)) {
	t.Helper()

	select {
	case <-got:
		t.Fatal("Acquire returned, but it had to wait")
	case <-time.After(blocked):
	}
}

func assertReturns(t *testing.T, got <-chan func(bool)) func(bool) {
	t.Helper()

	select {
	case release := <-got:
		return release
	case <-time.After(wait):
		t.Fatal("Acquire did not return")
	}

	return nil
}

func TestCoordinator_Acquire_NilCoordinator(t *testing.T) {
	var c *Coordinator

	assert.NotPanics(t, func() { c.acquire([]string{"a"})(true) })
}

func TestCoordinator_Acquire_TheFirstUseOfAContainerMakesTheOthersWait(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	first := c.acquire([]string{"a"})
	second := acquireInBackground(c, "a")

	assertWaits(t, second)

	first(true) // the render created the container

	assertReturns(t, second)(true)

	// It exists now: nobody waits any more, even while another render that uses it runs
	assertReturns(t, acquireInBackground(c, "a"))
	assertReturns(t, acquireInBackground(c, "a"))
}

func TestCoordinator_Acquire_AfterAFailedRenderTheNextOneTriesAgain(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	c.acquire([]string{"a"})(false) // the render failed: the container may not exist

	second := c.acquire([]string{"a"})
	third := acquireInBackground(c, "a")

	assertWaits(t, third) // the second one is the first use now

	second(true)

	assertReturns(t, third)(true)
}

func TestCoordinator_Acquire_DifferentContainersDoNotWaitForEachOther(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	c.acquire([]string{"a"}) // never released: "a" is busy for the whole test

	assertReturns(t, acquireInBackground(c, "b"))
}

func TestCoordinator_Acquire_SeveralContainersInAnyOrderCanNotDeadlock(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	var wg sync.WaitGroup

	// Each round uses new containers, so that both test cases are the first use of both: if one took them
	// in the order it was given, and the other in the opposite order, each would hold one and wait for the other
	for round := range 500 {
		first, second := fmt.Sprintf("a%d", round), fmt.Sprintf("b%d", round)

		wg.Go(func() { c.acquire([]string{first, second})(true) })
		wg.Go(func() { c.acquire([]string{second, first})(true) })
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(wait):
		t.Fatal("the test cases deadlocked")
	}
}

func TestCoordinator_Acquire_ReleasingTwiceIsFine(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	release := c.acquire([]string{"a"})

	assert.NotPanics(t, func() {
		release(true)
		release(false)
	})
}

func TestCoordinator_Acquire_ANameMoreThanOnceDoesNotWaitForItself(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	assertReturns(t, acquireInBackground(c, "a", "b", "a"))
}
