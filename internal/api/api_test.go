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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// boolPtr is a helper function to create a pointer to a boolean value.
func boolPtr(b bool) *bool {
	return &b
}

func TestPatches_hasConnectionSecret(t *testing.T) {
	tests := []struct {
		name     string
		patches  Patches
		expected bool
	}{
		{
			name: "ConnectionSecret nil",
			patches: Patches{
				ConnectionSecret: nil,
			},
			expected: false,
		},
		{
			name: "ConnectionSecret explicitly true",
			patches: Patches{
				ConnectionSecret: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "ConnectionSecret explicitly false",
			patches: Patches{
				ConnectionSecret: boolPtr(false),
			},
			expected: false,
		},
		{
			name: "ConnectionSecret true with name",
			patches: Patches{
				ConnectionSecret:     boolPtr(true),
				ConnectionSecretName: "my-secret",
			},
			expected: true,
		},
		{
			name: "ConnectionSecret false with name",
			patches: Patches{
				ConnectionSecret:     boolPtr(false),
				ConnectionSecretName: "my-secret",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.patches.HasConnectionSecret())
		})
	}
}

func TestPatches_hasPatches(t *testing.T) {
	tests := []struct {
		name     string
		patches  Patches
		expected bool
	}{
		{
			name:     "no patches set",
			patches:  Patches{},
			expected: false,
		},
		{
			name: "XRD set",
			patches: Patches{
				XRD: "my-xrd.yaml",
			},
			expected: true,
		},
		{
			name: "ConnectionSecret explicitly true",
			patches: Patches{
				ConnectionSecret: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "ConnectionSecret explicitly false",
			patches: Patches{
				ConnectionSecret: boolPtr(false),
			},
			expected: false,
		},
		{
			name: "ConnectionSecret nil",
			patches: Patches{
				ConnectionSecret: nil,
			},
			expected: false,
		},
		{
			name: "ConnectionSecretName set",
			patches: Patches{
				ConnectionSecretName: "my-secret",
			},
			expected: true,
		},
		{
			name: "ConnectionSecretNamespace set",
			patches: Patches{
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
		{
			name: "multiple patches set",
			patches: Patches{
				XRD:                       "my-xrd.yaml",
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
		{
			name: "only name and namespace (no ConnectionSecret flag)",
			patches: Patches{
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.patches.HasPatches())
		})
	}
}

func TestCheckValidTestSuiteFile(t *testing.T) {
	tests := []struct {
		name      string
		spec      *TestSuiteSpec
		wantErr   bool
		errSubstr []string
	}{
		{
			name: "valid test names and IDs",
			spec: &TestSuiteSpec{
				Tests: []TestCase{
					{
						Name: "Test 1",
						ID:   "test1",
						Inputs: Inputs{
							XR: "xr.yaml",
						},
					},
					{
						Name: "Test-2",
						ID:   "test-2",
						Inputs: Inputs{
							Claim: "claim.yaml",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty test name",
			spec: &TestSuiteSpec{
				Tests: []TestCase{
					{
						Name: "",
						Inputs: Inputs{
							XR: "xr.yaml",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: []string{"test case has empty name"},
		},
		{
			name: "invalid test case ID with special chars",
			spec: &TestSuiteSpec{
				Tests: []TestCase{
					{
						Name: "Test 1",
						ID:   "test@1#with$special%chars",
						Inputs: Inputs{
							XR: "xr.yaml",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: []string{"test case ID 'test@1#with$special%chars' contains invalid characters (allowed: alphanumeric, underscore, hyphen)"},
		},
		{
			name: "duplicate test IDs",
			spec: &TestSuiteSpec{
				Tests: []TestCase{
					{
						Name: "Test 1",
						ID:   "test1",
						Inputs: Inputs{
							XR: "xr.yaml",
						},
					},
					{
						Name: "Test 2",
						ID:   "test1", // duplicate ID
						Inputs: Inputs{
							Claim: "claim.yaml",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: []string{"duplicate test case ID 'test1' found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.CheckValidTestSuiteFile()
			if tt.wantErr {
				require.Error(t, err)

				for _, substr := range tt.errSubstr {
					assert.Contains(t, err.Error(), substr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTestSuiteSpec_hasCommonPatches(t *testing.T) {
	tests := []struct {
		name     string
		spec     TestSuiteSpec
		expected bool
	}{
		{
			name: "no patches in common",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs:  Inputs{},
					Patches: Patches{},
				},
			},
			expected: false,
		},
		{
			name: "XRD in common patches",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs: Inputs{},
					Patches: Patches{
						XRD: "common-xrd.yaml",
					},
				},
			},
			expected: true,
		},
		{
			name: "ConnectionSecret in common patches",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs: Inputs{},
					Patches: Patches{
						ConnectionSecret: boolPtr(true),
					},
				},
			},
			expected: true,
		},
		{
			name: "ConnectionSecretName in common patches",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs: Inputs{},
					Patches: Patches{
						ConnectionSecretName: "common-secret",
					},
				},
			},
			expected: true,
		},
		{
			name: "ConnectionSecretNamespace in common patches",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs: Inputs{},
					Patches: Patches{
						ConnectionSecretNamespace: "common-namespace",
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple patches in common",
			spec: TestSuiteSpec{
				Common: Common{
					Inputs: Inputs{},
					Patches: Patches{
						XRD:                       "common-xrd.yaml",
						ConnectionSecret:          boolPtr(true),
						ConnectionSecretName:      "common-secret",
						ConnectionSecretNamespace: "common-namespace",
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.spec.HasCommonPatches())
		})
	}
}

func TestTestCase_hasXR(t *testing.T) {
	tests := []struct {
		name     string
		inputs   Inputs
		expected bool
	}{
		{
			name: "has XR field",
			inputs: Inputs{
				XR: "xr.yaml",
			},
			expected: true,
		},
		{
			name: "empty XR field",
			inputs: Inputs{
				XR: "",
			},
			expected: false,
		},
		{
			name: "only has Claim field",
			inputs: Inputs{
				Claim: "claim.yaml",
			},
			expected: false,
		},
		{
			name: "has both XR and Claim",
			inputs: Inputs{
				XR:    "xr.yaml",
				Claim: "claim.yaml",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCase := TestCase{Inputs: tt.inputs}
			assert.Equal(t, tt.expected, testCase.HasXR())
		})
	}
}

func TestTestCase_hasClaim(t *testing.T) {
	tests := []struct {
		name     string
		inputs   Inputs
		expected bool
	}{
		{
			name: "has Claim field",
			inputs: Inputs{
				Claim: "claim.yaml",
			},
			expected: true,
		},
		{
			name: "empty Claim field",
			inputs: Inputs{
				Claim: "",
			},
			expected: false,
		},
		{
			name: "only has XR field",
			inputs: Inputs{
				XR: "xr.yaml",
			},
			expected: false,
		},
		{
			name: "has both XR and Claim",
			inputs: Inputs{
				XR:    "xr.yaml",
				Claim: "claim.yaml",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCase := TestCase{Inputs: tt.inputs}
			assert.Equal(t, tt.expected, testCase.HasClaim())
		})
	}
}

func TestTestCase_hasPatches(t *testing.T) {
	tests := []struct {
		name     string
		inputs   Inputs
		patches  Patches
		expected bool
	}{
		{
			name: "no patching flags set",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches:  Patches{},
			expected: false,
		},
		{
			name: "XRD flag set",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				XRD: "my-xrd.yaml",
			},
			expected: true,
		},
		{
			name: "connection secret explicitly enabled",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecret: boolPtr(true),
			},
			expected: true,
		},
		{
			name: "connection secret explicitly disabled",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecret: boolPtr(false),
			},
			expected: false,
		},
		{
			name: "connection secret name set",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecretName: "my-secret",
			},
			expected: true,
		},
		{
			name: "connection secret namespace set",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
		{
			name: "multiple patching flags set",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				XRD:                       "my-xrd.yaml",
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
		{
			name: "XRD and connection secret name set",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				XRD:                  "my-xrd.yaml",
				ConnectionSecretName: "my-secret",
			},
			expected: true,
		},
		{
			name: "connection secret disabled but name and namespace set (should still return true)",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			expected: true,
		},
		{
			name: "empty string values should not trigger patching",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				XRD:                       "",
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "",
			},
			expected: false,
		},
		{
			name: "nil connection secret pointer should not trigger patching",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			patches: Patches{
				ConnectionSecret: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCase := TestCase{Inputs: tt.inputs, Patches: tt.patches}
			assert.Equal(t, tt.expected, testCase.HasPatches())
		})
	}
}

func TestTestCase_mergeCommon(t *testing.T) {
	tests := []struct {
		name     string
		testCase TestCase
		common   Common
		expected TestCase
	}{
		{
			name: "test case with empty inputs fields uses common inputs",
			testCase: TestCase{
				Name: "test1",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "",
					Functions:   "",
					CRDs:        []string{},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml", "common-crd2.yaml"},
				},
			},
			expected: TestCase{
				Name: "test1",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml", "common-crd2.yaml"},
				},
			},
		},
		{
			name: "test case with populated inputs fields keeps test case values",
			testCase: TestCase{
				Name: "test2",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "test-composition.yaml",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml"},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml", "common-crd2.yaml"},
				},
			},
			expected: TestCase{
				Name: "test2",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "test-composition.yaml",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml"},
				},
			},
		},
		{
			name: "mixed scenario - some inputs from common, some from test case",
			testCase: TestCase{
				Name: "test3",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "test-composition.yaml",
					Functions:   "",
					CRDs:        []string{},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml"},
				},
			},
			expected: TestCase{
				Name: "test3",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "test-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml"},
				},
			},
		},
		{
			name: "empty common inputs with empty test case fields",
			testCase: TestCase{
				Name: "test4",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "",
					Functions:   "",
					CRDs:        []string{},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "",
					Functions:   "",
					CRDs:        []string{},
				},
			},
			expected: TestCase{
				Name: "test4",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "",
					Functions:   "",
					CRDs:        []string{},
				},
			},
		},
		{
			name: "empty common inputs with populated test case fields",
			testCase: TestCase{
				Name: "test5",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "test-composition.yaml",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml", "test-crd2.yaml"},
				},
			},
			common: Common{},
			expected: TestCase{
				Name: "test5",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "test-composition.yaml",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml", "test-crd2.yaml"},
				},
			},
		},
		{
			name: "test case with empty composition uses common composition",
			testCase: TestCase{
				Name: "test6",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml"},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
					CRDs:        []string{"common-crd1.yaml"},
				},
			},
			expected: TestCase{
				Name: "test6",
				Inputs: Inputs{
					XR:          "xr.yaml",
					Composition: "common-composition.yaml",
					Functions:   "test-functions.yaml",
					CRDs:        []string{"test-crd1.yaml"},
				},
			},
		},
		{
			name: "test case with empty extra fields uses common inputs",
			testCase: TestCase{
				Name: "test7",
				Inputs: Inputs{
					Claim:               "claim.yaml",
					Composition:         "",
					Functions:           "",
					CRDs:                []string{},
					ContextFiles:        map[string]string{},
					ContextValues:       map[string]string{},
					ObservedResources:   "",
					ExtraResources:      "",
					FunctionCredentials: "",
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition:         "common-composition.yaml",
					Functions:           "common-functions.yaml",
					CRDs:                []string{"common-crd1.yaml"},
					ContextFiles:        map[string]string{"file1": "value1"},
					ContextValues:       map[string]string{"key1": "val1"},
					ObservedResources:   "common-observed.yaml",
					ExtraResources:      "common-extra.yaml",
					FunctionCredentials: "common-creds.yaml",
				},
			},
			expected: TestCase{
				Name: "test7",
				Inputs: Inputs{
					Claim:               "claim.yaml",
					Composition:         "common-composition.yaml",
					Functions:           "common-functions.yaml",
					CRDs:                []string{"common-crd1.yaml"},
					ContextFiles:        map[string]string{"file1": "value1"},
					ContextValues:       map[string]string{"key1": "val1"},
					ObservedResources:   "common-observed.yaml",
					ExtraResources:      "common-extra.yaml",
					FunctionCredentials: "common-creds.yaml",
				},
			},
		},
		{
			name: "test case with populated extra fields keeps test case values",
			testCase: TestCase{
				Name: "test8",
				Inputs: Inputs{
					XR:                  "xr.yaml",
					Composition:         "test-composition.yaml",
					Functions:           "test-functions.yaml",
					CRDs:                []string{"test-crd1.yaml"},
					ContextFiles:        map[string]string{"file2": "value2"},
					ContextValues:       map[string]string{"key2": "val2"},
					ObservedResources:   "test-observed.yaml",
					ExtraResources:      "test-extra.yaml",
					FunctionCredentials: "test-creds.yaml",
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition:         "common-composition.yaml",
					Functions:           "common-functions.yaml",
					CRDs:                []string{"common-crd1.yaml"},
					ContextFiles:        map[string]string{"file1": "value1"},
					ContextValues:       map[string]string{"key1": "val1"},
					ObservedResources:   "common-observed.yaml",
					ExtraResources:      "common-extra.yaml",
					FunctionCredentials: "common-creds.yaml",
				},
			},
			expected: TestCase{
				Name: "test8",
				Inputs: Inputs{
					XR:                  "xr.yaml",
					Composition:         "test-composition.yaml",
					Functions:           "test-functions.yaml",
					CRDs:                []string{"test-crd1.yaml"},
					ContextFiles:        map[string]string{"file2": "value2"},
					ContextValues:       map[string]string{"key2": "val2"},
					ObservedResources:   "test-observed.yaml",
					ExtraResources:      "test-extra.yaml",
					FunctionCredentials: "test-creds.yaml",
				},
			},
		},
		{
			name: "test case with empty XRD uses common XRD",
			testCase: TestCase{
				Name: "test9",
				Patches: Patches{
					XRD: "",
				},
			},
			common: Common{
				Patches: Patches{
					XRD: "common-xrd.yaml",
				},
			},
			expected: TestCase{
				Name: "test9",
				Patches: Patches{
					XRD: "common-xrd.yaml",
				},
			},
		},
		{
			name: "test case with populated XRD keeps test case value",
			testCase: TestCase{
				Name: "test10",
				Patches: Patches{
					XRD: "test-xrd.yaml",
				},
			},
			common: Common{
				Patches: Patches{
					XRD: "common-xrd.yaml",
				},
			},
			expected: TestCase{
				Name: "test10",
				Patches: Patches{
					XRD: "test-xrd.yaml",
				},
			},
		},
		{
			name: "test case with connection secret fields",
			testCase: TestCase{
				Name: "test11",
				Inputs: Inputs{
					Claim: "claim.yaml",
				},
				Patches: Patches{
					ConnectionSecret:          nil,
					ConnectionSecretName:      "",
					ConnectionSecretNamespace: "",
				},
			},
			common: Common{
				Patches: Patches{
					ConnectionSecret:          boolPtr(true),
					ConnectionSecretName:      "common-secret",
					ConnectionSecretNamespace: "common-namespace",
				},
			},
			expected: TestCase{
				Name: "test11",
				Inputs: Inputs{
					Claim: "claim.yaml",
				},
				Patches: Patches{
					ConnectionSecret:          boolPtr(true),
					ConnectionSecretName:      "common-secret",
					ConnectionSecretNamespace: "common-namespace",
				},
			},
		},
		{
			name: "test case with no hooks, common with hooks",
			testCase: TestCase{
				Name: "test12",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
			expected: TestCase{
				Name: "test12",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
		},
		{
			name: "test case with pre-test hooks, common with hooks",
			testCase: TestCase{
				Name: "test13",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest: []Hook{{Name: "test-setup", Run: "echo test setup"}},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
			expected: TestCase{
				Name: "test13",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "test-setup", Run: "echo test setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
		},
		{
			name: "test case with post-test hooks, common with hooks",
			testCase: TestCase{
				Name: "test14",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
			expected: TestCase{
				Name: "test14",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
		},
		{
			name: "test case with both hooks, common with hooks",
			testCase: TestCase{
				Name: "test15",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "test-setup", Run: "echo test setup"}},
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "common-setup", Run: "echo common setup"}},
					PostTest: []Hook{{Name: "common-cleanup", Run: "echo common cleanup"}},
				},
			},
			expected: TestCase{
				Name: "test15",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "test-setup", Run: "echo test setup"}},
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
		},
		{
			name: "test case with hooks, common with no hooks",
			testCase: TestCase{
				Name: "test16",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "test-setup", Run: "echo test setup"}},
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Hooks: Hooks{},
			},
			expected: TestCase{
				Name: "test16",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "test-setup", Run: "echo test setup"}},
					PostTest: []Hook{{Name: "test-cleanup", Run: "echo test cleanup"}},
				},
			},
		},
		{
			name: "test case with no assertions, common with assertions",
			testCase: TestCase{
				Name: "test17",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "common-count", Type: "Count", Value: 3},
						{Name: "common-exists", Type: "Exists", Resource: "Deployment/my-app"},
					},
				},
			},
			expected: TestCase{
				Name: "test17",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "common-count", Type: "Count", Value: 3},
						{Name: "common-exists", Type: "Exists", Resource: "Deployment/my-app"},
					},
				},
			},
		},
		{
			name: "test case with assertions, common with assertions",
			testCase: TestCase{
				Name: "test18",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-count", Type: "Count", Value: 5},
					},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "common-count", Type: "Count", Value: 3},
						{Name: "common-exists", Type: "Exists", Resource: "Deployment/my-app"},
					},
				},
			},
			expected: TestCase{
				Name: "test18",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-count", Type: "Count", Value: 5},
					},
				},
			},
		},
		{
			name: "test case with assertions, common with no assertions",
			testCase: TestCase{
				Name: "test19",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-count", Type: "Count", Value: 5},
					},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
			},
			expected: TestCase{
				Name: "test19",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-count", Type: "Count", Value: 5},
					},
				},
			},
		},
		{
			name: "per-engine assertion merge: test has dyff only, common has xprin and diff",
			testCase: TestCase{
				Name: "test20",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Dyff: []AssertionGoldenFile{
						{Name: "test dyff", Expected: "golden.yaml"},
					},
				},
			},
			common: Common{
				Inputs: Inputs{
					Composition: "common-composition.yaml",
					Functions:   "common-functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "common-count", Type: "Count", Value: 2},
					},
					Diff: []AssertionGoldenFile{
						{Name: "common diff", Expected: "common_golden.yaml"},
					},
				},
			},
			expected: TestCase{
				Name: "test20",
				Inputs: Inputs{
					Claim:       "claim.yaml",
					Composition: "composition.yaml",
					Functions:   "functions.yaml",
				},
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "common-count", Type: "Count", Value: 2},
					},
					Diff: []AssertionGoldenFile{
						{Name: "common diff", Expected: "common_golden.yaml"},
					},
					Dyff: []AssertionGoldenFile{
						{Name: "test dyff", Expected: "golden.yaml"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original test case
			testCase := tt.testCase
			testCase.MergeCommon(tt.common)
			assert.Equal(t, tt.expected, testCase)
		})
	}
}

