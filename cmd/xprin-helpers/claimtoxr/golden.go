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

package claimtoxr

import (
	"bytes"
	"io"
	"os"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
)

// FindNameInGolden scans goldenPath, a YAML stream (e.g. a full render output), for the first document of kind
// wantKind, and returns its metadata.name.
//
// It matches by kind and not by position (the XR first) because `--golden` accepts any file that a user points it at.
//
// found is false, with a nil error, when goldenPath doesn't exist yet or has no document of that kind: there is no
// name to reproduce yet (before the first `xprin update-goldens`), and the caller uses its default.
func FindNameInGolden(fs afero.Fs, goldenPath, wantKind string) (name string, found bool, err error) {
	data, err := afero.ReadFile(fs, goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}

		return "", false, errors.Wrap(err, "failed to read golden file")
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				return "", false, nil
			}

			return "", false, errors.Wrap(err, "failed to parse golden file")
		}

		if obj.GetKind() == wantKind {
			return obj.GetName(), true, nil
		}
	}
}
