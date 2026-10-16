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
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
)

// validateFailureCasesRe matches the summary line of crossplane validate, e.g. "Total 1 resources: 0 missing schemas, 0 success cases, 1 failure cases".
var validateFailureCasesRe = regexp.MustCompile(`(\d+) failure cases`)

// runExpectInvalidTestCase validates the input XR directly against the CRDs (no render).
// The test passes only if the validation reports at least one schema failure and contains all expected errors.
func (r *Runner) runExpectInvalidTestCase(testCase api.TestCase, env *testCaseEnv, result *engine.TestCaseResult, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	var (
		err      error
		pathErrs []string
	)

	testCase.Inputs.XR, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.XR)
	if err != nil {
		pathErrs = append(pathErrs, fmt.Sprintf("failed to expand XR path: %v", err))
	} else if err := r.verifyPathExists(testCase.Inputs.XR); err != nil {
		pathErrs = append(pathErrs, fmt.Sprintf("XR file not found: %v", err))
	}

	for i, originalCRDPath := range testCase.Inputs.CRDs {
		testCase.Inputs.CRDs[i], err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, originalCRDPath)
		if err != nil {
			pathErrs = append(pathErrs, fmt.Sprintf("failed to expand CRD path %s: %v", originalCRDPath, err))
			continue
		}

		if err := r.verifyPathExists(testCase.Inputs.CRDs[i]); err != nil {
			pathErrs = append(pathErrs, fmt.Sprintf("crd file not found: %v", err))
		}
	}

	if len(pathErrs) > 0 {
		return result.Fail(fmt.Errorf("failed to expand or verify paths:\n\t%s", strings.Join(pathErrs, "\n\t")))
	}

	testCase.Inputs.XR, err = r.copyInput(env, testCase.Inputs.XR, "xr")
	if err != nil {
		return result.Fail(err)
	}

	crdsDir := filepath.Join(env.inputsDir, "crds")

	uniqueNames := uniqueBaseNamesForPaths(testCase.Inputs.CRDs)
	for i, crdPath := range testCase.Inputs.CRDs {
		testCase.Inputs.CRDs[i], err = r.copyToPath(crdPath, filepath.Join(crdsDir, uniqueNames[i]))
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.HasPreTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		result.PreTestHooksResults, err = hookExecutor.executeHooks(testCase.Hooks.PreTest, "pre-test", testCase.Inputs, nil, testSuiteResult.GetCompletedTests())
		result.ProcessPreTestHooksOutput()

		if err != nil {
			return result.Fail(nil)
		}
	}

	validateArgs := r.buildValidateArgs(crdsDir, testCase.Inputs.XR)
	if r.Debug {
		utils.DebugPrintf("Running validate command (expecting failure): %s %s\n", r.Dependencies["crossplane"], strings.Join(validateArgs, " "))
	}

	var validateErr error

	result.RawValidateOutput, validateErr = r.runCommand(r.Dependencies["crossplane"], validateArgs...)

	expectErr := checkExpectInvalid(validateErr, result.RawValidateOutput, testCase.ExpectInvalid.Errors)
	if expectErr != nil {
		result.HasFailedValidate = true
	}

	result.ProcessValidateOutput()

	validateOutputFile := filepath.Join(env.outputsDir, "validate.txt")
	if err := afero.WriteFile(r.fs, validateOutputFile, result.RawValidateOutput, 0o600); err != nil {
		return result.Fail(fmt.Errorf("failed to write validation output to file: %w", err))
	}

	result.Outputs.Validate = &validateOutputFile

	if testCase.HasPostTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		result.PostTestHooksResults, _ = hookExecutor.executeHooks(testCase.Hooks.PostTest, "post-test", testCase.Inputs, &result.Outputs, testSuiteResult.GetCompletedTests())
		result.ProcessPostTestHooksOutput()
	}

	if testCase.ID != "" {
		artifactsDir := filepath.Join(r.testSuiteArtifactsDir, testCase.ID)
		if err := r.copy(env.outputsDir, artifactsDir); err != nil {
			return result.Fail(fmt.Errorf("failed to copy outputs to testsuite artifacts directory: %w", err))
		}

		*result.Outputs.Validate = filepath.Join(artifactsDir, "validate.txt")
	}

	if expectErr != nil {
		return result.Fail(expectErr)
	}

	if result.HasPipelineFailure() {
		return result.Fail(nil)
	}

	return result.Complete()
}

// checkExpectInvalid returns an error unless the validate command failed because of at least one schema failure
// and its output contains all the expected errors.
func checkExpectInvalid(validateErr error, output []byte, expectedErrors []string) error {
	if validateErr == nil {
		return fmt.Errorf("expected the XR to be invalid, but validation succeeded")
	}

	// A non-zero exit can also come from missing schemas or a CLI error, which must not count as an invalid XR.
	match := validateFailureCasesRe.FindSubmatch(output)
	if match == nil {
		return fmt.Errorf("expected the XR to be invalid, but validation failed without reporting any failure case")
	}

	if failures, _ := strconv.Atoi(string(match[1])); failures == 0 {
		return fmt.Errorf("expected the XR to be invalid, but validation reported 0 failure cases")
	}

	var missing []string

	for _, expected := range expectedErrors {
		if !strings.Contains(string(output), expected) {
			missing = append(missing, fmt.Sprintf("expected error not found in validate output: %q", expected))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%s", strings.Join(missing, "\n"))
	}

	return nil
}
