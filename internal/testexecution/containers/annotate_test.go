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
	"fmt"
	"path/filepath"
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

const (
	imagePatchAndTransformV8 = "xpkg.crossplane.io/crossplane-contrib/function-patch-and-transform:v0.8.0"
	imageAutoReady           = "xpkg.crossplane.io/crossplane-contrib/function-auto-ready:v0.5.0"
	functionsFilePath        = "/suite/functions.yaml"
)

func writeFunctionsFile(t *testing.T, fs afero.Fs, path, content string) {
	t.Helper()
	require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o600))
}

func readAnnotations(t *testing.T, fs afero.Fs, path string) map[string]string {
	t.Helper()

	data, err := afero.ReadFile(fs, path)
	require.NoError(t, err)

	var obj map[string]any
	require.NoError(t, yaml.Unmarshal(data, &obj))

	meta, ok := obj["metadata"].(map[string]any)
	require.True(t, ok, "metadata missing")

	ann, ok := meta["annotations"].(map[string]any)
	if !ok {
		return nil
	}

	result := make(map[string]string, len(ann))
	for k, v := range ann {
		result[k] = fmt.Sprint(v)
	}

	return result
}

// namesOf returns the names of coord.containers as a slice, for comparison against an expected []string in tests.
func namesOf(coord *Coordinator) []string {
	names := make([]string, 0, len(coord.containers))
	for _, ctr := range coord.containers {
		names = append(names, ctr.name)
	}

	return names
}

func TestAnnotateFunctions_NilCoordinator_NoOp(t *testing.T) {
	fs := afero.NewMemMapFs()

	_, err := (*Coordinator)(nil).annotateFunctions(fs, functionsFilePath)
	require.NoError(t, err)
}

func TestAnnotateFunctions_PathNotExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	_, err := coord.annotateFunctions(fs, "/nonexistent/functions.yaml")
	require.NoError(t, err)
	assert.Empty(t, coord.containers)
}

func TestAnnotateFunctions_SingleFunction(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
`)

	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)

	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, wantName, ann[annotationDockerName])
	assert.Equal(t, dockerCleanupOrphan, ann[annotationDockerCleanup])
}

func TestAnnotateFunctions_SameImageKeyedIdentically(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: one-name
spec:
  package: `+imagePatchAndTransformV8+`
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: a-different-name-same-image
spec:
  package: `+imagePatchAndTransformV8+`
`)

	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)

	// The same image is tracked idempotently, even though two Function resources reference it.
	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))
}

func TestAnnotateFunctions_DifferentImagesDifferentNames(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
spec:
  package: `+imageAutoReady+`
`)

	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		coord.ContainerName(imagePatchAndTransformV8),
		coord.ContainerName(imageAutoReady),
	}, namesOf(coord))
}

func TestAnnotateFunctions_UserSetAnnotations_AreReplaced(t *testing.T) {
	tests := map[string]string{
		"a name":                 `render.crossplane.io/runtime-docker-name: my-custom-container`,
		"an empty name":          `render.crossplane.io/runtime-docker-name: ""`,
		"a cleanup":              `render.crossplane.io/runtime-docker-cleanup: Stop`,
		"a cleanup that removes": `render.crossplane.io/runtime-docker-cleanup: Remove`,
		"a network":              `render.crossplane.io/runtime-docker-network: my-network`,
	}

	for name, annotation := range tests {
		t.Run(name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			coord := NewCoordinator(&unittestsUtils.MockDocker{})

			writeFunctionsFile(t, fs, functionsFilePath, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
  annotations:
    `+annotation+`
    render.crossplane.io/runtime-docker-pull-policy: Always
spec:
  package: `+imagePatchAndTransformV8+`
`)

			returned, err := coord.annotateFunctions(fs, functionsFilePath)
			require.NoError(t, err)

			wantName := coord.ContainerName(imagePatchAndTransformV8)
			assert.Equal(t, []string{wantName}, returned, "the container that Acquire waits for is the one that xprin names")
			assert.Equal(t, []string{wantName}, namesOf(coord), "the container is always one that xprin removes at the end")

			ann := readAnnotations(t, fs, functionsFilePath)
			assert.Equal(t, wantName, ann[annotationDockerName])
			assert.Equal(t, dockerCleanupOrphan, ann[annotationDockerCleanup])
			assert.NotContains(t, ann, annotationDockerNetwork, "the network is the shared one, that render is told to use")
			assert.Equal(t, "Always", ann["render.crossplane.io/runtime-docker-pull-policy"], "the other annotations are the user's")
		})
	}
}

