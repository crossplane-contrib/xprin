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
	"errors"
	"fmt"
	"sync"
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestCoordinator_ContainerName(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})

	name1 := c.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.0")
	name2 := c.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.0")
	name3 := c.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.7.0")

	assert.Equal(t, name1, name2, "the same image reference must always produce the same name")
	assert.NotEqual(t, name1, name3, "different image references must produce different names")
	assert.Contains(t, name1, "xprin-", "names should be namespaced with the xprin prefix")

	c2 := NewCoordinator(&unittestsUtils.MockDocker{})
	name4 := c2.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.0")
	assert.NotEqual(t, name1, name4, "two different coordinators (invocations) must produce different names for the same image")
}

func TestCoordinator_Names_EmptyWhenUnused(t *testing.T) {
	c := NewCoordinator(&unittestsUtils.MockDocker{})
	assert.Empty(t, c.names)
}

func TestCoordinator_RemoveAll(t *testing.T) {
	mockDocker := &unittestsUtils.MockDocker{}

	c := NewCoordinator(mockDocker)
	c.names["xprin-abc-aaaaaaaaaaaa"] = struct{}{}
	c.names["xprin-abc-bbbbbbbbbbbb"] = struct{}{}

	c.RemoveAll()

	assert.ElementsMatch(t, []string{"xprin-abc-aaaaaaaaaaaa", "xprin-abc-bbbbbbbbbbbb"}, mockDocker.RemovedContainers)
}

func TestCoordinator_RemoveAll_NothingTracked_NoCalls(t *testing.T) {
	mockDocker := &unittestsUtils.MockDocker{}

	NewCoordinator(mockDocker).RemoveAll()

	assert.Empty(t, mockDocker.RemovedContainers)
	assert.Empty(t, mockDocker.RemovedNetworks, "no network must be removed when NetworkName was never called")
}

func TestCoordinator_NetworkName_CreatesOnce(t *testing.T) {
	mockDocker := &unittestsUtils.MockDocker{}
	c := NewCoordinator(mockDocker)

	name1, err := c.NetworkName()
	require.NoError(t, err)

	name2, err := c.NetworkName()
	require.NoError(t, err)

	assert.Equal(t, name1, name2, "repeated calls must return the same network name")
	assert.Equal(t, []string{name1}, mockDocker.CreatedNetworks, "the network must only be created once")
	assert.Contains(t, name1, "xprin-net-", "network names should be namespaced with the xprin prefix")
}

func TestCoordinator_NetworkName_CreateError(t *testing.T) {
	wantErr := errors.New("boom")
	c := NewCoordinator(&unittestsUtils.MockDocker{CreateNetworkErr: wantErr})

	_, err := c.NetworkName()
	require.ErrorIs(t, err, wantErr)
	assert.Empty(t, c.networkName, "a failed creation must not be cached")
}

func TestCoordinator_RemoveAll_RemovesNetwork_WhenCreated(t *testing.T) {
	mockDocker := &unittestsUtils.MockDocker{}
	c := NewCoordinator(mockDocker)

	name, err := c.NetworkName()
	require.NoError(t, err)

	c.RemoveAll()

	assert.Equal(t, []string{name}, mockDocker.RemovedNetworks)
}

// Test cases that run at the same time share one coordinator: run with -race.
func TestCoordinator_ConcurrentUse(t *testing.T) {
	const testCases = 50

	mockDocker := &unittestsUtils.MockDocker{}
	c := NewCoordinator(mockDocker)

	var wg sync.WaitGroup

	for i := range testCases {
		wg.Go(func() {
			c.track(fmt.Sprintf("xprin-abc-%012d", i))

			_, err := c.NetworkName()
			assert.NoError(t, err)
		})
	}

	wg.Wait()

	assert.Len(t, mockDocker.CreatedNetworks, 1, "the network must be created once, whoever asks first")

	c.RemoveAll()

	assert.Len(t, mockDocker.RemovedContainers, testCases, "every container that was tracked is removed")
	assert.Len(t, mockDocker.RemovedNetworks, 1)
}
