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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const functionSettingAll = `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
  annotations:
    render.crossplane.io/runtime-docker-name: my-other-container
    render.crossplane.io/runtime-docker-cleanup: Stop
    render.crossplane.io/runtime-docker-network: my-network
spec:
  package: ` + imageAutoReady + `
`

func TestFindUserSetAnnotations_File(t *testing.T) {
	fs := afero.NewMemMapFs()

	writeFunctionsFile(t, fs, functionsFilePath, `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-patch-and-transform
  annotations:
    render.crossplane.io/runtime-docker-name: my-custom-container
spec:
  package: `+imagePatchAndTransformV8+`
---
`+functionSettingAll+`---
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-without-annotations
spec:
  package: `+imageAutoReady+`
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: not-a-function
  annotations:
    render.crossplane.io/runtime-docker-name: whatever
`)

	found := FindUserSetAnnotations(fs, functionsFilePath)

	require.Len(t, found, 2)
	assert.Equal(t, UserSetAnnotations{File: functionsFilePath, Function: "function-patch-and-transform", Annotations: []string{annotationDockerName}}, found[0])
	assert.Equal(t, UserSetAnnotations{File: functionsFilePath, Function: "function-auto-ready", Annotations: []string{annotationDockerName, annotationDockerCleanup, annotationDockerNetwork}}, found[1])
}

func TestFindUserSetAnnotations_Directory(t *testing.T) {
	fs := afero.NewMemMapFs()

	writeFunctionsFile(t, fs, "/functions/a.yaml", functionSettingAll)
	writeFunctionsFile(t, fs, "/functions/b.yml", functionSettingAll)
	writeFunctionsFile(t, fs, "/functions/notes.txt", functionSettingAll)
	writeFunctionsFile(t, fs, "/functions/sub/c.yaml", functionSettingAll)

	found := FindUserSetAnnotations(fs, "/functions")

	require.Len(t, found, 2, "only the YAML files of the directory itself")
	assert.Equal(t, "/functions/a.yaml", found[0].File)
	assert.Equal(t, "/functions/b.yml", found[1].File)
}

func TestFindUserSetAnnotations_NothingToFind(t *testing.T) {
	fs := afero.NewMemMapFs()

	writeFunctionsFile(t, fs, "/functions/not-yaml.yaml", "{{ not yaml")
	writeFunctionsFile(t, fs, "/functions/none.yaml", `apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-auto-ready
spec:
  package: `+imageAutoReady+`
`)

	assert.Empty(t, FindUserSetAnnotations(fs, "/functions"))
	assert.Empty(t, FindUserSetAnnotations(fs, "/does-not-exist"))
}

func TestUserSetAnnotations_Notice(t *testing.T) {
	assert.Equal(t, "/suite_xprin.yaml: Functions found that set annotations of their container (render.crossplane.io/runtime-docker-name, -cleanup or -network). This is deprecated: from 0.7.0 xprin ignores them and manages the containers itself while container reuse is on. Use --no-container-reuse to keep yours. See "+annotationsDocs, Notice("/suite_xprin.yaml"))

	one := UserSetAnnotations{File: "/f.yaml", Function: "fn", Annotations: []string{annotationDockerName}}
	assert.Equal(t, "  - /f.yaml: fn: runtime-docker-name", one.Line())

	three := UserSetAnnotations{File: "/f.yaml", Function: "fn", Annotations: []string{annotationDockerName, annotationDockerCleanup, annotationDockerNetwork}}
	assert.Equal(t, "  - /f.yaml: fn: runtime-docker-name, runtime-docker-cleanup, runtime-docker-network", three.Line())
}
