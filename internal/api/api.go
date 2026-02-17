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

// Package api provides the API type definitions and validation methods for test suite specifications.
package api

import (
	"fmt"
	"maps"
	"strings"
)

// TestSuiteSpec represents the structure of a testsuite YAML file used by xprin.
type TestSuiteSpec struct {
	Common Common     `json:"common,omitempty"` // Common config for all tests (Optional)
	Tests  []TestCase `json:"tests"`            // List of test cases (Required)
}

// Patches represents XR patching configuration.
type Patches struct {
	XRD                       string `json:"xrd,omitempty"`                         // Path to the XR's or Claim's XRD (Optional)
	ConnectionSecret          *bool  `json:"connection-secret,omitempty"`           // When true, create a connection secret for the XR (Optional)
	ConnectionSecretName      string `json:"connection-secret-name,omitempty"`      // Name of the connection secret (Optional)
	ConnectionSecretNamespace string `json:"connection-secret-namespace,omitempty"` // Namespace of the connection secret (Optional)
}

// Hooks represents the execution hooks configuration.
type Hooks struct {
	PreTest  []Hook `json:"pre-test,omitempty"`  // Hooks that are executed before the testcase (Optional)
	PostTest []Hook `json:"post-test,omitempty"` // Hooks that are executed after the testcase (Optional)
}

// Hook represents a single executable step with optional metadata.
type Hook struct {
	Name string `json:"name,omitempty"` // Descriptive name for the hook (Optional)
	Run  string `json:"run"`            // Command to run (Required)
}

// AssertionXprin represents a single xprin assertion (single-resource or Count).
type AssertionXprin struct {
	Name     string `json:"name"`                                                                                                                                      // Descriptive name for the assertion (Required)
	Type     string `json:"type"               jsonschema:"enum=Count,enum=Exists,enum=NotExists,enum=FieldType,enum=FieldExists,enum=FieldNotExists,enum=FieldValue"` // Type of assertion (Required)
	Resource string `json:"resource,omitempty"`                                                                                                                        // Resource identifier for resource-based assertions (format: Kind/Name e.g. "Cluster/platform-aws-rds") (Optional)
	Field    string `json:"field,omitempty"`                                                                                                                           // Field path for field-based assertions (e.g., "metadata.name") (Optional)
	Operator string `json:"operator,omitempty" jsonschema:"enum===,enum=is"`                                                                                           // Operator for field value assertions (== or is) (Optional)
	Value    any    `json:"value,omitempty"`                                                                                                                           // Expected value for the assertion (Optional)
}

// AssertionGoldenFile represents a single golden-file assertion (compare actual output to expected file; used by diff and dyff).
type AssertionGoldenFile struct {
	Name     string `json:"name"`               // Descriptive name for the assertion (Required)
	Expected string `json:"expected"`           // Path to golden (expected) file (Required)
	Resource string `json:"resource,omitempty"` // Resource identifier for resource-based assertions (format: Kind/Name e.g. "Cluster/platform-aws-rds") (Optional)
}

// Assertions represents assertions grouped by execution engine.
type Assertions struct {
	Xprin []AssertionXprin      `json:"xprin,omitempty"` // xprin assertions (in-process) (Optional)
	Diff  []AssertionGoldenFile `json:"diff,omitempty"`  // diff assertions (go-native compare to golden file) (Optional)
	Dyff  []AssertionGoldenFile `json:"dyff,omitempty"`  // dyff assertions (dyff between expected and actual) (Optional)
}

// Common represents the common configuration for a testsuite file.
type Common struct {
	Inputs     Inputs     `json:"inputs,omitempty"`     // Common inputs (composition, Claim/XR, etc.) for all testcases (Optional)
	Patches    Patches    `json:"patches,omitempty"`    // Common XR patching configuration for all testcases (Optional)
	Hooks      Hooks      `json:"hooks,omitempty"`      // Common hooks for all testcases (Optional)
	Assertions Assertions `json:"assertions,omitempty"` // Common assertions to validate rendered resources for all testcases (Optional)
}

// TestCase represents a single test case.
type TestCase struct {
	Name       string     `json:"name"`                 // Descriptive name for the testcase (Required)
	ID         string     `json:"id,omitempty"`         // Unique identifier for the testcase (Optional)
	Inputs     Inputs     `json:"inputs,omitempty"`     // Inputs of a testcase (Required unless specified in the common inputs)
	Patches    Patches    `json:"patches,omitempty"`    // XR patching configuration (Optional)
	Hooks      Hooks      `json:"hooks,omitempty"`      // Execution hooks (Optional)
	Assertions Assertions `json:"assertions,omitempty"` // Assertions to validate rendered resources (Optional)
}

