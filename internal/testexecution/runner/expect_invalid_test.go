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

package runner

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const (
	validateOutputInvalid = `[x] schema validation error aws.example.com/v1, Kind=XAWSInfrastructure, missing-region : spec.region: Required value
Total 1 resources: 0 missing schemas, 0 success cases, 1 failure cases
crossplane: error: could not validate all resources`
	validateOutputValid = `[✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
Total 1 resources: 0 missing schemas, 1 success cases, 0 failure cases`
	validateOutputMissingSchema = `[!] could not find CRD/XRD for: gcp.example.com/v1, Kind=GCPPlatformClaim
Total 1 resources: 1 missing schemas, 0 success cases, 0 failure cases
crossplane: error: could not validate all resources, schema(s) missing`
)

func TestCheckExpectInvalid(t *testing.T) {
	errExit := errors.New("exit status 1")

	tests := []struct {
		name           string
		validateErr    error
		output         string
		expectedErrors []string
		wantErr        string
	}{
		{
			name:        "invalid XR",
			validateErr: errExit,
			output:      validateOutputInvalid,
		},
		{
			name:           "invalid XR with all expected errors",
			validateErr:    errExit,
			output:         validateOutputInvalid,
			expectedErrors: []string{"spec.region: Required value", "missing-region"},
		},
		{
			name:           "invalid XR with a missing expected error",
			validateErr:    errExit,
			output:         validateOutputInvalid,
			expectedErrors: []string{"spec.region: Required value", "spec.team: Required value"},
			wantErr:        `expected error not found in validate output: "spec.team: Required value"`,
		},
		{
			name:    "valid XR",
			output:  validateOutputValid,
			wantErr: "expected the XR to be invalid, but validation succeeded",
		},
		{
			name:        "missing schema",
			validateErr: errExit,
			output:      validateOutputMissingSchema,
			wantErr:     "validation reported 0 failure cases",
		},
		{
			name:        "command error without summary",
			validateErr: errExit,
			output:      "crossplane: error: unexpected argument",
			wantErr:     "validation failed without reporting any failure case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkExpectInvalid(tt.validateErr, []byte(tt.output), tt.expectedErrors)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRunTestCase_ExpectInvalid(t *testing.T) {
	cfg := &config.Config{Dependencies: map[string]string{config.CrossplaneCmd: config.CrossplaneCmd}}

	tests := []struct {
		name           string
		validateOutput string
		validateErr    error
		expectInvalid  *api.ExpectInvalid
		wantStatus     engine.Status
		wantErr        string
	}{
		{
			name:           "passes when the XR is rejected",
			validateOutput: validateOutputInvalid,
			validateErr:    errors.New("exit status 1"),
			expectInvalid:  &api.ExpectInvalid{Errors: []string{"spec.region: Required value"}},
			wantStatus:     engine.StatusPass(),
		},
		{
			name:           "fails when the XR is accepted",
			validateOutput: validateOutputValid,
			expectInvalid:  &api.ExpectInvalid{},
			wantStatus:     engine.StatusFail(),
			wantErr:        "validation succeeded",
		},
		{
			name:           "fails when an expected error is missing",
			validateOutput: validateOutputInvalid,
			validateErr:    errors.New("exit status 1"),
			expectInvalid:  &api.ExpectInvalid{Errors: []string{"spec.team: Required value"}},
			wantStatus:     engine.StatusFail(),
			wantErr:        "spec.team: Required value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newMockRunner(makeOptions(cfg, nil, nil))
			runner.testSuiteSpec = &api.TestSuiteSpec{}

			var validateCalls [][]string

			runner.runCommand = func(name string, args ...string) ([]byte, error) {
				require.Equal(t, config.CrossplaneCmd, name)
				require.NotEqual(t, config.RenderSubcommand, args[0], "render must not run for expect-invalid test cases")

				validateCalls = append(validateCalls, args)

				return []byte(tt.validateOutput), tt.validateErr
			}

			testCase := api.TestCase{
				Name:          tt.name,
				Inputs:        api.Inputs{XR: "invalid_xr.yaml", CRDs: []string{"xrd.yaml"}},
				ExpectInvalid: tt.expectInvalid,
			}

			result := runner.runTestCase(testCase, 1, engine.NewTestSuiteResult(testSuiteFile, false))

			require.Len(t, validateCalls, 1)
			args := validateCalls[0]
			assert.Equal(t, "crds", filepath.Base(args[len(args)-2]))
			assert.Equal(t, "invalid_xr.yaml", filepath.Base(args[len(args)-1]))
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantStatus == engine.StatusFail(), result.HasFailedValidate)
			require.NotNil(t, result.Outputs.Validate)

			if tt.wantErr == "" {
				require.NoError(t, result.Error)
			} else {
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tt.wantErr)
			}
		})
	}
}
