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

package discovery

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestCheckTestSuiteFile(t *testing.T) {
	tests := []struct {
		name      string
		spec      *api.TestSuiteSpec
		wantErr   bool
		errSubstr []string
	}{
		{
			name: "valid test names and IDs",
			spec: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "Test 1",
						ID:   "test1",
						Inputs: api.Inputs{
							XR: "xr.yaml",
						},
					},
					{
						Name: "Test-2",
						ID:   "test-2",
						Inputs: api.Inputs{
							Claim: "claim.yaml",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty test name",
			spec: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "",
						Inputs: api.Inputs{
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
			spec: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "Test 1",
						ID:   "test@1#with$special%chars",
						Inputs: api.Inputs{
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
			spec: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "Test 1",
						ID:   "test1",
						Inputs: api.Inputs{
							XR: "xr.yaml",
						},
					},
					{
						Name: "Test 2",
						ID:   "test1", // duplicate ID
						Inputs: api.Inputs{
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
			err := checkTestSuiteFile(tt.spec)
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

func TestCheckTestSuiteFile_Needs(t *testing.T) {
	tests := map[string]struct {
		tests   []api.TestCase
		wantErr string
	}{
		"nothing needed": {
			tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b"}},
		},
		"the testcase before": {
			tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", Needs: []string{"a"}}},
		},
		"several, in any order": {
			tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", ID: "b"}, {Name: "c", Needs: []string{"b", "a"}}},
		},
		"a testcase that is needed can need another": {
			tests: []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", ID: "b", Needs: []string{"a"}}, {Name: "c", Needs: []string{"b"}}},
		},
		"an ID that does not exist": {
			tests:   []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", Needs: []string{"nope"}}},
			wantErr: `test case "b" needs "nope", but no test case has that ID`,
		},
		"a testcase listed after it": {
			tests: []api.TestCase{{Name: "a", Needs: []string{"b"}}, {Name: "b", ID: "b"}},
		},
		"itself": {
			tests:   []api.TestCase{{Name: "a", ID: "a", Needs: []string{"a"}}},
			wantErr: "needs cycle: a -> a",
		},
		"each other": {
			tests:   []api.TestCase{{Name: "a", ID: "a", Needs: []string{"b"}}, {Name: "b", ID: "b", Needs: []string{"a"}}},
			wantErr: "needs cycle: a -> b -> a",
		},
		"an empty list": {
			tests:   []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", Needs: []string{}}},
			wantErr: "test case \"b\" has an empty `needs`",
		},
		"an empty ID": {
			tests:   []api.TestCase{{Name: "a", ID: "a"}, {Name: "b", Needs: []string{""}}},
			wantErr: `test case "b" needs "", but no test case has that ID`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := checkTestSuiteFile(&api.TestSuiteSpec{Tests: tc.tests})

			if tc.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}

	t.Run("every problem is reported", func(t *testing.T) {
		err := checkTestSuiteFile(&api.TestSuiteSpec{Tests: []api.TestCase{{Name: "a", Needs: []string{"x", "y"}}}})

		require.ErrorContains(t, err, `needs "x"`)
		require.ErrorContains(t, err, `needs "y"`)
	})
}
