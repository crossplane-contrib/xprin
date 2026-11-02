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

// Package containers coordinates reuse of Docker containers for composition functions across
// one xprin test invocation, and patches Function resources' annotations to opt into it.
package containers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/spf13/afero"
)

// Coordinator tracks Docker container names used for composition function reuse across one xprin test invocation.
// It is safe for concurrent use: test cases that run at the same time share it.
type Coordinator struct {
	runID  string
	docker Docker

	mu          sync.Mutex // guards containers and networkName
	containers  []*container
	networkName string
}

// NewCoordinator creates an empty coordinator scoped to one xprin test invocation. The
// generated run ID namespaces every container name this coordinator produces, isolating this
// invocation's containers from any other xprin test invocation running on the same host. docker
// performs the container/network operations the coordinator needs.
func NewCoordinator(docker Docker) *Coordinator {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	return &Coordinator{
		runID:  hex.EncodeToString(b),
		docker: docker,
	}
}

// AnnotateAndAcquireFunctions is called before a command that uses the containers of the Functions in functionsPath (a file or a
// directory of files), for example `crossplane render`.
// It sets the annotations of their containers (see annotateFunctions), and waits for the containers that another command is
// creating: the first command that uses a container creates it, and the others that use it wait until that command is done.
// The function it returns is called when the command is done, with whether it succeeded.
// Only a success marks the containers as existing, so after a failure the next command tries again.
// It is safe to call it on a nil Coordinator (container reuse is off): there is nothing to set or to wait for.
func (c *Coordinator) AnnotateAndAcquireFunctions(fs afero.Fs, functionsPath string) (release func(created bool), err error) {
	names, err := c.annotateFunctions(fs, functionsPath)
	if err != nil {
		return nil, err
	}

	return c.acquire(names), nil
}

// containerOf returns the container with the name, which is added if there is none yet.
// The containers it knows are the ones that RemoveAll removes.
func (c *Coordinator) containerOf(name string) *container {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ctr := range c.containers {
		if ctr.name == name {
			return ctr
		}
	}

	ctr := &container{name: name}
	c.containers = append(c.containers, ctr)

	return ctr
}

// ContainerName returns the container name for the given function image reference. The same
// image reference always maps to the same name within this coordinator; different image
// references — even for functions with the same metadata.name — always map to different names.
func (c *Coordinator) ContainerName(imageRef string) string {
	sum := sha256.Sum256([]byte(imageRef))
	return "xprin-" + c.runID + "-" + hex.EncodeToString(sum[:])[:12]
}

// NetworkName returns the name of this invocation's shared Docker network, creating it on first
// use. crossplane's dockerized render engine otherwise creates its own new, uniquely-named Docker
// network for each `crossplane render` call and only connects that call's own containers to it —
// a container reused from an earlier call is left on the earlier call's network and unreachable.
// Passing this shared network to every render call via --crossplane-docker-network keeps a reused
// container reachable for as long as the container itself is reused.
func (c *Coordinator) NetworkName() (string, error) {
	// Held while the network is created, so that two test cases don't create it both
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.networkName != "" {
		return c.networkName, nil
	}

	name := "xprin-net-" + c.runID

	if err := c.docker.CreateNetwork(name); err != nil {
		return "", err
	}

	c.networkName = name

	return name, nil
}

// RemoveAll removes every container tracked by this coordinator, and the shared network from
// NetworkName, if one was created. Meant to be called once, via defer, when the whole xprin test
// invocation is about to exit.
func (c *Coordinator) RemoveAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ctr := range c.containers {
		c.docker.RemoveContainer(ctr.name)
	}

	if c.networkName != "" {
		c.docker.RemoveNetwork(c.networkName)
	}
}
