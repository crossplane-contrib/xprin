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
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// newRenderRunner returns a runner for the CLI of xprin test on v2, that modify can change (if it is not nil), and the
// environment of a test case on an in-memory filesystem. The arguments of the render command are set in renderArgs.
func newRenderRunner(t *testing.T, modify func(r *Runner)) (r *Runner, env *testCaseEnv, renderArgs *[]string) {
	t.Helper()

	r = newMockRunner(&testexecutionOptions.Options{
		Dependencies: map[string]string{"crossplane": config.CrossplaneCmd},
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
	})
	r.fs = afero.NewMemMapFs()
	renderArgs = new([]string)
	r.runCommand = func(_, _ string, args ...string) ([]byte, error) {
		*renderArgs = args

		return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), nil
	}

	if modify != nil {
		modify(r)
	}

	env = newTestEnv()
	env.inputsDir, env.outputsDir = "/inputs", "/outputs"
	require.NoError(t, r.fs.MkdirAll(env.outputsDir, 0o755))

	return r, env, renderArgs
}

// renderWith runs renderTestCase on the test case, with a runner from newRenderRunner, and returns what renderTestCase
// returned and the arguments of the render command.
func renderWith(t *testing.T, testCase api.TestCase, modify func(r *Runner)) (failed *engine.TestCaseResult, renderArgs []string) {
	t.Helper()

	r, env, args := newRenderRunner(t, modify)
	failed = r.renderTestCase(env, testCase, engine.NewTestCaseResult(testCase.Name, "", false, false, false, false, false))

	return failed, *args
}