// Inputs represents the inputs for a test case or common configuration.
type Inputs struct {
	Claim               string            `json:"claim,omitempty"`                // Path to Claim file (one of Claim or XR must be set, either in the test case or in the common inputs)
	XR                  string            `json:"xr,omitempty"`                   // Path to XR file (one of Claim or XR must be set, either in the test case or in the common inputs)
	Composition         string            `json:"composition,omitempty"`          // Path to composition file (Required unless specified in the common inputs)
	Functions           string            `json:"functions,omitempty"`            // Path to functions file or directory (Required unless specified in the common inputs)
	CRDs                []string          `json:"crds,omitempty"`                 // Paths to CRD files (Optional)
	ContextFiles        map[string]string `json:"context-files,omitempty"`        // Map of context keys to file paths (Optional)
	ContextValues       map[string]string `json:"context-values,omitempty"`       // Map of context keys to inline values (Optional)
	ObservedResources   string            `json:"observed-resources,omitempty"`   // Path to observed resources file (Optional)
	ExtraResources      string            `json:"extra-resources,omitempty"`      // Path to extra resources file (Optional)
	FunctionCredentials string            `json:"function-credentials,omitempty"` // Path to function credentials file (Optional)
}

// HasConnectionSecret returns true if ConnectionSecret is explicitly set to true.
func (p *Patches) HasConnectionSecret() bool {
	return p.ConnectionSecret != nil && *p.ConnectionSecret
}

// HasPatches returns true if any patches are set.
func (p *Patches) HasPatches() bool {
	return p.XRD != "" ||
		p.HasConnectionSecret() ||
		p.ConnectionSecretName != "" ||
		p.ConnectionSecretNamespace != ""
}

// CheckConnectionSecret validates connection secret configuration:
// - ConnectionSecret unset && ConnectionSecretName/Namespace set => error
// - ConnectionSecret true && ConnectionSecretName/Namespace set => enable
// - ConnectionSecret false && ConnectionSecretName/Namespace set => disable (no error).
func (p *Patches) CheckConnectionSecret() error {
	// If name or namespace are provided, check connection-secret state
	if p.ConnectionSecretName != "" || p.ConnectionSecretNamespace != "" {
		if p.ConnectionSecret == nil {
			// ConnectionSecret unset && ConnectionSecretName/Namespace set => error
			return fmt.Errorf("connection-secret must be set to true when using connection-secret-name or connection-secret-namespace")
		}
		// ConnectionSecret true => enable (no error)
		// ConnectionSecret false => disable (no error)
	}

	return nil
}

// HasPreTestHooks returns true if any pre-test hooks are set.
func (h *Hooks) HasPreTestHooks() bool {
	return len(h.PreTest) > 0
}

// HasPostTestHooks returns true if any post-test hooks are set.
func (h *Hooks) HasPostTestHooks() bool {
	return len(h.PostTest) > 0
}

// HasHooks returns true if any hooks are set.
func (h *Hooks) HasHooks() bool {
	return h.HasPreTestHooks() || h.HasPostTestHooks()
}

// HasAssertionsXprin returns true if any xprin assertions are set.
func (a *Assertions) HasAssertionsXprin() bool {
	return len(a.Xprin) > 0
}

// HasAssertionsDiff returns true if any diff assertions are set.
func (a *Assertions) HasAssertionsDiff() bool {
	return len(a.Diff) > 0
}

// HasAssertionsDyff returns true if any dyff assertions are set.
func (a *Assertions) HasAssertionsDyff() bool {
	return len(a.Dyff) > 0
}

// HasAssertions returns true if any assertions are set.
func (a *Assertions) HasAssertions() bool {
	return a.HasAssertionsXprin() || a.HasAssertionsDiff() || a.HasAssertionsDyff()
}

