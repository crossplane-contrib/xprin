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

package utils

// MockDocker is a recording mock of containers.Docker, for tests that must not touch a real
// Docker daemon.
type MockDocker struct {
	RemovedContainers []string
	CreatedNetworks   []string
	RemovedNetworks   []string
	// CreateNetworkErr, if set, is returned by CreateNetwork (and nothing is recorded).
	CreateNetworkErr error
}

// RemoveContainer records name.
func (m *MockDocker) RemoveContainer(name string) {
	m.RemovedContainers = append(m.RemovedContainers, name)
}

// CreateNetwork records name, or returns CreateNetworkErr if set.
func (m *MockDocker) CreateNetwork(name string) error {
	if m.CreateNetworkErr != nil {
		return m.CreateNetworkErr
	}

	m.CreatedNetworks = append(m.CreatedNetworks, name)

	return nil
}

// RemoveNetwork records name.
func (m *MockDocker) RemoveNetwork(name string) {
	m.RemovedNetworks = append(m.RemovedNetworks, name)
}
