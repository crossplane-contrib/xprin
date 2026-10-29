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
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

const annotationsDocs = "https://github.com/crossplane-contrib/xprin/blob/release-0.6/docs/how-it-works.md#migrating-from-user-set-container-annotations"

// UserSetAnnotations is a Function that sets annotations of its container (the name, the cleanup and the network),
// which xprin will not keep from xprin 0.7.0. It goes away with the support of them.
type UserSetAnnotations struct {
	File        string
	Function    string
	Annotations []string
}

// annotationPrefix is what the three annotations have in common.
const annotationPrefix = "render.crossplane.io/"

// Notice is the deprecation notice for the Functions of a testsuite file that set annotations of their container,
// which is followed by a line for each of them (see Line).
func Notice(testSuiteFile string) string {
	return fmt.Sprintf("%s: Functions found that set annotations of their container (%sruntime-docker-name, -cleanup or -network). This is deprecated: from 0.7.0 xprin ignores them and manages the containers itself while container reuse is on. Use --no-container-reuse to keep yours. See %s",
		testSuiteFile, annotationPrefix, annotationsDocs)
}

// Line is the line of the Function under the notice: where it is and which annotations it sets.
func (u UserSetAnnotations) Line() string {
	names := make([]string, 0, len(u.Annotations))
	for _, annotation := range u.Annotations {
		names = append(names, strings.TrimPrefix(annotation, annotationPrefix))
	}

	return fmt.Sprintf("  - %s: %s: %s", u.File, u.Function, strings.Join(names, ", "))
}

// FindUserSetAnnotations returns the Functions in functionsPath (a file or a directory of files) that set annotations of
// their container. A path that doesn't exist, or can't be read, has none: running the test case fails with the reason.
func FindUserSetAnnotations(fs afero.Fs, functionsPath string) []UserSetAnnotations {
	info, err := fs.Stat(functionsPath)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		return findInFile(fs, functionsPath)
	}

	entries, err := afero.ReadDir(fs, functionsPath)
	if err != nil {
		return nil
	}

	var found []UserSetAnnotations

	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); !e.IsDir() && (ext == ".yaml" || ext == ".yml") {
			found = append(found, findInFile(fs, filepath.Join(functionsPath, e.Name()))...)
		}
	}

	return found
}

func findInFile(fs afero.Fs, path string) []UserSetAnnotations {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil
	}

	var found []UserSetAnnotations

	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			// The end of the file, or a file that can't be decoded, which fails when the test case runs
			return found
		}

		if obj.Object == nil || obj.GetKind() != "Function" {
			continue
		}

		var set []string

		for _, annotation := range []string{annotationDockerName, annotationDockerCleanup, annotationDockerNetwork} {
			if _, exists := obj.GetAnnotations()[annotation]; exists {
				set = append(set, annotation)
			}
		}

		if len(set) > 0 {
			found = append(found, UserSetAnnotations{File: path, Function: obj.GetName(), Annotations: set})
		}
	}
}
