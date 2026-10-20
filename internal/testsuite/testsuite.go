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

// Package testsuite has the type of a testsuite file that was loaded: where it is, what it says, and the
// dependencies between its test cases.
package testsuite

import (
	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/graph"
)

// TestSuite is a testsuite file that was loaded.
type TestSuite struct {
	// File is the path of the testsuite file, as it was given to discovery.
	File string
	// Spec is what the file says.
	Spec *api.TestSuiteSpec
	// Graph is the dependencies between the test cases of Spec. It is nil when there is no Spec.
	Graph *graph.Graph
}

// New returns the test suite of a file and its spec, with the dependency graph of the spec.
func New(file string, spec *api.TestSuiteSpec) *TestSuite {
	testSuite := &TestSuite{File: file, Spec: spec}

	if spec != nil {
		testSuite.Graph = graph.Build(spec)
	}

	return testSuite
}
