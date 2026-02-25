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

package patchxr

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crossplane/crossplane-runtime/v2/pkg/test"

	"github.com/crossplane/crossplane/v2/cmd/crank/render"
)

// testXR is a basic test XR for testing patch functionality.
var testXR = &unstructured.Unstructured{
	Object: map[string]any{
		"apiVersion": "example.org/v1alpha1",
		"kind":       "XTestApp",
		"metadata": map[string]any{
			"name": "test-app",
		},
		"spec": map[string]any{},
	},
}

// validTestXRD returns a valid XRD YAML string for testing.
func validTestXRD() string {
	return `apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xtestapps.example.org
spec:
  group: example.org
  names:
    kind: XTestApp
    plural: xtestapps
  versions:
  - name: v1alpha1
    served: true
    referenceable: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              title:
                type: string
                default: "Default Title"
              replicas:
                type: integer
                default: 3
              config:
                type: object
                properties:
                  environment:
                    type: string
                    default: "production"
                  timeout:
                    type: integer
                    default: 30`
}

func TestHasPatchingFlags(t *testing.T) {
	tests := []struct {
		name                      string
		xrd                       string
		addConnectionSecret       bool
		connectionSecretName      string
		connectionSecretNamespace string
		want                      bool
	}{
		{
			name: "no flags",
			want: false,
		},
		{
			name: "xrd flag only",
			xrd:  "xrd.yaml",
			want: true,
		},
		{
			name:                "add-connection-secret true",
			addConnectionSecret: true,
			want:                true,
		},
		{
			name:                "add-connection-secret false",
			addConnectionSecret: false,
			want:                false,
		},
		{
			name:                "xrd and connection secret",
			xrd:                 "xrd.yaml",
			addConnectionSecret: true,
			want:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cmd{
				XRD:                       tt.xrd,
				AddConnectionSecret:       tt.addConnectionSecret,
				ConnectionSecretName:      tt.connectionSecretName,
				ConnectionSecretNamespace: tt.connectionSecretNamespace,
			}
			if got := c.hasPatchingFlags(); got != tt.want {
				t.Errorf("Cmd.hasPatchingFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultValuesFromXRD(t *testing.T) {
	tests := []struct {
		name       string
		xr         map[string]any
		apiVersion string
		xrd        string
		want       map[string]any
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "apply defaults for missing fields",
			apiVersion: "example.org/v1alpha1",
			xr: map[string]any{
				"spec": map[string]any{
					"replicas": 5,
					"config": map[string]any{
						"timeout": 60,
					},
				},
			},
			xrd: validTestXRD(),
			want: map[string]any{
				"spec": map[string]any{
					"title":    "Default Title", // Added by default
					"replicas": 5,               // Preserved (not overridden)
					"config": map[string]any{
						"environment": "production", // Added by default
						"timeout":     60,           // Preserved (not overridden)
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "no defaults when fields exist",
			apiVersion: "example.org/v1alpha1",
			xr: map[string]any{
				"spec": map[string]any{
					"title":    "Custom Title",
					"replicas": 10,
					"config": map[string]any{
						"environment": "staging",
						"timeout":     120,
					},
				},
			},
			xrd: validTestXRD(),
			want: map[string]any{
				"spec": map[string]any{
					"title":    "Custom Title", // Preserved
					"replicas": 10,             // Preserved
					"config": map[string]any{
						"environment": "staging", // Preserved
						"timeout":     120,       // Preserved
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "version not found",
			apiVersion: "example.org/v2beta1",
			xr: map[string]any{
				"spec": map[string]any{},
			},
			xrd: `apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xtestapps.example.org
spec:
  group: example.org
  names:
    kind: XTestApp
    plural: xtestapps
  versions:
  - name: v1alpha1
    served: true
    referenceable: true`,
			wantErr: true,
			errMsg:  "the specified API version 'example.org/v2beta1' does not exist in the XRD",
		},
		{
			name:       "no schema in version",
			apiVersion: "example.org/v1alpha1",
			xr: map[string]any{
				"spec": map[string]any{},
			},
			xrd: `apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xtestapps.example.org
spec:
  group: example.org
  names:
    kind: XTestApp
    plural: xtestapps
  versions:
  - name: v1alpha1
    served: true
    referenceable: true`,
			want: map[string]any{
				"spec": map[string]any{},
			},
			wantErr: false,
		},
		{
			name:       "invalid schema JSON",
			apiVersion: "example.org/v1alpha1",
			xr: map[string]any{
				"spec": map[string]any{},
			},
			xrd: `apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xtestapps.example.org
spec:
  group: example.org
  names:
    kind: XTestApp
    plural: xtestapps
  versions:
  - name: v1alpha1
    served: true
    referenceable: true
    schema:
      openAPIV3Schema: "invalid json"`,
			wantErr: true,
			errMsg:  "failed to unmarshal OpenAPIV3Schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if err := afero.WriteFile(fs, "test-xrd.yaml", []byte(tt.xrd), 0o644); err != nil {
				t.Fatalf("Failed to write test XRD file: %v", err)
			}

			xrd, err := render.LoadXRD(fs, "test-xrd.yaml")
			if err != nil {
				t.Fatalf("Failed to load test XRD: %v", err)
			}

			err = DefaultValuesFromXRD(tt.xr, tt.apiVersion, *xrd)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DefaultValuesFromXRD() expected error but got none")
					return
				}

				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DefaultValuesFromXRD() error = %v, want error containing %v", err, tt.errMsg)
				}

				return
			}

			if err != nil {
				t.Errorf("DefaultValuesFromXRD() unexpected error = %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, tt.xr); diff != "" {
				t.Errorf("DefaultValuesFromXRD() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// isValidUUID checks if a string is a valid UUID.
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func TestAddConnectionSecret(t *testing.T) {
	type args struct {
		xr                        *unstructured.Unstructured
		connectionSecretName      string
		connectionSecretNamespace string
	}

	type want struct {
		err error
		xr  *unstructured.Unstructured
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"WithConnectionSecret": {
			reason: "Should add writeConnectionSecretToRef when adding connection secret",
			args: args{
				xr:                        testXR.DeepCopy(),
				connectionSecretName:      "",
				connectionSecretNamespace: "",
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]any{
							"name": "test-app",
						},
						"spec": map[string]any{
							"writeConnectionSecretToRef": map[string]any{
								"name":      "generated-uuid", // Will be ignored in comparison
								"namespace": "default",
							},
						},
					},
				},
			},
		},
		"WithCustomConnectionSecretName": {
			reason: "Should use custom connection secret name when provided",
			args: args{
				xr:                        testXR.DeepCopy(),
				connectionSecretName:      "my-custom-secret",
				connectionSecretNamespace: "",
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]any{
							"name": "test-app",
						},
						"spec": map[string]any{
							"writeConnectionSecretToRef": map[string]any{
								"name":      "my-custom-secret",
								"namespace": "default",
							},
						},
					},
				},
			},
		},
		"WithCustomConnectionSecretNamespace": {
			reason: "Should use custom connection secret namespace when provided",
			args: args{
				xr:                        testXR.DeepCopy(),
				connectionSecretName:      "",
				connectionSecretNamespace: "custom-namespace",
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]any{
							"name": "test-app",
						},
						"spec": map[string]any{
							"writeConnectionSecretToRef": map[string]any{
								"name":      "generated-uuid", // Will be ignored in comparison
								"namespace": "custom-namespace",
							},
						},
					},
				},
			},
		},
		"WithBothCustomConnectionSecretNameAndNamespace": {
			reason: "Should use both custom connection secret name and namespace when provided",
			args: args{
				xr:                        testXR.DeepCopy(),
				connectionSecretName:      "my-custom-secret",
				connectionSecretNamespace: "custom-namespace",
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]any{
							"name": "test-app",
						},
						"spec": map[string]any{
							"writeConnectionSecretToRef": map[string]any{
								"name":      "my-custom-secret",
								"namespace": "custom-namespace",
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Create a copy of the test XR
			got := tc.args.xr.DeepCopy()

			// Add connection secret
			err := AddConnectionSecret(got, tc.args.connectionSecretName, tc.args.connectionSecretNamespace)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nAddConnectionSecret(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if err != nil {
				return
			}

			// Use custom comparison to ignore generated values
			opt := cmpopts.IgnoreMapEntries(func(k, v any) bool {
				key, ok := k.(string)
				if !ok {
					return false
				}
				// Ignore generated uid field in metadata
				if key == "uid" {
					if vStr, ok := v.(string); ok && isValidUUID(vStr) {
						return true
					}
				}
				// Ignore generated name field in writeConnectionSecretToRef when it's a UUID
				if key == "name" {
					if vStr, ok := v.(string); ok && isValidUUID(vStr) {
						return true
					}
				}

				return false
			})

			if diff := cmp.Diff(tc.want.xr, got, opt); diff != "" {
				t.Errorf("\n%s\nAddConnectionSecret(...): -want, +got:\n%s", tc.reason, diff)
			}

			// Validate connection secret always has name and namespace
			if got != nil {
				if got.Object["spec"] != nil {
					if spec, ok := got.Object["spec"].(map[string]any); ok {
						if writeConnSecret, ok := spec["writeConnectionSecretToRef"].(map[string]any); ok {
							if secretName, ok := writeConnSecret["name"].(string); !ok || secretName == "" {
								t.Errorf("\n%s\nConnection secret name should always be present and non-empty", tc.reason)
							}

							if secretNamespace, ok := writeConnSecret["namespace"].(string); !ok || secretNamespace == "" {
								t.Errorf("\n%s\nConnection secret namespace should always be present and non-empty", tc.reason)
							}
						} else {
							t.Errorf("\n%s\nwriteConnectionSecretToRef should be present when adding connection secret", tc.reason)
						}
					}
				}
			}
		})
	}
}

func TestAddConnectionSecret_UUIDGeneration(t *testing.T) {
	t.Run("GeneratesUUIDWhenNoNameProvided", func(t *testing.T) {
		// Create a test XR object
		got := testXR.DeepCopy()

		// Add connection secret to test UUID generation
		if err := AddConnectionSecret(got, "", ""); err != nil {
			t.Errorf("AddConnectionSecret() error = %v, want nil", err)
			return
		}

		// Check that a UID was generated and set
		if uid, ok := got.Object["metadata"].(map[string]any)["uid"].(string); !ok || uid == "" {
			t.Error("Expected metadata.uid to be set")
		}

		// Check that writeConnectionSecretToRef name was generated (should be the UID)
		spec, ok := got.Object["spec"].(map[string]any)
		if !ok {
			t.Error("Expected spec to be present")
			return
		}

		writeConnSecret, ok := spec["writeConnectionSecretToRef"].(map[string]any)
		if !ok {
			t.Error("Expected writeConnectionSecretToRef to be present")
			return
		}

		secretName, ok := writeConnSecret["name"].(string)
		if !ok || secretName == "" {
			t.Error("Expected writeConnectionSecretToRef.name to be present and non-empty")
			return
		}

		// Validate UUID format
		if !isValidUUID(secretName) {
			t.Errorf("Expected writeConnectionSecretToRef.name to be a UUID, got: %s", secretName)
		}

		// Validate namespace is set correctly
		secretNamespace, ok := writeConnSecret["namespace"].(string)
		if !ok || secretNamespace != "default" {
			t.Errorf("Expected namespace 'default', got %v", secretNamespace)
		}

		// Check that .metadata.uid is created and is a valid UUID
		metadata, ok := got.Object["metadata"].(map[string]any)
		if !ok {
			t.Error("XR metadata is not a map")
			return
		}

		uid, ok := metadata["uid"].(string)
		if !ok || uid == "" {
			t.Error("XR metadata.uid not found or empty")
			return
		}

		if !isValidUUID(uid) {
			t.Errorf("XR metadata.uid is not a valid UUID: %s", uid)
		}
	})
}
