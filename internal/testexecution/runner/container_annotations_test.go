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

package runner

import (
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const annotatedFunctions = `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
  annotations:
    render.crossplane.io/runtime-docker-name: my-container
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-auto-ready:v0.5.0
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
  annotations:
    render.crossplane.io/runtime-docker-network: my-network
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.0
`

// warningsOf runs the notice of a testsuite file with the test cases, and returns what it prints.
func warningsOf(t *testing.T, files map[string]string, spec *api.TestSuiteSpec, mocks ...func(*testexecutionOptions.Options)) string {
	t.Helper()

	fs := afero.NewMemMapFs()
	for path, content := range files {
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o600))
	}

	options := &testexecutionOptions.Options{
		Containers:   containers.NewCoordinator(&unittestsUtils.MockDocker{}),
		Repositories: map[string]string{"shared": "/shared"},
	}
	for _, mock := range mocks {
		mock(options)
	}

	r := newMockRunner(options, func(r *Runner) {
		r.fs = fs
		r.testSuite = testsuite.New(testSuiteFile, spec)
	})

	return unittestsUtils.CaptureStderr(r.warnUserSetContainerAnnotations)
}

func TestWarnUserSetContainerAnnotations_OnceForEachFile(t *testing.T) {
	spec := &api.TestSuiteSpec{
		Common: api.Common{Inputs: api.Inputs{Functions: "/functions.yaml"}},
		Tests: []api.TestCase{
			{Name: "uses common"},
			{Name: "uses common too"},
			{Name: "names the same file", Inputs: api.Inputs{Functions: "/functions.yaml"}},
			{Name: "uses another file", Inputs: api.Inputs{Functions: "/other.yaml"}},
		},
	}

	out := warningsOf(t, map[string]string{"/functions.yaml": annotatedFunctions, "/other.yaml": annotatedFunctions}, spec)

	assert.Equal(t, 1, strings.Count(out, "Functions found that set annotations"), "one notice for the testsuite file")
	assert.Contains(t, out, testSuiteFile+": Functions found that set annotations")
	assert.Equal(t, 5, strings.Count(out, "WARNING:"), "the notice and the two Functions of each of the two files, not of each test case")
	assert.Equal(t, 2, strings.Count(out, "  - /functions.yaml: "))
	assert.Equal(t, 2, strings.Count(out, "  - /other.yaml: "))
	assert.Contains(t, out, "/functions.yaml: function-auto-ready: runtime-docker-name\n")
	assert.Contains(t, out, "/functions.yaml: function-patch-and-transform: runtime-docker-network\n")
}

func TestWarnUserSetContainerAnnotations_Directory(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "a", Inputs: api.Inputs{Functions: "/functions"}}}}

	out := warningsOf(t, map[string]string{"/functions/a.yaml": annotatedFunctions}, spec)

	assert.Equal(t, 2, strings.Count(out, "  - /functions/a.yaml: "))
}

func TestWarnUserSetContainerAnnotations_TemplatedPath(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "repository", Inputs: api.Inputs{Functions: placeholder.Create(".Repositories.shared") + "/functions.yaml"}},
		{Name: "output of another test case", Inputs: api.Inputs{Functions: placeholder.Create(".Tests.first.Outputs.XR") + "/functions.yaml"}},
		{Name: "unknown repository", Inputs: api.Inputs{Functions: placeholder.Create(".Repositories.nope") + "/functions.yaml"}},
	}}

	out := warningsOf(t, map[string]string{"/shared/functions.yaml": annotatedFunctions}, spec)

	assert.Equal(t, 2, strings.Count(out, "  - /shared/functions.yaml: "), "the repository is rendered, the other two are skipped")
}

func TestWarnUserSetContainerAnnotations_NothingToTell(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{
		{Name: "a", Inputs: api.Inputs{Functions: "/functions.yaml"}},
		{Name: "no functions"},
		{Name: "missing file", Inputs: api.Inputs{Functions: "/missing.yaml"}},
	}}

	t.Run("container reuse is off", func(t *testing.T) {
		r := newMockRunner(&testexecutionOptions.Options{}, func(r *Runner) {
			r.fs = afero.NewMemMapFs()
			require.NoError(t, afero.WriteFile(r.fs, "/functions.yaml", []byte(annotatedFunctions), 0o600))
			r.testSuite = testsuite.New(testSuiteFile, spec)
		})

		assert.Empty(t, unittestsUtils.CaptureStderr(r.warnUserSetContainerAnnotations))
	})

	t.Run("no annotations", func(t *testing.T) {
		out := warningsOf(t, map[string]string{"/functions.yaml": "apiVersion: pkg.crossplane.io/v1\nkind: Function\nmetadata:\n  name: fn\nspec:\n  package: x/y:v1\n"}, spec)

		assert.Empty(t, out)
	})
}