func TestAnnotateFunctions_MissingImageRef_Untouched(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-with-no-package
spec: {}
`)

	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)
	assert.Empty(t, coord.containers)

	ann := readAnnotations(t, fs, path)
	assert.Nil(t, ann, "no annotations should be added when there is no image reference to key on")
}

func TestAnnotateFunctions_NonFunctionKind_Untouched(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
`)

	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)
	assert.Empty(t, coord.containers)

	ann := readAnnotations(t, fs, path)
	assert.Nil(t, ann)
}

func TestAnnotateFunctions_Directory_SameImageAcrossFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	dir := "/suite/functions"
	writeFunctionsFile(t, fs, filepath.Join(dir, "fn1.yaml"), `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
`)
	writeFunctionsFile(t, fs, filepath.Join(dir, "fn2.yaml"), `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: same-image-different-file
spec:
  package: `+imagePatchAndTransformV8+`
`)
	writeFunctionsFile(t, fs, filepath.Join(dir, "readme.txt"), `not a yaml file`)

	_, err := coord.annotateFunctions(fs, dir)
	require.NoError(t, err)

	// The same image appears in both files; claiming is idempotent, so it's still tracked once.
	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))
}

func TestAnnotateFunctions_SecondTestCaseReusesTrackedContainer(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
`)

	// First test case.
	_, err := coord.annotateFunctions(fs, path)
	require.NoError(t, err)

	wantName := coord.ContainerName(imagePatchAndTransformV8)
	require.Equal(t, []string{wantName}, namesOf(coord))

	// Second test case, same functions file content (re-write since the first call rewrote it
	// with annotations, but the image reference is unchanged): must resolve to the same name.
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
`)

	_, err = coord.annotateFunctions(fs, path)
	require.NoError(t, err)
	assert.Equal(t, []string{wantName}, namesOf(coord), "still just the one name — reuse, not a second container")

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, wantName, ann[annotationDockerName], "the second test case must reuse the exact same container name")
}

func TestAnnotateFunctions_ReturnsTheNamesOfTheContainersOfTheFunctions(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	dir := "/suite/functions"
	writeFunctionsFile(t, fs, filepath.Join(dir, "a.yaml"), `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: `+imagePatchAndTransformV8+`
---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
spec:
  package: `+imageAutoReady+`
`)
	writeFunctionsFile(t, fs, filepath.Join(dir, "b.yaml"), `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform-again
spec:
  package: `+imagePatchAndTransformV8+`
`)

	names, err := coord.annotateFunctions(fs, dir)
	require.NoError(t, err)

	want := []string{coord.ContainerName(imagePatchAndTransformV8), coord.ContainerName(imageAutoReady), coord.ContainerName(imagePatchAndTransformV8)}

	assert.Equal(t, want, names, "in the order of the files and of the Functions, a name for each Function that uses it")
}

func TestAnnotateFunctions_NoContainersToReturn(t *testing.T) {
	fs := afero.NewMemMapFs()

	names, err := (*Coordinator)(nil).annotateFunctions(fs, functionsFilePath)
	require.NoError(t, err)
	assert.Empty(t, names, "nil coordinator: reuse is not enabled")

	writeFunctionsFile(t, fs, functionsFilePath, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: no-image
spec: {}
`)

	names, err = NewCoordinator(&unittestsUtils.MockDocker{}).annotateFunctions(fs, functionsFilePath)
	require.NoError(t, err)
	assert.Empty(t, names, "a Function without an image reference has no container to key")
}
