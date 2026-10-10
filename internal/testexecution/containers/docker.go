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
	"context"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
)

// Docker is the subset of the Docker API the Coordinator needs, so tests can mock it without a
// real Docker daemon.
type Docker interface {
	// RemoveContainer force-removes a container by name, ignoring errors (best-effort cleanup).
	RemoveContainer(name string)
	// CreateNetwork creates a network by name.
	CreateNetwork(name string) error
	// RemoveNetwork removes a network by name, ignoring errors (best-effort cleanup).
	RemoveNetwork(name string)
}

type dockerSDK struct {
	cli *dockerclient.Client
}

// NewDocker returns a Docker backed by the Docker Go SDK, configured from the environment.
func NewDocker() (Docker, error) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &dockerSDK{cli: cli}, nil
}

func (d *dockerSDK) RemoveContainer(name string) {
	_ = d.cli.ContainerRemove(context.Background(), name, dockercontainer.RemoveOptions{Force: true})
}

func (d *dockerSDK) CreateNetwork(name string) error {
	_, err := d.cli.NetworkCreate(context.Background(), name, dockernetwork.CreateOptions{})

	return err
}

func (d *dockerSDK) RemoveNetwork(name string) {
	_ = d.cli.NetworkRemove(context.Background(), name)
}
