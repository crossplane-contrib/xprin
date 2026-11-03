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
	"errors"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func TestSaveOutputs(t *testing.T) {
	newRunner := func() (*Runner, *[]string) {
		r := NewRunner(&testexecutionOptions.Options{}, testsuite.New("/suite_xprin.yaml", &api.TestSuiteSpec{}))
		r.env = &testSuiteEnv{savedOutputsDir: "/saved-outputs"}

		var copied []string

		r.copy = func(src, dest string, _ ...cp.Options) error {
			copied = append(copied, src+" -> "+dest)

			return nil
		}

		return r, &copied
	}

	t.Run("a test case with an ID: the outputs are saved, and point to the internal artifacts", func(t *testing.T) {
		r, copied := newRunner()
		validate, assertions := "/tmp/outputs/validate.txt", "/tmp/outputs/assertions.txt"
		result := engine.NewTestCaseResult("a", "a", false, false, false, false, false)
		result.Outputs = engine.Outputs{
			Render:     "/tmp/outputs/rendered.yaml",
			XR:         "/tmp/outputs/xr.yaml",
			Validate:   &validate,
			Assertions: &assertions,
			Rendered:   map[string]string{"Kind/name": "/tmp/outputs/rendered/Kind-name.yaml"},
		}

		require.NoError(t, r.saveOutputs(api.TestCase{ID: "a"}, &testCaseEnv{outputsDir: "/tmp/outputs"}, result))

		assert.Equal(t, []string{"/tmp/outputs -> /saved-outputs/a"}, *copied)
		assert.Equal(t, "/saved-outputs/a/rendered.yaml", result.Outputs.Render)
		assert.Equal(t, "/saved-outputs/a/xr.yaml", result.Outputs.XR)
		assert.Equal(t, "/saved-outputs/a/validate.txt", *result.Outputs.Validate)
		assert.Equal(t, "/saved-outputs/a/assertions.txt", *result.Outputs.Assertions)
		assert.Equal(t, "/saved-outputs/a/Kind-name.yaml", result.Outputs.Rendered["Kind/name"])
	})

	t.Run("a test case without an ID: nothing is copied", func(t *testing.T) {
		r, copied := newRunner()
		result := engine.NewTestCaseResult("a", "", false, false, false, false, false)
		result.Outputs.XR = "/tmp/outputs/xr.yaml"

		require.NoError(t, r.saveOutputs(api.TestCase{}, &testCaseEnv{outputsDir: "/tmp/outputs"}, result))

		assert.Empty(t, *copied)
		assert.Equal(t, "/tmp/outputs/xr.yaml", result.Outputs.XR)
	})
}

func TestRenderTestCase_AppliesTheContainerReuseToTheFunctionsRightBeforeRendering(t *testing.T) {
	const functionsPath = "/inputs/functions.yaml"

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, functionsPath, []byte(`apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
  annotations:
    render.crossplane.io/runtime-docker-name: set-by-a-hook
    render.crossplane.io/runtime-docker-network: my-network
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-auto-ready:v0.5.0
`), 0o600))

	coord := containers.NewCoordinator(&unittestsUtils.MockDocker{})
	r := newMockRunner(&testexecutionOptions.Options{Containers: coord}, func(r *Runner) { r.fs = fs })

	var annotations map[string]string

	r.runCommand = func(string, ...string) ([]byte, error) {
		data, err := afero.ReadFile(fs, functionsPath)
		require.NoError(t, err)

		function := &unstructured.Unstructured{}
		require.NoError(t, yaml.Unmarshal(data, &function.Object))

		annotations = function.GetAnnotations()

		return nil, errors.New("stop here")
	}

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: functionsPath}}
	r.renderTestCase(&testCaseEnv{}, testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

	assert.Equal(t, coord.ContainerName("xpkg.crossplane.io/crossplane-contrib/function-auto-ready:v0.5.0"), annotations["render.crossplane.io/runtime-docker-name"])
	assert.Equal(t, "Orphan", annotations["render.crossplane.io/runtime-docker-cleanup"])
	assert.NotContains(t, annotations, "render.crossplane.io/runtime-docker-network")
}

func TestRenderTestCase_APanicOfTheRenderDoesNotLeaveTheContainersLocked(t *testing.T) {
	const functionsPath = "/inputs/functions.yaml"

	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, functionsPath, []byte(`apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
spec:
  package: xpkg.crossplane.io/crossplane-contrib/function-auto-ready:v0.5.0
`), 0o600))

	coord := containers.NewCoordinator(&unittestsUtils.MockDocker{})
	r := newMockRunner(&testexecutionOptions.Options{Containers: coord}, func(r *Runner) { r.fs = fs })
	r.runCommand = func(string, ...string) ([]byte, error) { panic("boom") }

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: functionsPath}}

	assert.PanicsWithValue(t, "boom", func() {
		r.renderTestCase(&testCaseEnv{}, testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))
	})

	// The test cases after it that use the container don't wait for a render that is not running any more
	acquired := make(chan struct{})

	go func() {
		release, err := coord.AnnotateAndAcquireFunctions(fs, functionsPath)
		assert.NoError(t, err)
		release(true)
		close(acquired)
	}()

	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the container is still locked after the render panicked")
	}
}