func TestTestCase_checkMandatoryFields(t *testing.T) {
	tests := []struct {
		name    string
		inputs  Inputs
		patches Patches
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid TestCase with Claim field",
			inputs: Inputs{
				Claim:       "claim.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: false,
		},
		{
			name: "valid TestCase with XR field",
			inputs: Inputs{
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: false,
		},
		{
			name: "invalid TestCase - missing both Claim and XR",
			inputs: Inputs{
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: true,
			errMsg:  "missing mandatory field: either 'claim' or 'xr' must be specified",
		},
		{
			name: "invalid TestCase - both Claim and XR specified",
			inputs: Inputs{
				Claim:       "claim.yaml",
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: true,
			errMsg:  "conflicting fields: both 'claim' and 'xr' are specified, but only one is allowed",
		},
		{
			name: "invalid TestCase - missing composition",
			inputs: Inputs{
				Claim:     "claim.yaml",
				XR:        "xr.yaml",
				Functions: "functions.yaml",
			},
			wantErr: true,
			errMsg:  "missing mandatory field: composition",
		},
		{
			name: "invalid TestCase - missing functions",
			inputs: Inputs{
				Claim:       "claim.yaml",
				XR:          "xr.yaml",
				Composition: "composition.yaml",
			},
			wantErr: true,
			errMsg:  "missing mandatory field: functions",
		},
		{
			name:    "invalid TestCase - multiple missing fields",
			inputs:  Inputs{},
			wantErr: true,
			errMsg:  "missing mandatory field: either 'claim' or 'xr' must be specified",
		},
		{
			name: "invalid TestCase with empty strings (should be treated as missing)",
			inputs: Inputs{
				Claim:       "",
				XR:          "",
				Composition: "",
				Functions:   "",
			},
			wantErr: true,
			errMsg:  "missing mandatory field: either 'claim' or 'xr' must be specified",
		},
		{
			name: "valid TestCase with Claim and empty XR",
			inputs: Inputs{
				Claim:       "claim.yaml",
				XR:          "",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: false,
		},
		{
			name: "valid TestCase with XR and empty Claim",
			inputs: Inputs{
				Claim:       "",
				XR:          "xr.yaml",
				Composition: "composition.yaml",
				Functions:   "functions.yaml",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCase := TestCase{Inputs: tt.inputs}

			err := testCase.CheckMandatoryFields()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPatches_checkConnectionSecret(t *testing.T) {
	tests := []struct {
		name        string
		patches     Patches
		wantErr     bool
		errContains string
	}{
		{
			name: "valid - no connection secret fields",
			patches: Patches{
				XRD: "xrd.yaml",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly true with name",
			patches: Patches{
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly true with namespace",
			patches: Patches{
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly true with both name and namespace",
			patches: Patches{
				ConnectionSecret:          boolPtr(true),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
		{
			name: "invalid - ConnectionSecretName without ConnectionSecret=true",
			patches: Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "invalid - ConnectionSecretNamespace without ConnectionSecret=true",
			patches: Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "invalid - both name and namespace without ConnectionSecret=true",
			patches: Patches{
				ConnectionSecret:          nil,
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr:     true,
			errContains: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "valid - ConnectionSecret explicitly false with name (disable)",
			patches: Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly false with namespace (disable)",
			patches: Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
		{
			name: "valid - ConnectionSecret explicitly false with both name and namespace (disable)",
			patches: Patches{
				ConnectionSecret:          boolPtr(false),
				ConnectionSecretName:      "my-secret",
				ConnectionSecretNamespace: "my-namespace",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.patches.CheckConnectionSecret()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHooks_hasPreTestHooks(t *testing.T) {
	tests := []struct {
		name     string
		hooks    Hooks
		expected bool
	}{
		{
			name:     "no hooks",
			hooks:    Hooks{},
			expected: false,
		},
		{
			name: "only post-test hooks",
			hooks: Hooks{
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			hooks: Hooks{
				PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			hooks: Hooks{
				PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: true,
		},
		{
			name: "multiple pre-test hooks",
			hooks: Hooks{
				PreTest: []Hook{
					{Name: "setup1", Run: "echo setup1"},
					{Name: "setup2", Run: "echo setup2"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.hooks.HasPreTestHooks())
		})
	}
}

func TestHooks_hasPostTestHooks(t *testing.T) {
	tests := []struct {
		name     string
		hooks    Hooks
		expected bool
	}{
		{
			name:     "no hooks",
			hooks:    Hooks{},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			hooks: Hooks{
				PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
			},
			expected: false,
		},
		{
			name: "only post-test hooks",
			hooks: Hooks{
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			hooks: Hooks{
				PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: true,
		},
		{
			name: "multiple post-test hooks",
			hooks: Hooks{
				PostTest: []Hook{
					{Name: "cleanup1", Run: "echo cleanup1"},
					{Name: "cleanup2", Run: "echo cleanup2"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.hooks.HasPostTestHooks())
		})
	}
}

func TestHooks_hasHooks(t *testing.T) {
	tests := []struct {
		name     string
		hooks    Hooks
		expected bool
	}{
		{
			name:     "no hooks",
			hooks:    Hooks{},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			hooks: Hooks{
				PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
			},
			expected: true,
		},
		{
			name: "only post-test hooks",
			hooks: Hooks{
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			hooks: Hooks{
				PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
				PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.hooks.HasHooks())
		})
	}
}

func TestTestSuiteSpec_hasCommonHooks(t *testing.T) {
	tests := []struct {
		name     string
		spec     TestSuiteSpec
		expected bool
	}{
		{
			name: "no common hooks",
			spec: TestSuiteSpec{
				Common: Common{
					Hooks: Hooks{},
				},
			},
			expected: false,
		},
		{
			name: "common pre-test hooks",
			spec: TestSuiteSpec{
				Common: Common{
					Hooks: Hooks{
						PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "common post-test hooks",
			spec: TestSuiteSpec{
				Common: Common{
					Hooks: Hooks{
						PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
					},
				},
			},
			expected: true,
		},
		{
			name: "common pre-test and post-test hooks",
			spec: TestSuiteSpec{
				Common: Common{
					Hooks: Hooks{
						PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
						PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.spec.HasCommonHooks())
		})
	}
}

func TestTestCase_hasPreTestHooks(t *testing.T) {
	tests := []struct {
		name     string
		testCase TestCase
		expected bool
	}{
		{
			name: "no hooks",
			testCase: TestCase{
				Hooks: Hooks{},
			},
			expected: false,
		},
		{
			name: "only post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
				},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.testCase.HasPreTestHooks())
		})
	}
}

func TestTestCase_hasPostTestHooks(t *testing.T) {
	tests := []struct {
		name     string
		testCase TestCase
		expected bool
	}{
		{
			name: "no hooks",
			testCase: TestCase{
				Hooks: Hooks{},
			},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
				},
			},
			expected: false,
		},
		{
			name: "only post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.testCase.HasPostTestHooks())
		})
	}
}

func TestTestCase_hasHooks(t *testing.T) {
	tests := []struct {
		name     string
		testCase TestCase
		expected bool
	}{
		{
			name: "no hooks",
			testCase: TestCase{
				Hooks: Hooks{},
			},
			expected: false,
		},
		{
			name: "only pre-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest: []Hook{{Name: "setup", Run: "echo setup"}},
				},
			},
			expected: true,
		},
		{
			name: "only post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: true,
		},
		{
			name: "both pre-test and post-test hooks",
			testCase: TestCase{
				Hooks: Hooks{
					PreTest:  []Hook{{Name: "setup", Run: "echo setup"}},
					PostTest: []Hook{{Name: "cleanup", Run: "echo cleanup"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.testCase.HasHooks())
		})
	}
}

func TestTestCase_hasAssertions(t *testing.T) {
	tests := []struct {
		name     string
		testCase TestCase
		expected bool
	}{
		{
			name: "no assertions",
			testCase: TestCase{
				Assertions: Assertions{Xprin: []AssertionXprin{}},
			},
			expected: false,
		},
		{
			name: "one assertion",
			testCase: TestCase{
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-assertion", Type: "Count", Value: 3},
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple assertions",
			testCase: TestCase{
				Assertions: Assertions{
					Xprin: []AssertionXprin{
						{Name: "test-count", Type: "Count", Value: 3},
						{Name: "test-exists", Type: "Exists", Resource: "Deployment/my-app"},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.testCase.HasAssertions())
		})
	}
}
