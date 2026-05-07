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

package runner

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

func (r *Runner) copyXRInput(inputXR, observedResources string) (string, error) {
	copiedXR, err := r.copyInput(inputXR, "xr")
	if err != nil {
		return "", err
	}

	if observedResources == "" {
		return copiedXR, nil
	}

	xr, err := loadXRResource(r.fs, inputXR)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return copiedXR, nil
		}

		return "", err
	}

	observed, err := loadObservedResources(r.fs, observedResources)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return copiedXR, nil
		}

		return "", err
	}

	for _, candidate := range observed {
		if !sameResource(candidate, xr) {
			continue
		}

		out, err := yaml.Marshal(candidate.Object)
		if err != nil {
			return "", err
		}

		if err := afero.WriteFile(r.fs, copiedXR, out, 0o600); err != nil {
			return "", err
		}

		return copiedXR, nil
	}

	return copiedXR, nil
}

func loadXRResource(fs afero.Fs, path string) (*unstructured.Unstructured, error) {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	xr := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(data, xr); err != nil {
		return nil, fmt.Errorf("failed to parse XR YAML: %w", err)
	}

	return xr, nil
}

func loadObservedResources(fs afero.Fs, path string) ([]*unstructured.Unstructured, error) {
	info, err := fs.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		files, err := afero.ReadDir(fs, path)
		if err != nil {
			return nil, err
		}

		var resources []*unstructured.Unstructured
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			switch filepath.Ext(file.Name()) {
			case ".yaml", ".yml":
			default:
				continue
			}

			docs, err := loadObservedResources(fs, filepath.Join(path, file.Name()))
			if err != nil {
				return nil, err
			}
			resources = append(resources, docs...)
		}

		return resources, nil
	}

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
	resources := make([]*unstructured.Unstructured, 0)

	for {
		resource := &unstructured.Unstructured{}
		if err := decoder.Decode(resource); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}
		if resource.GetAPIVersion() == "" && resource.GetKind() == "" && resource.GetName() == "" {
			continue
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func sameResource(a, b *unstructured.Unstructured) bool {
	return a.GetAPIVersion() == b.GetAPIVersion() &&
		a.GetKind() == b.GetKind() &&
		a.GetName() == b.GetName() &&
		a.GetNamespace() == b.GetNamespace()
}
