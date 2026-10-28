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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	annotationDockerName    = "render.crossplane.io/runtime-docker-name"
	annotationDockerCleanup = "render.crossplane.io/runtime-docker-cleanup"
	dockerCleanupOrphan     = "Orphan"
)

// ApplyReuse patches the Function resources in functionsPath (a file or a directory of files) with container-reuse
// annotations, keyed by the image reference of each function via coord.
// It returns the names of the containers they use, once each, for Coordinator.Acquire.
// It does nothing, and returns no names, if coord is nil (container reuse is off) or functionsPath does not exist.
func ApplyReuse(fs afero.Fs, coord *Coordinator, functionsPath string) ([]string, error) {
	if coord == nil {
		return nil, nil
	}

	info, err := fs.Stat(functionsPath)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to stat functions path: %w", err)
	}

	if info.IsDir() {
		entries, err := afero.ReadDir(fs, functionsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read functions directory: %w", err)
		}

		var names []string

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			if ext := filepath.Ext(e.Name()); ext != ".yaml" && ext != ".yml" {
				continue
			}

			fileNames, err := applyReuseFile(fs, coord, filepath.Join(functionsPath, e.Name()))
			if err != nil {
				return nil, err
			}

			names = append(names, fileNames...)
		}

		return sortedUnique(names), nil
	}

	names, err := applyReuseFile(fs, coord, functionsPath)
	if err != nil {
		return nil, err
	}

	return sortedUnique(names), nil
}

// applyReuseFile patches the Functions of one file, and returns the names of their containers.
func applyReuseFile(fs afero.Fs, coord *Coordinator, path string) ([]string, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read functions file %s: %w", path, err)
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))

	var (
		docs  [][]byte
		names []string
	)

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("failed to decode %s: %w", path, err)
		}

		if obj.Object == nil {
			continue
		}

		if obj.GetKind() == "Function" {
			name, err := patchFunctionAnnotations(coord, obj)
			if err != nil {
				return nil, err
			}

			if name != "" {
				names = append(names, name)
			}
		}

		out, err := yaml.Marshal(obj.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal function resource: %w", err)
		}

		docs = append(docs, out)
	}

	var buf bytes.Buffer

	for i, doc := range docs {
		if i > 0 {
			buf.WriteString("---\n")
		}

		buf.Write(doc)
	}

	if err := afero.WriteFile(fs, path, buf.Bytes(), 0o600); err != nil {
		return nil, fmt.Errorf("failed to write patched functions file %s: %w", path, err)
	}

	return names, nil
}

// patchFunctionAnnotations injects runtime-docker-name/runtime-docker-cleanup annotations into
// a single Function resource, unless the user already set them. It returns the name of the
// container of the Function (the one that was set, or the user's), or "" when the Function has
// no image reference to key reuse on.
func patchFunctionAnnotations(coord *Coordinator, obj *unstructured.Unstructured) (string, error) {
	imageRef, _, err := unstructured.NestedString(obj.Object, "spec", "package")
	if err != nil {
		return "", fmt.Errorf("failed to read spec.package: %w", err)
	}

	if imageRef == "" {
		// No image reference to key reuse on; leave this Function's annotations untouched.
		return "", nil
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if _, exists := annotations[annotationDockerName]; !exists {
		name := coord.ContainerName(imageRef)
		annotations[annotationDockerName] = name
		coord.track(name)
	}

	if _, exists := annotations[annotationDockerCleanup]; !exists {
		annotations[annotationDockerCleanup] = dockerCleanupOrphan
	}

	obj.SetAnnotations(annotations)

	return annotations[annotationDockerName], nil
}
