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

package claimtoxr

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

// Helper functions for common Claim modifications

// generateTestClaim creates a test Claim object with a default structure.
// It accepts optional modifier functions that can customize the Claim's content.
func generateTestClaim(opts ...func(*unstructured.Unstructured)) *unstructured.Unstructured {
	claim := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.org/v1alpha1",
			"kind":       "TestApp",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "myclaims",
			},
			"spec": map[string]interface{}{},
		},
	}

	for _, opt := range opts {
		opt(claim)
	}

	return claim
}

// Shared test data variables.
var (
	testClaim = generateTestClaim()

	testClaimWithLabels = generateTestClaim(
		withLabels(map[string]interface{}{
			"existing-label": "value",
		}),
	)

	testClaimWithComplexSpec = generateTestClaim(
		withComplexSpec(),
	)

	testClaimWithAnnotations = generateTestClaim(
		withAnnotations(map[string]interface{}{
			"test-annotation": "value",
		}),
	)

	testClaimInvalidAPIVersion = generateTestClaim(
		withAPIVersion("example.org/v1/v2"),
	)

	testClaimWithMultiFieldSpec = generateTestClaim(
		withMultiFieldSpec(),
	)
)

// withoutMandatoryField returns an option function that removes a mandatory field from the Claim.
// This is used to test error handling when processing Claims without a mandatory field.
func withoutMandatoryField(field string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		delete(u.Object, field)
	}
}

// withMultiFieldSpec returns an option function that adds multiple fields of different types
// to the Claim's spec. This is used to test proper handling of different field types.
func withMultiFieldSpec() func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		u.Object["spec"] = map[string]interface{}{
			"stringField": "value1",
			"numberField": float64(42), // YAML unmarshals integers as float64
			"boolField":   true,
			"objectField": map[string]interface{}{
				"nested": "value",
			},
			"arrayField": []interface{}{
				"item1",
				"item2",
			},
		}
	}
}

// withLabels returns an option function that adds the specified labels to the Claim's metadata.
// This is used to test proper handling and merging of existing labels with Crossplane-specific labels.
func withLabels(labels map[string]interface{}) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		meta := u.Object["metadata"].(map[string]interface{})
		meta["labels"] = labels
	}
}

// withAnnotations returns an option function that adds the specified annotations to the Claim's metadata.
// This is used to test proper copying of annotations from Claim to XR.
func withAnnotations(annotations map[string]interface{}) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		meta := u.Object["metadata"].(map[string]interface{})
		meta["annotations"] = annotations
	}
}

// withAPIVersion returns an option function that sets the apiVersion field of the Claim.
// This is used to test handling of different API versions.
func withAPIVersion(apiVersion string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		u.Object["apiVersion"] = apiVersion
	}
}

// withComplexSpec returns an option function that adds a complex nested spec to the Claim.
func withComplexSpec() func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		u.Object["spec"] = map[string]interface{}{
			"intField":    float64(42), // YAML unmarshals integers as float64
			"floatField":  float64(3.14159),
			"stringField": "hello",
			"boolField":   true,
			"objectField": map[string]interface{}{
				"nestedInt":   float64(123), // YAML unmarshals integers as float64
				"nestedFloat": float64(456.789),
				"nestedObject": map[string]interface{}{
					"deeplyNested": "value",
				},
			},
			"arrayField": []interface{}{
				float64(1234), // YAML unmarshals integers as float64
				"string in array",
				map[string]interface{}{
					"objectInArray": true,
				},
			},
		}
	}
}

// generateExpectedXR creates an expected XR object based on a Claim, applying standard transformations.
func generateExpectedXR(claim *unstructured.Unstructured, kind string, direct bool, opts ...func(*unstructured.Unstructured)) *unstructured.Unstructured {
	name := claim.GetName()
	if !direct {
		name += "-abcde"
	}

	xrKind := kind
	if xrKind == "" {
		xrKind = "X" + claim.GetKind()
	}

	// Create base XR
	xr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "example.org/v1alpha1",
			"kind":       xrKind,
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{},
		},
	}

	if !direct {
		// Add Crossplane labels
		metadata := xr.Object["metadata"].(map[string]interface{})
		metadata["labels"] = map[string]any{
			"crossplane.io/claim-name":      claim.GetName(),
			"crossplane.io/claim-namespace": claim.GetNamespace(),
		}

		// Add claimRef
		spec := xr.Object["spec"].(map[string]interface{})
		spec["claimRef"] = map[string]interface{}{
			"apiVersion": claim.GetAPIVersion(),
			"kind":       claim.GetKind(),
			"name":       claim.GetName(),
			"namespace":  claim.GetNamespace(),
		}
	}

	// Apply any additional options
	for _, opt := range opts {
		opt(xr)
	}

	return xr
}

