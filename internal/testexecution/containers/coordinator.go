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
)

// Coordinator tracks Docker container names used for composition function reuse across one
// xprin test invocation.
type Coordinator struct {
	runID       string
	names       map[string]struct{}
	networkName string
	docker      Docker
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
		names:  make(map[string]struct{}),
		docker: docker,
	}
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
	for name := range c.names {
		c.docker.RemoveContainer(name)
	}

	if c.networkName != "" {
		c.docker.RemoveNetwork(c.networkName)
	}
}
