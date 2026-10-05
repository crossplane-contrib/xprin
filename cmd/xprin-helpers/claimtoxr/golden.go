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

// FindNameInGolden scans goldenPath - a YAML stream of one or more documents, e.g. a full render
// output - for the first document whose kind matches wantKind, and returns its metadata.name.
//
// Matching by kind (rather than trusting a fixed position, e.g. "the XR is always the first
// document") matters here specifically because, unlike xprin's own runner - which only ever reads
// back a file it rendered itself - this is reachable from the standalone `--golden` CLI flag, which
// accepts whatever file a user points it at; trusting document order would silently return the
// wrong resource's name for anything that isn't a real, XR-first render output.
//
// found is false, with a nil error, both when goldenPath doesn't exist yet and when it exists but
// contains no document of that kind. Neither is an error: a missing or non-matching golden file
// just means there's nothing to reproduce a name from yet (e.g. before the first
// `xprin update-goldens` capture), so the caller should fall back to its own default behaviour.
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
