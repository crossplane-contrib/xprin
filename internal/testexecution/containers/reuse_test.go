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

// namesOf returns coord.names as a slice, for comparison against an expected []string in tests.
func namesOf(coord *Coordinator) []string {
	names := make([]string, 0, len(coord.names))
	for name := range coord.names {
		names = append(names, name)
	}

	return names
}

func TestApplyReuse_NilCoordinator_NoOp(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := ApplyReuse(fs, nil, functionsFilePath)
	require.NoError(t, err)
}

func TestApplyReuse_PathNotExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	err := ApplyReuse(fs, coord, "/nonexistent/functions.yaml")
	require.NoError(t, err)
	assert.Empty(t, coord.names)
}

func TestApplyReuse_SingleFunction(t *testing.T) {
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

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)

	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, wantName, ann[annotationDockerName])
	assert.Equal(t, dockerCleanupOrphan, ann[annotationDockerCleanup])
}

func TestApplyReuse_SameImageKeyedIdentically(t *testing.T) {
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

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)

	// The same image is tracked idempotently, even though two Function resources reference it.
	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))
}

func TestApplyReuse_DifferentImagesDifferentNames(t *testing.T) {
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

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		coord.ContainerName(imagePatchAndTransformV8),
		coord.ContainerName(imageAutoReady),
	}, namesOf(coord))
}

func TestApplyReuse_ExistingDockerName_NotOverridden(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
  annotations:
    render.crossplane.io/runtime-docker-name: my-custom-container
spec:
  package: `+imagePatchAndTransformV8+`
`)

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)
	assert.Empty(t, coord.names, "a user-supplied name must not be tracked by xprin")

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, "my-custom-container", ann[annotationDockerName])
	assert.Equal(t, dockerCleanupOrphan, ann[annotationDockerCleanup])
}

func TestApplyReuse_ExistingDockerCleanup_NotOverridden(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
  annotations:
    render.crossplane.io/runtime-docker-cleanup: Stop
spec:
  package: `+imagePatchAndTransformV8+`
`)

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)
	assert.Len(t, coord.names, 1, "the name is still ours to manage even though cleanup was overridden")

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, "Stop", ann[annotationDockerCleanup], "user-defined cleanup should not be overridden")
}

func TestApplyReuse_MissingImageRef_Untouched(t *testing.T) {
	fs := afero.NewMemMapFs()
	coord := NewCoordinator(&unittestsUtils.MockDocker{})

	path := functionsFilePath
	writeFunctionsFile(t, fs, path, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-with-no-package
spec: {}
`)

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)
	assert.Empty(t, coord.names)

	ann := readAnnotations(t, fs, path)
	assert.Nil(t, ann, "no annotations should be added when there is no image reference to key on")
}

func TestApplyReuse_NonFunctionKind_Untouched(t *testing.T) {
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

	err := ApplyReuse(fs, coord, path)
	require.NoError(t, err)
	assert.Empty(t, coord.names)

	ann := readAnnotations(t, fs, path)
	assert.Nil(t, ann)
}

func TestApplyReuse_Directory_SameImageAcrossFiles(t *testing.T) {
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

	err := ApplyReuse(fs, coord, dir)
	require.NoError(t, err)

	// The same image appears in both files; claiming is idempotent, so it's still tracked once.
	wantName := coord.ContainerName(imagePatchAndTransformV8)
	assert.Equal(t, []string{wantName}, namesOf(coord))
}

func TestApplyReuse_SecondTestCaseReusesTrackedContainer(t *testing.T) {
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
	err := ApplyReuse(fs, coord, path)
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

	err = ApplyReuse(fs, coord, path)
	require.NoError(t, err)
	assert.Equal(t, []string{wantName}, namesOf(coord), "still just the one name — reuse, not a second container")

	ann := readAnnotations(t, fs, path)
	assert.Equal(t, wantName, ann[annotationDockerName], "the second test case must reuse the exact same container name")
}