func TestSaveOutputs(t *testing.T) {
	newRunner := func() (*Runner, *[]string) {
		r := NewRunner(&testexecutionOptions.Options{})

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

		require.NoError(t, r.saveOutputs(api.TestCase{ID: "a"}, &testCaseEnv{TestSuiteContext: TestSuiteContext{SavedOutputsDir: "/saved-outputs"}, outputsDir: "/tmp/outputs"}, result))

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

		require.NoError(t, r.saveOutputs(api.TestCase{}, &testCaseEnv{TestSuiteContext: TestSuiteContext{SavedOutputsDir: "/saved-outputs"}, outputsDir: "/tmp/outputs"}, result))

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

	r.runCommand = func(string, string, ...string) ([]byte, error) {
		data, err := afero.ReadFile(fs, functionsPath)
		require.NoError(t, err)

		function := &unstructured.Unstructured{}
		require.NoError(t, yaml.Unmarshal(data, &function.Object))

		annotations = function.GetAnnotations()

		return nil, errors.New("stop here")
	}

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: functionsPath}}
	r.renderTestCase(newTestEnv(), testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

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
	r.runCommand = func(string, string, ...string) ([]byte, error) { panic("boom") }

	testCase := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: functionsPath}}

	assert.PanicsWithValue(t, "boom", func() {
		r.renderTestCase(newTestEnv(), testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))
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

				result := r.expandInputPaths(newTestEnv(), &testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

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

	result := r.expandInputPaths(newTestEnv(), &testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

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

	result := r.expandInputPaths(newTestEnv(), &testCase, engine.NewTestCaseResult("test", "", false, false, false, false, false))

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

func TestRenderTestCase_BuildsTheRenderCommand(t *testing.T) {
	xr := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}
	base := []string{config.RenderSubcommand, config.RenderFlags}

	allInputs := xr
	allInputs.Inputs.ContextFiles = map[string]string{"ctx": "context_file.yaml"}
	allInputs.Inputs.ContextValues = map[string]string{"ctx": `{"key":"value"}`}
	allInputs.Inputs.ObservedResources = "observed.yaml"
	allInputs.Inputs.ExtraResources = "extra.yaml"
	allInputs.Inputs.FunctionCredentials = "func_creds.yaml"

	withXRD := xr
	withXRD.Patches.XRD = "my-xrd.yaml"

	for name, tt := range map[string]struct {
		testCase api.TestCase
		modify   func(r *Runner)
		want     []string
	}{
		"the XR, the composition and the functions": {
			testCase: xr,
			want:     append(slices.Clone(base), "xr.yaml", "comp.yaml", "functions.yaml"),
		},
		"every optional input": {
			testCase: allInputs,
			want: append(slices.Clone(base), "xr.yaml", "comp.yaml", "functions.yaml",
				"--context-files", "ctx=context_file.yaml",
				"--context-values", `ctx={"key":"value"}`,
				"--observed-resources", "observed.yaml",
				"--extra-resources", "extra.yaml",
				"--function-credentials", "func_creds.yaml"),
		},
		"the crossplane version, on a CLI that supports it": {
			testCase: xr,
			modify:   func(r *Runner) { r.CrossplaneVersion = "v1.2.3" },
			want:     append(slices.Clone(base), "--crossplane-version=v1.2.3", "xr.yaml", "comp.yaml", "functions.yaml"),
		},
		"no crossplane version, on a CLI that does not support it": {
			testCase: xr,
			modify: func(r *Runner) {
				r.CrossplaneVersion = "v1.2.3"
				r.XPCLI = xpcli.InitXPCLI(xpcli.TierV2Legacy)
			},
			want: append(slices.Clone(base), "xr.yaml", "comp.yaml", "functions.yaml"),
		},
		"the XRD, on a CLI that applies it": {
			testCase: withXRD,
			want:     append(slices.Clone(base), "xr.yaml", "comp.yaml", "functions.yaml", "--xrd=my-xrd.yaml"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			failed, args := renderWith(t, tt.testCase, tt.modify)

			assert.Nil(t, failed, "the render should go on")
			assert.Equal(t, tt.want, args)
		})
	}
}

// With container reuse the render is told to use the shared network of the run, also without a crossplane version,
// because the dockerized render engine would otherwise create a network for each call.
func TestRenderTestCase_PinsTheDockerNetwork(t *testing.T) {
	xr := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}
	hasNetwork := func(args []string) bool {
		return slices.ContainsFunc(args, func(arg string) bool { return strings.HasPrefix(arg, "--crossplane-docker-network=") })
	}

	for name, tt := range map[string]struct {
		modify func(r *Runner)
		want   bool
	}{
		"container reuse, with a crossplane version": {
			modify: func(r *Runner) {
				r.CrossplaneVersion = "v1.2.3"
				r.Containers = containers.NewCoordinator(&unittestsUtils.MockDocker{})
			},
			want: true,
		},
		"container reuse, without a crossplane version": {
			modify: func(r *Runner) { r.Containers = containers.NewCoordinator(&unittestsUtils.MockDocker{}) },
			want:   true,
		},
		"no container reuse": {
			modify: func(r *Runner) { r.CrossplaneVersion = "v1.2.3" },
			want:   false,
		},
		"container reuse, on a CLI that does not support it": {
			modify: func(r *Runner) {
				r.Containers = containers.NewCoordinator(&unittestsUtils.MockDocker{})
				r.XPCLI = xpcli.InitXPCLI(xpcli.TierV2Legacy)
			},
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			failed, args := renderWith(t, xr, tt.modify)

			assert.Nil(t, failed)
			assert.Equal(t, tt.want, hasNetwork(args))
		})
	}
}