func TestConvertClaimToXR(t *testing.T) {
	type args struct {
		claim  *unstructured.Unstructured
		kind   string
		direct bool
	}

	type want struct {
		xr      *unstructured.Unstructured
		err     error
		nameLen int // Length of the generated name, for validation of suffix length
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"NilClaim": {
			reason: "Should return error when Claim is nil",
			args: args{
				claim:  nil,
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.New(errNilInput),
			},
		},
		"EmptyObject": {
			reason: "Should return error when Claim object is nil",
			args: args{
				claim:  &unstructured.Unstructured{},
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.New(errEmptyClaimYAML),
			},
		},
		"Direct": {
			reason: "Should keep original name when Direct is true",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: true,
			},
			want: want{
				xr:      generateExpectedXR(testClaim, "", true),
				err:     nil,
				nameLen: len("test-app"),
			},
		},
		"NotDirect": {
			reason: "Should append random suffix when Direct is false",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: false,
			},
			want: want{
				xr:      generateExpectedXR(testClaim, "", false),
				err:     nil,
				nameLen: len("test-app-") + 5, // name length should be original name + hyphen + 5 char suffix
			},
		},
		"ErrNoAPIVersion": {
			reason: "Should return error when Claim has no apiVersion",
			args: args{
				claim:  generateTestClaim(withoutMandatoryField("apiVersion")),
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.New(errNoAPIVersion),
			},
		},
		"InvalidAPIVersion": {
			reason: "Should return error for invalid API version",
			args: args{
				claim:  testClaimInvalidAPIVersion,
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.Wrap(errors.New("unexpected GroupVersion string: example.org/v1/v2"), errParseAPIVersion),
			},
		},
		"ErrNoKind": {
			reason: "Should return error when Claim has no kind",
			args: args{
				claim:  generateTestClaim(withoutMandatoryField("kind")),
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.New(errNoKind),
			},
		},
		"ErrNoSpec": {
			reason: "Should return error when Claim has no spec",
			args: args{
				claim:  generateTestClaim(withoutMandatoryField("spec")),
				kind:   "",
				direct: false,
			},
			want: want{
				xr:  nil,
				err: errors.New(errNoSpecSection),
			},
		},
		"PreservesComplexSpec": {
			reason: "Should preserve complex spec fields and their native types when converting from Claim to XR",
			args: args{
				claim:  testClaimWithComplexSpec,
				kind:   "",
				direct: true,
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app",
						},
						"spec": map[string]interface{}{
							"intField":    int64(42), // Whole numbers become int64
							"floatField":  float64(3.14159),
							"stringField": "hello",
							"boolField":   true,
							"objectField": map[string]interface{}{
								"nestedInt":   int64(123), // Whole numbers become int64
								"nestedFloat": float64(456.789),
								"nestedObject": map[string]interface{}{
									"deeplyNested": "value",
								},
							},
							"arrayField": []interface{}{
								int64(1234), // Whole numbers become int64
								"string in array",
								map[string]interface{}{
									"objectInArray": true,
								},
							},
						},
					},
				},
			},
		},
		"StandardLabelsWithoutExistingLabels": {
			reason: "Should add standard Crossplane labels when no other labels exist",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: false,
			},
			want: want{
				err:     nil,
				xr:      generateExpectedXR(testClaim, "", false),
				nameLen: len("test-app-") + 5,
			},
		},
		"LabelsHandling": {
			reason: "Should properly merge existing labels with Crossplane labels",
			args: args{
				claim: generateTestClaim(
					withLabels(map[string]interface{}{
						"existing-label": "value",
						labelClaimName:   "old-value",
					}),
				),
				kind:   "",
				direct: false,
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app-abcde",
							"labels": map[string]any{
								"existing-label":    "value",
								labelClaimName:      "test-app",
								labelClaimNamespace: "myclaims",
							},
						},
						"spec": map[string]interface{}{
							"claimRef": map[string]interface{}{
								"apiVersion": "example.org/v1alpha1",
								"kind":       "TestApp",
								"name":       "test-app",
								"namespace":  "myclaims",
							},
						},
					},
				},
			},
		},
		"WithAnnotations": {
			reason: "Should properly copy annotations from Claim to XR",
			args: args{
				claim:  testClaimWithAnnotations,
				kind:   "",
				direct: true,
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app",
							"annotations": map[string]any{
								"test-annotation": "value",
							},
						},
						"spec": map[string]any{},
					},
				},
			},
		},
		"NoNamespaceInXR": {
			reason: "Should not include namespace in XR metadata as XRs are cluster-scoped",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: false,
			},
			want: want{
				err: nil,
				xr:  generateExpectedXR(testClaim, "", false),
			},
		},
		"CustomKindFlag": {
			reason: "Should use provided kind instead of deriving from Claim kind",
			args: args{
				claim:  testClaim,
				kind:   "CustomKind",
				direct: false,
			},
			want: want{
				xr:  generateExpectedXR(testClaim, "CustomKind", false),
				err: nil,
			},
		},
		"DirectCustomKindFlag": {
			reason: "Direct XR should use provided kind instead of deriving from Claim kind",
			args: args{
				claim:  testClaim,
				kind:   "CustomKind",
				direct: true,
			},
			want: want{
				xr:  generateExpectedXR(testClaim, "CustomKind", true),
				err: nil,
			},
		},
		"DirectNoLabels": {
			reason: "Direct XR should have no Crossplane labels",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: true,
			},
			want: want{
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app",
						},
						"spec": map[string]interface{}{},
					},
				},
				err:     nil,
				nameLen: len("test-app"),
			},
		},
		"DirectWithExistingLabels": {
			reason: "Direct XR should keep existing labels but not add Crossplane labels",
			args: args{
				claim:  testClaimWithLabels,
				kind:   "",
				direct: true,
			},
			want: want{
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app",
							"labels": map[string]interface{}{
								"existing-label": "value",
							},
						},
						"spec": map[string]interface{}{},
					},
				},
				err:     nil,
				nameLen: len("test-app"),
			},
		},
		"LabelsShouldMatchGeneratedName": {
			reason: "Labels should match the generated name in XR",
			args: args{
				claim:  testClaim,
				kind:   "",
				direct: false,
			},
			want: want{
				xr:      generateExpectedXR(testClaim, "", false),
				err:     nil,
				nameLen: len("test-app-") + 5, // name length should be original name + hyphen + 5 char suffix
			},
		},
		"CopyAllSpecFields": {
			reason: "Should copy all spec fields from Claim to XR, preserving types",
			args: args{
				claim:  testClaimWithMultiFieldSpec,
				kind:   "",
				direct: true,
			},
			want: want{
				err: nil,
				xr: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "example.org/v1alpha1",
						"kind":       "XTestApp",
						"metadata": map[string]interface{}{
							"name": "test-app",
						},
						"spec": map[string]interface{}{
							"stringField": "value1",
							"numberField": int64(42), // Whole numbers become int64
							"boolField":   true,
							"objectField": map[string]interface{}{
								"nested": "value",
							},
							"arrayField": []interface{}{
								"item1",
								"item2",
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ConvertClaimToXR(tc.args.claim, tc.args.kind, tc.args.direct)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nConvertClaimToXR(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			// Use custom comparison to ignore generated name suffixes
			opt := cmpopts.IgnoreMapEntries(func(k, v interface{}) bool {
				key, ok := k.(string)
				if !ok {
					return false
				}
				// Ignore generated name suffixes and composite label values
				if key == "name" && v.(string) != "test-app" {
					return true
				}

				if key == "crossplane.io/composite" && strings.HasPrefix(v.(string), "test-app-") {
					return true
				}

				return false
			})
			if diff := cmp.Diff(tc.want.xr, got, opt); diff != "" {
				t.Errorf("\n%s\nConvertClaimToXR(...): -want, +got:\n%s", tc.reason, diff)
			}

			// Verify name length if specified in test case
			if tc.want.nameLen > 0 && got != nil {
				gotName, exists, err := unstructured.NestedString(got.Object, "metadata", "name")
				if err != nil {
					t.Errorf("\n%s\nError getting name: %v", tc.reason, err)
				}

				if !exists {
					t.Errorf("\n%s\nName field not found in output", tc.reason)
				}

				if len(gotName) != tc.want.nameLen {
					t.Errorf("\n%s\nName length mismatch: want %d, got %d", tc.reason, tc.want.nameLen, len(gotName))
				}
			}
		})
	}
}
