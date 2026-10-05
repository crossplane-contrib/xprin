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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const singleDocGolden = `
apiVersion: example.org/v1alpha1
kind: XWidget
metadata:
  name: platform-widget-abcde
`

const multiDocGolden = `
apiVersion: example.org/v1alpha1
kind: XWidget
metadata:
  name: platform-widget-abcde
---
apiVersion: compute.example.org/v1beta1
kind: Firewall
metadata:
  name: platform-widget-abcde-firewall
`

func TestFindNameInGolden(t *testing.T) {
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, "/single.yaml", []byte(singleDocGolden), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/multi.yaml", []byte(multiDocGolden), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/malformed.yaml", []byte(":\nnot: [valid"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/empty.yaml", []byte(""), 0o644))

	t.Run("found in single-document golden", func(t *testing.T) {
		name, found, err := FindNameInGolden(fs, "/single.yaml", "XWidget")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "platform-widget-abcde", name)
	})

	t.Run("found in multi-document golden, not the first document", func(t *testing.T) {
		name, found, err := FindNameInGolden(fs, "/multi.yaml", "Firewall")
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "platform-widget-abcde-firewall", name)
	})

	t.Run("not found: golden file does not exist", func(t *testing.T) {
		name, found, err := FindNameInGolden(fs, "/does-not-exist.yaml", "XWidget")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, name)
	})

	t.Run("not found: no document of the wanted kind", func(t *testing.T) {
		name, found, err := FindNameInGolden(fs, "/single.yaml", "XOther")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, name)
	})

	t.Run("not found: empty golden file", func(t *testing.T) {
		name, found, err := FindNameInGolden(fs, "/empty.yaml", "XWidget")
		require.NoError(t, err)
		assert.False(t, found)
		assert.Empty(t, name)
	})

	t.Run("error: malformed golden file", func(t *testing.T) {
		_, found, err := FindNameInGolden(fs, "/malformed.yaml", "XWidget")
		require.Error(t, err)
		assert.False(t, found)
	})
}