// Every input of a test case is expanded, and then verified. When one can't be, the test case fails and the error says
// which input, and whether it could not be expanded or does not exist.
func TestExpandInputPaths_ReportsTheInputThatFails(t *testing.T) {
	const bad = "bad.yaml"

	inputs := []struct {
		name      string
		set       func(testCase *api.TestCase)
		expandErr string
		verifyErr string
	}{
		{"XR", func(tc *api.TestCase) { tc.Inputs.XR = bad }, "failed to expand XR path", "XR file not found"},
		{"Claim", func(tc *api.TestCase) { tc.Inputs.XR, tc.Inputs.Claim = "", bad }, "failed to expand Claim path", "Claim file not found"},
		{"composition", func(tc *api.TestCase) { tc.Inputs.Composition = bad }, "failed to expand composition path", "composition file not found"},
		{"functions", func(tc *api.TestCase) { tc.Inputs.Functions = bad }, "failed to expand functions path", "functions file or dir not found"},
		{"CRD", func(tc *api.TestCase) { tc.Inputs.CRDs = []string{bad} }, "failed to expand CRD path " + bad, "crd file not found"},
		{"context file", func(tc *api.TestCase) { tc.Inputs.ContextFiles = map[string]string{"foo": bad} }, "failed to expand context file path for key 'foo'", "context file not found for key 'foo'"},
		{"observed resources", func(tc *api.TestCase) { tc.Inputs.ObservedResources = bad }, "failed to expand observed resources path", "observed resources file or dir not found"},
		{"extra resources", func(tc *api.TestCase) { tc.Inputs.ExtraResources = bad }, "failed to expand extra resources path", "extra resources file or dir not found"},
		{"function credentials", func(tc *api.TestCase) { tc.Inputs.FunctionCredentials = bad }, "failed to expand function credentials path", "function credentials file or dir not found"},
		{"XRD patch", func(tc *api.TestCase) { tc.Patches.XRD = bad }, "failed to expand XRD path", "XRD file or dir not found"},
	}

	for _, in := range inputs {
		for _, fails := range []string{"expand", "verify"} {
			t.Run(in.name+" fails to "+fails, func(t *testing.T) {
				r := newMockRunner(&testexecutionOptions.Options{})
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if fails == "expand" && path == bad {
						return "", errors.New("expand fail")
					}

					return path, nil
				}
				r.verifyPathExists = func(path string) error {
					if fails == "verify" && path == bad {
						return errors.New("not found")
					}

					return nil
				}

				testCase := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "composition.yaml", Functions: "functions.yaml"}}
				in.set(&testCase)

				want := in.expandErr + ": expand fail"
				if fails == "verify" {
					want = in.verifyErr + ": not found"
				}

				result := r.expandInputPaths(&testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

				require.NotNil(t, result, "the test case should fail")
				assert.Equal(t, engine.StatusFail(), result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), want)
			})
		}
	}
}

func TestExpandInputPaths_ReportsEveryInputThatFails(t *testing.T) {
	r := newMockRunner(&testexecutionOptions.Options{})
	r.verifyPathExists = func(path string) error {
		if path == "missing_composition.yaml" || path == "missing_crd.yaml" {
			return errors.New("not found")
		}

		return nil
	}

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{
		XR:          "xr.yaml",
		Composition: "missing_composition.yaml",
		Functions:   "functions.yaml",
		CRDs:        []string{"crd.yaml", "missing_crd.yaml"},
	}}

	result := r.expandInputPaths(&testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

	require.NotNil(t, result)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "composition file not found: not found")
	assert.Contains(t, result.Error.Error(), "crd file not found: not found")
}

func TestExpandInputPaths_ExpandsRelativePathsAndKeepsTheAbsoluteOnes(t *testing.T) {
	r := newMockRunner(&testexecutionOptions.Options{}, func(r *Runner) {
		r.expandPathRelativeToTestSuiteFile = testsuite.ExpandPath
	})

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{
		XR:                  "xr.yaml",
		Composition:         "/abs/composition.yaml",
		Functions:           "functions",
		CRDs:                []string{"crds/a.yaml", "/abs/b.yaml"},
		ContextFiles:        map[string]string{"foo": "context.json"},
		ObservedResources:   "observed",
		ExtraResources:      "extra",
		FunctionCredentials: "credentials.yaml",
	}, Patches: api.Patches{XRD: "xrd.yaml"}}

	result := r.expandInputPaths(&testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

	assert.Nil(t, result, "nothing should fail")
	assert.Equal(t, "/xr.yaml", testCase.Inputs.XR)
	assert.Equal(t, "/abs/composition.yaml", testCase.Inputs.Composition)
	assert.Equal(t, "/functions", testCase.Inputs.Functions)
	assert.Equal(t, []string{"/crds/a.yaml", "/abs/b.yaml"}, testCase.Inputs.CRDs)
	assert.Equal(t, map[string]string{"foo": "/context.json"}, testCase.Inputs.ContextFiles)
	assert.Equal(t, "/observed", testCase.Inputs.ObservedResources)
	assert.Equal(t, "/extra", testCase.Inputs.ExtraResources)
	assert.Equal(t, "/credentials.yaml", testCase.Inputs.FunctionCredentials)
	assert.Equal(t, "/xrd.yaml", testCase.Patches.XRD)
}