func TestRenderTestCase_PatchesTheXR(t *testing.T) {
	const patchedXR = "/inputs/patched-xr.yaml"

	for name, tt := range map[string]struct {
		patches     api.Patches
		xpcli       xpcli.Tier
		wantPatched *api.Patches // what patchXR is called with, nil if it is not called
		wantXR      string       // the XR that render gets
		wantXRD     bool         // whether render gets --xrd
	}{
		"a connection secret": {
			patches:     api.Patches{ConnectionSecret: new(true)},
			xpcli:       xpcli.TierV2,
			wantPatched: &api.Patches{ConnectionSecret: new(true)},
			wantXR:      patchedXR,
		},
		"an XRD, on v1, where render has no --xrd": {
			patches:     api.Patches{XRD: "my-xrd.yaml"},
			xpcli:       xpcli.TierV1,
			wantPatched: &api.Patches{XRD: "my-xrd.yaml"},
			wantXR:      patchedXR,
		},
		"only an XRD, on v2, where render applies it": {
			patches: api.Patches{XRD: "my-xrd.yaml"},
			xpcli:   xpcli.TierV2,
			wantXR:  "xr.yaml",
			wantXRD: true,
		},
		"an XRD and a connection secret, on v2: patchXR does not get the XRD": {
			patches:     api.Patches{XRD: "my-xrd.yaml", ConnectionSecret: new(true), ConnectionSecretName: "my-secret", ConnectionSecretNamespace: "my-namespace"},
			xpcli:       xpcli.TierV2,
			wantPatched: &api.Patches{ConnectionSecret: new(true), ConnectionSecretName: "my-secret", ConnectionSecretNamespace: "my-namespace"},
			wantXR:      patchedXR,
			wantXRD:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var gotPatches *api.Patches

			testCase := api.TestCase{Name: "test", Patches: tt.patches, Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}

			failed, args := renderWith(t, testCase, func(r *Runner) {
				r.XPCLI = xpcli.InitXPCLI(tt.xpcli)
				r.patchXRFunc = func(_ *Runner, _, outputPath string, patches api.Patches) (string, error) {
					gotPatches = &patches

					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
			})

			assert.Nil(t, failed)
			assert.Equal(t, tt.wantPatched, gotPatches)
			assert.Equal(t, tt.wantXR, args[2], "the XR that render gets")
			assert.Equal(t, tt.wantXRD, slices.Contains(args, "--xrd=my-xrd.yaml"))
		})
	}
}

