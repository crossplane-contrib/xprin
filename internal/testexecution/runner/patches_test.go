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
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// boolPtr is a helper function to create a pointer to a boolean value.
func boolPtr(b bool) *bool {
	return &b
}

// TestPatchXR tests the patchXR function directly.
func TestPatchXR(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a simple XR file
	xrContent := `apiVersion: example.org/v1
kind: XExample
metadata:
  name: test-xr
spec:
  field: value`
	xrFile := "/xr.yaml"
	require.NoError(t, afero.WriteFile(fs, xrFile, []byte(xrContent), 0o644))

	// Create output directory
	outputDir := "/output"
	require.NoError(t, fs.MkdirAll(outputDir, 0o755))

	// Create a runner
	options := &testexecutionUtils.Options{
		Debug: false,
	}
	runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
	runner.fs = fs // Use in-memory filesystem

	tests := []struct {
		name        string
		patches     api.Patches
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid - no connection secret fields",
			patches: api.Patches{},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly true with name",
			patches: api.Patches{
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr: false,
		},
		{
			name: "invalid - ConnectionSecretName without ConnectionSecret set",
			patches: api.Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "invalid - ConnectionSecretNamespace without ConnectionSecret set",
			patches: api.Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "invalid - both name and namespace without ConnectionSecret set",
			patches: api.Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "valid - ConnectionSecret false with name (disable)",
			patches: api.Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret false with namespace (disable)",
			patches: api.Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret false with both name and namespace (disable)",
			patches: api.Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runner.patchXR(xrFile, outputDir, tt.patches)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}