// CheckValidTestSuiteFile checks:
// - if test case names are non-empty
// - if test case IDs are unique (only for tests that have IDs)
// and returns a list of all validation errors found.
func (ts *TestSuiteSpec) CheckValidTestSuiteFile() error {
	var allErrors []string

	// Check if an ID contains only alphanumeric characters, underscores, and hyphens
	hasValidID := func(id string) bool {
		if len(id) == 0 {
			return false
		}

		for _, char := range id {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' && char != '-' {
				return false
			}
		}

		return true
	}

	// Track used IDs to detect duplicates
	usedIDs := make(map[string]bool)

	for i := range ts.Tests {
		test := &ts.Tests[i]

		// Check for empty name
		if test.Name == "" {
			allErrors = append(allErrors, "test case has empty name")
		}

		// Only validate and check uniqueness for IDs that are explicitly provided
		if test.ID != "" {
			// Validate test ID format
			if !hasValidID(test.ID) {
				allErrors = append(allErrors, fmt.Sprintf("test case ID '%s' contains invalid characters (allowed: alphanumeric, underscore, hyphen)", test.ID))
			}

			// Check for duplicate IDs (only among tests that have IDs)
			if usedIDs[test.ID] {
				allErrors = append(allErrors, fmt.Sprintf("duplicate test case ID '%s' found", test.ID))
			} else {
				usedIDs[test.ID] = true
			}
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("invalid testsuite file:\n- %s", strings.Join(allErrors, "\n- "))
	}

	return nil
}

// HasCommonPatches returns true if any common patches are set in the test suite.
func (ts *TestSuiteSpec) HasCommonPatches() bool {
	return ts.Common.Patches.HasPatches()
}

// HasCommonHooks returns true if any common hooks are set in the test suite.
func (ts *TestSuiteSpec) HasCommonHooks() bool {
	return ts.Common.Hooks.HasHooks()
}

// HasCommonAssertions returns true if any common assertions are set in the test suite.
func (ts *TestSuiteSpec) HasCommonAssertions() bool {
	return ts.Common.Assertions.HasAssertions()
}

// HasCommon returns true if any common inputs are set in the test suite spec.
func (ts *TestSuiteSpec) HasCommon() bool {
	return ts.Common.Inputs.XR != "" ||
		ts.Common.Inputs.Claim != "" ||
		ts.Common.Inputs.Composition != "" ||
		ts.Common.Inputs.Functions != "" ||
		len(ts.Common.Inputs.CRDs) > 0 ||
		len(ts.Common.Inputs.ContextFiles) > 0 ||
		len(ts.Common.Inputs.ContextValues) > 0 ||
		ts.Common.Inputs.ObservedResources != "" ||
		ts.Common.Inputs.ExtraResources != "" ||
		ts.Common.Inputs.FunctionCredentials != "" ||
		ts.HasCommonPatches() ||
		ts.HasCommonHooks() ||
		ts.HasCommonAssertions()
}

// HasXR returns true if the TestCase has an XR field specified.
func (tc *TestCase) HasXR() bool {
	return tc.Inputs.XR != ""
}

// HasClaim returns true if the TestCase has a Claim field specified.
func (tc *TestCase) HasClaim() bool {
	return tc.Inputs.Claim != ""
}

// HasPatches checks if any patches are set in the test case.
func (tc *TestCase) HasPatches() bool {
	return tc.Patches.HasPatches()
}

// HasPreTestHooks checks if any pre-test hooks are set in the test case.
func (tc *TestCase) HasPreTestHooks() bool {
	return tc.Hooks.HasPreTestHooks()
}

// HasPostTestHooks checks if any post-test hooks are set in the test case.
func (tc *TestCase) HasPostTestHooks() bool {
	return tc.Hooks.HasPostTestHooks()
}

// HasHooks checks if any hooks are set in the test case.
func (tc *TestCase) HasHooks() bool {
	return tc.Hooks.HasHooks()
}

// HasAssertionsXprin checks if any xprin assertions are set in the test case.
func (tc *TestCase) HasAssertionsXprin() bool {
	return tc.Assertions.HasAssertionsXprin()
}

// HasAssertionsDiff checks if any diff assertions are set in the test case.
func (tc *TestCase) HasAssertionsDiff() bool {
	return tc.Assertions.HasAssertionsDiff()
}

// HasAssertionsDyff checks if any dyff assertions are set in the test case.
func (tc *TestCase) HasAssertionsDyff() bool {
	return tc.Assertions.HasAssertionsDyff()
}

// HasAssertions returns true if any assertions are defined.
func (tc *TestCase) HasAssertions() bool {
	return tc.Assertions.HasAssertions()
}

