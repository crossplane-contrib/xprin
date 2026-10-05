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
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

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
				ConnectionSecret:          new(true),
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
				ConnectionSecret:          new(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret false with namespace (disable)",
			patches: api.Patches{
				ConnectionSecret:          new(false),
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret false with both name and namespace (disable)",
			patches: api.Patches{
				ConnectionSecret:          new(false),
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

// xrdWithClaim is a minimal XRD YAML where the XR kind doesn't follow the "X" + Claim kind
// convention.
const xrdWithClaim = `
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: widgets.example.org
spec:
  group: example.org
  names:
    kind: Widget
    plural: widgets
  claimNames:
    kind: WidgetClaim
    plural: widgetclaims
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
`

// TestConvertClaimToXR tests the convertClaimToXR function directly, in particular its XRD-based
// kind resolution.
func TestConvertClaimToXR(t *testing.T) {
	claimContent := `apiVersion: example.org/v1alpha1
kind: WidgetClaim
metadata:
  name: test-claim
  namespace: default
spec:
  field: value`

	tests := []struct {
		name          string
		xrdPath       string
		goldenPath    string
		goldenContent string
		wantKind      string
		wantName      string // exact match if set; otherwise just assert a random suffix was generated
	}{
		{
			name:     "no XRD falls back to X-prefixed guess",
			xrdPath:  "",
			wantKind: "XWidgetClaim",
		},
		{
			name:     "XRD resolves the real XR kind",
			xrdPath:  "/xrd.yaml",
			wantKind: "Widget",
		},
		{
			name:       "golden file provides a reusable name",
			xrdPath:    "/xrd.yaml",
			goldenPath: "/golden.yaml",
			goldenContent: `apiVersion: example.org/v1alpha1
kind: Widget
metadata:
  name: test-claim-abcde`,
			wantKind: "Widget",
			wantName: "test-claim-abcde",
		},
		{
			name:       "golden file without a matching document falls back to a random suffix",
			xrdPath:    "/xrd.yaml",
			goldenPath: "/golden-nomatch.yaml",
			goldenContent: `apiVersion: example.org/v1alpha1
kind: SomethingElse
metadata:
  name: unrelated`,
			wantKind: "Widget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			claimFile := "/claim.yaml"
			require.NoError(t, afero.WriteFile(fs, claimFile, []byte(claimContent), 0o644))

			if tt.xrdPath != "" {
				require.NoError(t, afero.WriteFile(fs, tt.xrdPath, []byte(xrdWithClaim), 0o644))
			}

			if tt.goldenPath != "" {
				require.NoError(t, afero.WriteFile(fs, tt.goldenPath, []byte(tt.goldenContent), 0o644))
			}

			outputDir := "/output"
			require.NoError(t, fs.MkdirAll(outputDir, 0o755))

			options := &testexecutionUtils.Options{Debug: false}
			runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
			runner.fs = fs

			xrPath, err := runner.convertClaimToXR(claimFile, tt.xrdPath, tt.goldenPath, outputDir)
			require.NoError(t, err)

			xrData, err := afero.ReadFile(fs, xrPath)
			require.NoError(t, err)

			xr := &unstructured.Unstructured{}
			require.NoError(t, yaml.Unmarshal(xrData, xr))

			assert.Equal(t, tt.wantKind, xr.GetKind())

			if tt.wantName != "" {
				assert.Equal(t, tt.wantName, xr.GetName())
			} else {
				assert.True(t, strings.HasPrefix(xr.GetName(), "test-claim-"), "expected a random-suffixed name, got %q", xr.GetName())
				assert.NotEqual(t, "test-claim-abcde", xr.GetName())
			}
		})
	}
}

// TestFirstFullRenderGolden tests the pure function that picks the golden file to reuse an XR
// name from, out of a test case's assertions.
func TestFirstFullRenderGolden(t *testing.T) {
	tests := []struct {
		name       string
		assertions api.Assertions
		want       string
	}{
		{
			name:       "no assertions at all",
			assertions: api.Assertions{},
			want:       "",
		},
		{
			name: "only resource-scoped entries, no full-render one",
			assertions: api.Assertions{
				Diff: []api.AssertionGoldenFile{{Name: "a", Expected: "a.yaml", Resource: "Pod/foo"}},
				Dyff: []api.AssertionGoldenFile{{Name: "b", Expected: "b.yaml", Resource: "Pod/bar"}},
			},
			want: "",
		},
		{
			name: "full-render diff entry wins",
			assertions: api.Assertions{
				Diff: []api.AssertionGoldenFile{{Name: "a", Expected: "golden_diff.yaml"}},
				Dyff: []api.AssertionGoldenFile{{Name: "b", Expected: "golden_dyff.yaml"}},
			},
			want: "golden_diff.yaml",
		},
		{
			name: "falls back to full-render dyff entry when diff has none",
			assertions: api.Assertions{
				Diff: []api.AssertionGoldenFile{{Name: "a", Expected: "a.yaml", Resource: "Pod/foo"}},
				Dyff: []api.AssertionGoldenFile{{Name: "b", Expected: "golden_dyff.yaml"}},
			},
			want: "golden_dyff.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, firstFullRenderGolden(tt.assertions))
		})
	}
}

// TestUniqueBaseNamesForPaths tests the pure function that maps paths to unique base filenames.
func TestUniqueBaseNamesForPaths(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"nil returns empty", nil, nil},
		{"empty returns empty", []string{}, []string{}},
		{"single path keeps base name", []string{"/single/xrd.yaml"}, []string{"xrd.yaml"}},
		{"two files same base name", []string{"/aws/xrd.yaml", "/gcp/xrd.yaml"}, []string{"xrd.yaml", "xrd_1.yaml"}},
		{"two dirs same base name", []string{"/path/to/crds", "/another/path/to/crds"}, []string{"crds", "crds_1"}},
		{"three files same base name", []string{"/a/xrd.yaml", "/b/xrd.yaml", "/c/xrd.yaml"}, []string{"xrd.yaml", "xrd_1.yaml", "xrd_2.yaml"}},
		{"mixed unique names", []string{"/a/one.yaml", "/b/two.yaml"}, []string{"one.yaml", "two.yaml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueBaseNamesForPaths(tt.paths)
			assert.Equal(t, tt.want, got)
		})
	}
}
