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
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/config"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestCopyXRInputUsesMatchingObservedXR(t *testing.T) {
	fs, r := newCopyXRInputTestRunner(t)

	require.NoError(t, afero.WriteFile(fs, "/xr.yaml", []byte(`
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: original
`), 0o600))
	require.NoError(t, afero.WriteFile(fs, "/observed.yaml", []byte(`
apiVersion: example.org/v1
kind: Other
metadata:
  name: example
---
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: observed
status:
  ready: true
`), 0o600))

	got, err := r.copyXRInput("/xr.yaml", "/observed.yaml")
	require.NoError(t, err)
	require.Equal(t, "/inputs/xr/xr.yaml", got)

	data, err := afero.ReadFile(fs, got)
	require.NoError(t, err)
	require.Contains(t, string(data), "value: observed")
	require.Contains(t, string(data), "ready: true")
}

func TestCopyXRInputUsesMatchingObservedXRFromDirectory(t *testing.T) {
	fs, r := newCopyXRInputTestRunner(t)

	require.NoError(t, afero.WriteFile(fs, "/xr.yaml", []byte(`
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: original
`), 0o600))
	require.NoError(t, fs.MkdirAll("/observed", 0o700))
	require.NoError(t, afero.WriteFile(fs, "/observed/xr.yaml", []byte(`
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: observed
status:
  ready: true
`), 0o600))
	require.NoError(t, afero.WriteFile(fs, "/observed/ignored.txt", []byte(`
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: ignored
`), 0o600))

	got, err := r.copyXRInput("/xr.yaml", "/observed")
	require.NoError(t, err)
	require.Equal(t, "/inputs/xr/xr.yaml", got)

	data, err := afero.ReadFile(fs, got)
	require.NoError(t, err)
	require.Contains(t, string(data), "value: observed")
	require.Contains(t, string(data), "ready: true")
}

func TestCopyXRInputKeepsOriginalXRWithoutObservedMatch(t *testing.T) {
	fs, r := newCopyXRInputTestRunner(t)

	require.NoError(t, afero.WriteFile(fs, "/xr.yaml", []byte(`
apiVersion: example.org/v1
kind: XExample
metadata:
  name: example
spec:
  value: original
`), 0o600))
	require.NoError(t, afero.WriteFile(fs, "/observed.yaml", []byte(`
apiVersion: example.org/v1
kind: Other
metadata:
  name: example
spec:
  value: observed
`), 0o600))

	got, err := r.copyXRInput("/xr.yaml", "/observed.yaml")
	require.NoError(t, err)
	require.Equal(t, "/inputs/xr/xr.yaml", got)

	data, err := afero.ReadFile(fs, got)
	require.NoError(t, err)
	require.Contains(t, string(data), "value: original")
	require.NotContains(t, string(data), "value: observed")
}

func newCopyXRInputTestRunner(t *testing.T) (afero.Fs, *Runner) {
	t.Helper()

	fs := afero.NewMemMapFs()
	r := NewRunner(&testexecutionUtils.Options{
		Dependencies: map[string]string{"crossplane": config.CrossplaneCmd},
	}, testSuiteFile, nil)
	r.fs = fs
	r.inputsDir = "/inputs"
	r.copy = func(src, dest string, _ ...cp.Options) error {
		data, err := afero.ReadFile(fs, src)
		if err != nil {
			return err
		}
		if err := fs.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		return afero.WriteFile(fs, dest, data, 0o644)
	}

	require.NoError(t, fs.MkdirAll("/inputs", 0o700))

	return fs, r
}