// MergeCommon merges common inputs and patches into the test case.
//
//nolint:gocognit // too many ifs, but not that complex
func (tc *TestCase) MergeCommon(common Common) {
	if tc.Inputs.XR == "" {
		tc.Inputs.XR = common.Inputs.XR
	}

	if tc.Inputs.Claim == "" {
		tc.Inputs.Claim = common.Inputs.Claim
	}

	if tc.Inputs.Composition == "" {
		tc.Inputs.Composition = common.Inputs.Composition
	}

	if tc.Inputs.Functions == "" {
		tc.Inputs.Functions = common.Inputs.Functions
	}

	if len(tc.Inputs.CRDs) == 0 && len(common.Inputs.CRDs) > 0 {
		tc.Inputs.CRDs = make([]string, len(common.Inputs.CRDs))
		copy(tc.Inputs.CRDs, common.Inputs.CRDs)
	}

	if len(tc.Inputs.ContextFiles) == 0 && len(common.Inputs.ContextFiles) > 0 {
		tc.Inputs.ContextFiles = make(map[string]string)
		maps.Copy(tc.Inputs.ContextFiles, common.Inputs.ContextFiles)
	}

	if len(tc.Inputs.ContextValues) == 0 && len(common.Inputs.ContextValues) > 0 {
		tc.Inputs.ContextValues = make(map[string]string)
		maps.Copy(tc.Inputs.ContextValues, common.Inputs.ContextValues)
	}

	if tc.Inputs.ObservedResources == "" {
		tc.Inputs.ObservedResources = common.Inputs.ObservedResources
	}

	if tc.Inputs.ExtraResources == "" {
		tc.Inputs.ExtraResources = common.Inputs.ExtraResources
	}

	if tc.Inputs.FunctionCredentials == "" {
		tc.Inputs.FunctionCredentials = common.Inputs.FunctionCredentials
	}

	// Always merge patches if common has patches
	if common.Patches.HasPatches() {
		if tc.Patches.XRD == "" {
			tc.Patches.XRD = common.Patches.XRD
		}

		if tc.Patches.ConnectionSecret == nil {
			tc.Patches.ConnectionSecret = common.Patches.ConnectionSecret
		}

		if tc.Patches.ConnectionSecretName == "" {
			tc.Patches.ConnectionSecretName = common.Patches.ConnectionSecretName
		}

		if tc.Patches.ConnectionSecretNamespace == "" {
			tc.Patches.ConnectionSecretNamespace = common.Patches.ConnectionSecretNamespace
		}
	}

	// Always merge hooks if common has hooks
	if common.Hooks.HasHooks() {
		if !tc.HasPreTestHooks() {
			tc.Hooks.PreTest = common.Hooks.PreTest
		}

		if !tc.HasPostTestHooks() {
			tc.Hooks.PostTest = common.Hooks.PostTest
		}
	}

	// Merge assertions per engine: if common has assertions for an engine and the test case does not, use common's.
	if common.Assertions.HasAssertionsXprin() && !tc.HasAssertionsXprin() {
		tc.Assertions.Xprin = make([]AssertionXprin, len(common.Assertions.Xprin))
		copy(tc.Assertions.Xprin, common.Assertions.Xprin)
	}

	if common.Assertions.HasAssertionsDiff() && !tc.HasAssertionsDiff() {
		tc.Assertions.Diff = make([]AssertionGoldenFile, len(common.Assertions.Diff))
		copy(tc.Assertions.Diff, common.Assertions.Diff)
	}

	if common.Assertions.HasAssertionsDyff() && !tc.HasAssertionsDyff() {
		tc.Assertions.Dyff = make([]AssertionGoldenFile, len(common.Assertions.Dyff))
		copy(tc.Assertions.Dyff, common.Assertions.Dyff)
	}
}

// CheckMandatoryFields checks if all mandatory fields are present in the test case.
func (tc *TestCase) CheckMandatoryFields() error {
	var allErrors []string

	if tc.HasClaim() && tc.HasXR() {
		allErrors = append(allErrors, "conflicting fields: both 'claim' and 'xr' are specified, but only one is allowed")
	}

	if !tc.HasClaim() && !tc.HasXR() {
		allErrors = append(allErrors, "missing mandatory field: either 'claim' or 'xr' must be specified (it can be specified either in the test case or in the common inputs)")
	}

	if tc.Inputs.Composition == "" {
		allErrors = append(allErrors, "missing mandatory field: composition (it can be specified either in the test case or in the common inputs)")
	}

	if tc.Inputs.Functions == "" {
		allErrors = append(allErrors, "missing mandatory field: functions (it can be specified either in the test case or in the common inputs)")
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(allErrors, "\n    "))
	}

	return nil
}
