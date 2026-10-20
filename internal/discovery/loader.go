/*
Copyright 2025 The Crossplane Authors.

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

package discovery

import (
	"fmt"
	"os"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/graph"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// failureInvalid is the failure reason reported for a testsuite file that can not be loaded.
const failureInvalid = "invalid testsuite file"

// TestSuite is a testsuite file that was loaded and checked: what it says, and the dependencies between
// its test cases.
type TestSuite struct {
	Spec  *api.TestSuiteSpec
	Graph *graph.Graph
}

// Load reads, parses and validates a testsuite file. The test suite is nil, without an error, when the
// file has no test cases (which is reported, unless quiet). Other problems are reported and returned.
func Load(fs afero.Fs, testSuiteFile string, options Options) (*TestSuite, error) {
	if options.Debug {
		utils.DebugPrintf("Processing testsuite file %s\n", testSuiteFile)
	}

	data, err := afero.ReadFile(fs, testSuiteFile)
	if err != nil {
		return nil, ReportTestSuiteError(testSuiteFile, fmt.Errorf("failed to read testsuite file %s: %w", testSuiteFile, err), failureInvalid)
	}

	content := string(data)

	if strings.Contains(content, "{{") {
		content = placeholder.Replace(content)
	}

	var testSuiteSpec api.TestSuiteSpec
	if err := yaml.Unmarshal([]byte(content), &testSuiteSpec); err != nil {
		return nil, ReportTestSuiteError(testSuiteFile, fmt.Errorf("failed to parse testsuite file %s: %w", testSuiteFile, err), failureInvalid)
	}

	if len(testSuiteSpec.Tests) == 0 {
		if !options.Quiet {
			fmt.Fprintf(os.Stderr, "?   \t%s\t[no test cases found]\n", testSuiteFile)
		}

		return nil, nil
	}

	// Now that we know we have tests to run, find their dependencies (it works on any file, valid or not)
	// and check for empty names, duplicate IDs, wrong needs and cycles
	g := graph.Build(&testSuiteSpec)

	if err := checkTestSuiteFile(&testSuiteSpec, g); err != nil {
		return nil, ReportTestSuiteError(testSuiteFile, err, failureInvalid)
	}

	return &TestSuite{Spec: &testSuiteSpec, Graph: g}, nil
}