func TestRenderTestCase_ConvertsTheClaim(t *testing.T) {
	claim := api.TestCase{Name: "test", Inputs: api.Inputs{Claim: "claim.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}

	failed, args := renderWith(t, claim, func(r *Runner) {
		r.convertClaimToXRFunc = func(_ *Runner, claimPath, _, _, outputPath string) (string, error) {
			assert.Equal(t, "claim.yaml", claimPath)

			return filepath.Join(outputPath, "converted-xr.yaml"), nil
		}
	})

	assert.Nil(t, failed)
	assert.Equal(t, "/inputs/converted-xr.yaml", args[2], "render gets the converted XR")
}

func TestRenderTestCase_Fails(t *testing.T) {
	xr := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}
	claim := api.TestCase{Name: "test", Inputs: api.Inputs{Claim: "claim.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}

	invalidPatches := xr
	invalidPatches.Patches = api.Patches{ConnectionSecretName: "my-secret", ConnectionSecretNamespace: "my-namespace"}

	withPatches := xr
	withPatches.Patches = api.Patches{ConnectionSecret: new(true)}

	for name, tt := range map[string]struct {
		testCase  api.TestCase
		modify    func(r *Runner)
		wantError string
	}{
		"the claim can't be converted": {
			testCase: claim,
			modify: func(r *Runner) {
				r.convertClaimToXRFunc = func(*Runner, string, string, string, string) (string, error) { return "", errors.New("boom") }
			},
			wantError: "failed to convert Claim: boom",
		},
		"the XR can't be patched": {
			testCase: withPatches,
			modify: func(r *Runner) {
				r.patchXRFunc = func(*Runner, string, string, api.Patches) (string, error) { return "", errors.New("boom") }
			},
			wantError: "failed to patch XR: boom",
		},
		"the connection secret name has no connection secret": {
			testCase:  invalidPatches,
			wantError: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
	} {
		t.Run(name, func(t *testing.T) {
			failed, args := renderWith(t, tt.testCase, tt.modify)

			require.NotNil(t, failed)
			assert.Equal(t, engine.StatusFail(), failed.Status)
			require.Error(t, failed.Error)
			assert.Contains(t, failed.Error.Error(), tt.wantError)
			assert.Nil(t, args, "render should not run")
		})
	}
}

// The rendered resources are written to the outputs directory of the test case, one file for each and one for the XR,
// which is the first resource, and the result has the paths.
func TestRenderTestCase_WritesTheOutputs(t *testing.T) {
	const rendered = `apiVersion: example.org/v1alpha1
kind: Pod
metadata:
  name: test-pod
---
apiVersion: example.org/v1alpha1
kind: ConfigMap
metadata:
  name: test-configmap`

	r, env, _ := newRenderRunner(t, func(r *Runner) {
		r.runCommand = func(string, string, ...string) ([]byte, error) { return []byte(rendered), nil }
	})
	result := engine.NewTestCaseResult("test", "", false, false, false, false, false)

	failed := r.renderTestCase(env, api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}, result)

	require.Nil(t, failed)
	assert.Equal(t, "/outputs/rendered.yaml", result.Outputs.Render)
	assert.Equal(t, "/outputs/xr.yaml", result.Outputs.XR)
	assert.Equal(t, 2, result.Outputs.RenderCount)
	assert.Equal(t, map[string]string{
		"Pod/test-pod":             "/outputs/rendered-pod-test-pod.yaml",
		"ConfigMap/test-configmap": "/outputs/rendered-configmap-test-configmap.yaml",
	}, result.Outputs.Rendered)

	for _, path := range []string{result.Outputs.Render, result.Outputs.XR, result.Outputs.Rendered["Pod/test-pod"], result.Outputs.Rendered["ConfigMap/test-configmap"]} {
		exists, err := afero.Exists(r.fs, path)
		require.NoError(t, err)
		assert.True(t, exists, "%s should have been written", path)
	}

	xr, err := afero.ReadFile(r.fs, result.Outputs.XR)
	require.NoError(t, err)
	assert.Contains(t, string(xr), "test-pod", "the XR is the first rendered resource")
}

func TestRenderTestCase_FailsWithTheOutputWhenTheRenderFails(t *testing.T) {
	xr := api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "comp.yaml", Functions: "functions.yaml"}}

	failed, _ := renderWith(t, xr, func(r *Runner) {
		r.runCommand = func(string, string, ...string) ([]byte, error) {
			return []byte("render fail"), errors.New("exit status 1")
		}
	})

	require.NotNil(t, failed)
	assert.Equal(t, engine.StatusFail(), failed.Status)
	require.NoError(t, failed.Error, "a failed render is shown in its section")
	assert.Contains(t, failed.FormattedRenderOutput, "render fail")
}

func TestValidateTestCase_PassesTheCrossplaneVersionAsTheImage(t *testing.T) {
	for name, tt := range map[string]struct {
		version string
		crds    []string
		want    []string // nil when validate does not run
	}{
		"with a crossplane version": {
			version: "v1.2.3",
			crds:    []string{"crd.yaml"},
			want:    []string{config.ValidateSubcommand, "--crossplane-image=xpkg.crossplane.io/crossplane/crossplane:v1.2.3", "/inputs/crds", "/outputs/rendered.yaml"},
		},
		"without a crossplane version": {
			crds: []string{"crd.yaml"},
			want: []string{config.ValidateSubcommand, "/inputs/crds", "/outputs/rendered.yaml"},
		},
		"without CRDs there is nothing to validate against": {
			version: "v1.2.3",
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newMockRunner(&testexecutionOptions.Options{
				Dependencies:      map[string]string{"crossplane": config.CrossplaneCmd},
				Validate:          []string{config.ValidateSubcommand},
				CrossplaneVersion: tt.version,
			})
			r.fs = afero.NewMemMapFs()

			var gotArgs []string

			r.runCommand = func(_, _ string, args ...string) ([]byte, error) {
				gotArgs = args

				return []byte("validate ok"), nil
			}

			env := newTestEnv()
			env.inputsDir, env.outputsDir = "/inputs", "/outputs"
			require.NoError(t, r.fs.MkdirAll(env.outputsDir, 0o755))

			result := engine.NewTestCaseResult("test", "", false, false, false, false, false)
			result.Outputs.Render = "/outputs/rendered.yaml"

			failed := r.validateTestCase(env, api.TestCase{Name: "test", Inputs: api.Inputs{CRDs: tt.crds}}, result)

			assert.Nil(t, failed)
			assert.Equal(t, tt.want, gotArgs)

			if tt.want == nil {
				assert.Nil(t, result.Outputs.Validate, "there is no validation output")
			} else {
				require.NotNil(t, result.Outputs.Validate)
				assert.Equal(t, "/outputs/validate.txt", *result.Outputs.Validate)
			}
		})
	}
}
