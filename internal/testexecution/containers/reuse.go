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

// ApplyReuse patches Function resources in functionsPath (a file or a directory of files) with
// container-reuse annotations, keyed by each function's image reference via coord.
//
// Does nothing if coord is nil (container reuse not enabled — the default unless running
// through processor.ProcessTargets) or if functionsPath does not exist (e.g. when copy is
// mocked out in tests).
func ApplyReuse(fs afero.Fs, coord *Coordinator, functionsPath string) error {
	if coord == nil {
		return nil
	}

	info, err := fs.Stat(functionsPath)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to stat functions path: %w", err)
	}

	if info.IsDir() {
		entries, err := afero.ReadDir(fs, functionsPath)
		if err != nil {
			return fmt.Errorf("failed to read functions directory: %w", err)
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}

			if ext := filepath.Ext(e.Name()); ext != ".yaml" && ext != ".yml" {
				continue
			}

			if err := applyReuseFile(fs, coord, filepath.Join(functionsPath, e.Name())); err != nil {
				return err
			}
		}

		return nil
	}

	return applyReuseFile(fs, coord, functionsPath)
}

func applyReuseFile(fs afero.Fs, coord *Coordinator, path string) error {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return fmt.Errorf("failed to read functions file %s: %w", path, err)
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))

	var docs [][]byte

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("failed to decode %s: %w", path, err)
		}

		if obj.Object == nil {
			continue
		}

		if obj.GetKind() == "Function" {
			if err := patchFunctionAnnotations(coord, obj); err != nil {
				return err
			}
		}

		out, err := yaml.Marshal(obj.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal function resource: %w", err)
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
		return fmt.Errorf("failed to write patched functions file %s: %w", path, err)
	}

	return nil
}

// patchFunctionAnnotations injects runtime-docker-name/runtime-docker-cleanup annotations into
// a single Function resource, unless the user already set them.
func patchFunctionAnnotations(coord *Coordinator, obj *unstructured.Unstructured) error {
	imageRef, _, err := unstructured.NestedString(obj.Object, "spec", "package")
	if err != nil {
		return fmt.Errorf("failed to read spec.package: %w", err)
	}

	if imageRef == "" {
		// No image reference to key reuse on; leave this Function's annotations untouched.
		return nil
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if _, exists := annotations[annotationDockerName]; !exists {
		name := coord.ContainerName(imageRef)
		annotations[annotationDockerName] = name
		coord.names[name] = struct{}{}
	}

	if _, exists := annotations[annotationDockerCleanup]; !exists {
		annotations[annotationDockerCleanup] = dockerCleanupOrphan
	}

	obj.SetAnnotations(annotations)

	return nil
}
