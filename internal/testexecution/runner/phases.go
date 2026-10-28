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
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/testexecution/containers"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// The phases of a test case: what runTestCase, and the mode of the command, run one after the other. Each is a
// method of the Runner, and returns the failed result, to end the run of the test case, or nil to go on.
//
// They are the phases of docs/how-it-works.md, where one phase can be several of these:
//
//	Setup:             expandInputPaths, copyInputs, runPreTestHooks (runTestCase makes the temporary directory)
//	Patch and Render:  renderTestCase
//	Validate:          validateTestCase
//	Assert:            runAssertions
//	Finish:            runPostTestHooks, saveOutputs

// expandInputPaths makes the paths of the inputs of the test case (and its XRD patch) absolute, relative to the
// testsuite file, and checks that they exist. It returns the failed result when they don't, to end the run of the
// test case, and nil to go on.
//
//nolint:gocognit // Every path of the test case is expanded and verified the same way, one after the other
func (r *Runner) expandInputPaths(testCase *api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	var err error

	// Always resolve compositionPath, functionPath and all the crdPaths relative to the testsuite file and verify they exist
	// Only resolve Claim or XR path based on which input type is being used
	var (
		failedExpandedPaths []string
		unverifiedPaths     []string
		anyPathExpanded     bool
	)

	// Expand input path based on type (Claim or XR)

	if testCase.HasXR() {
		if !filepath.IsAbs(testCase.Inputs.XR) {
			anyPathExpanded = true
		}

		testCase.Inputs.XR, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.XR)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand XR path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Inputs.XR); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("XR file not found: %v", err))
		}
	} else {
		if !filepath.IsAbs(testCase.Inputs.Claim) {
			anyPathExpanded = true
		}

		testCase.Inputs.Claim, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.Claim)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand Claim path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Inputs.Claim); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("Claim file not found: %v", err))
		}
	}

	if !filepath.IsAbs(testCase.Inputs.Composition) {
		anyPathExpanded = true
	}

	testCase.Inputs.Composition, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.Composition)
	if err != nil {
		failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand composition path: %v", err))
	}

	if err := r.verifyPathExists(testCase.Inputs.Composition); err != nil {
		unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("composition file not found: %v", err))
	}

	if !filepath.IsAbs(testCase.Inputs.Functions) {
		anyPathExpanded = true
	}

	testCase.Inputs.Functions, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.Functions)
	if err != nil {
		failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand functions path: %v", err))
	}

	if err := r.verifyPathExists(testCase.Inputs.Functions); err != nil {
		unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("functions file or dir not found: %v", err))
	}

	for i, originalCRDPath := range testCase.Inputs.CRDs {
		if !filepath.IsAbs(originalCRDPath) {
			anyPathExpanded = true
		}

		testCase.Inputs.CRDs[i], err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, originalCRDPath)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand CRD path %s: %v", originalCRDPath, err))
			continue
		}

		if err := r.verifyPathExists(testCase.Inputs.CRDs[i]); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("crd file not found: %v", err))
			continue
		}
	}

	for key, originalContextFilePath := range testCase.Inputs.ContextFiles {
		if !filepath.IsAbs(originalContextFilePath) {
			anyPathExpanded = true
		}

		testCase.Inputs.ContextFiles[key], err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, originalContextFilePath)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand context file path for key '%s': %v", key, err))
			continue
		}

		if err := r.verifyPathExists(testCase.Inputs.ContextFiles[key]); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("context file not found for key '%s': %v", key, err))
			continue
		}
	}

	if testCase.Inputs.ObservedResources != "" {
		if !filepath.IsAbs(testCase.Inputs.ObservedResources) {
			anyPathExpanded = true
		}

		testCase.Inputs.ObservedResources, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.ObservedResources)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand observed resources path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Inputs.ObservedResources); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("observed resources file or dir not found: %v", err))
		}
	}

	if testCase.Inputs.ExtraResources != "" {
		if !filepath.IsAbs(testCase.Inputs.ExtraResources) {
			anyPathExpanded = true
		}

		testCase.Inputs.ExtraResources, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.ExtraResources)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand extra resources path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Inputs.ExtraResources); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("extra resources file or dir not found: %v", err))
		}
	}

	if testCase.Inputs.FunctionCredentials != "" {
		if !filepath.IsAbs(testCase.Inputs.FunctionCredentials) {
			anyPathExpanded = true
		}

		testCase.Inputs.FunctionCredentials, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Inputs.FunctionCredentials)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand function credentials path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Inputs.FunctionCredentials); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("function credentials file or dir not found: %v", err))
		}
	}

	if testCase.Patches.XRD != "" {
		if !filepath.IsAbs(testCase.Patches.XRD) {
			anyPathExpanded = true
		}

		testCase.Patches.XRD, err = r.expandPathRelativeToTestSuiteFile(r.testSuite.File, testCase.Patches.XRD)
		if err != nil {
			failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand XRD path: %v", err))
		}

		if err := r.verifyPathExists(testCase.Patches.XRD); err != nil {
			unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("XRD file or dir not found: %v", err))
		}
	}

	// Throw combined error if any paths failed to expand or verify
	if len(failedExpandedPaths) > 0 || len(unverifiedPaths) > 0 {
		return result.Fail(fmt.Errorf("failed to expand or verify paths: %s\n\t%s", strings.Join(failedExpandedPaths, "\n\t"), strings.Join(unverifiedPaths, "\n\t")))
	}

	if r.Debug && anyPathExpanded {
		r.debugPrintTestCase(*testCase, "Test specification with expanded input paths:")
	}

	return nil
}

// copyInputs copies the inputs of the test case to the inputs directory of its environment, and points the test
// case to the copies. It returns the failed result when one can't be copied, to end the run of the test case, and
// nil to go on.
func (r *Runner) copyInputs(env *testCaseEnv, testCase *api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	var err error

	// Copy all inputs to the temporary inputs directory
	if testCase.HasXR() {
		testCase.Inputs.XR, err = r.copyInput(env, testCase.Inputs.XR, "xr")
		if err != nil {
			return result.Fail(err)
		}
	} else {
		testCase.Inputs.Claim, err = r.copyInput(env, testCase.Inputs.Claim, "claim")
		if err != nil {
			return result.Fail(err)
		}
	}

	testCase.Inputs.Composition, err = r.copyInput(env, testCase.Inputs.Composition, "composition")
	if err != nil {
		return result.Fail(err)
	}

	testCase.Inputs.Functions, err = r.copyInput(env, testCase.Inputs.Functions, "functions")
	if err != nil {
		return result.Fail(err)
	}

	// ApplyReuse does nothing when container reuse is disabled (r.Containers is nil)
	if err := containers.ApplyReuse(r.fs, r.Containers, testCase.Inputs.Functions); err != nil {
		return result.Fail(fmt.Errorf("failed to apply container reuse: %w", err))
	}

	crdsDir := filepath.Join(env.inputsDir, "crds")

	uniqueNames := uniqueBaseNamesForPaths(testCase.Inputs.CRDs)
	for i, crdPath := range testCase.Inputs.CRDs {
		dest := filepath.Join(crdsDir, uniqueNames[i])

		testCase.Inputs.CRDs[i], err = r.copyToPath(crdPath, dest, "crds")
		if err != nil {
			return result.Fail(err)
		}
	}

	for key, contextFile := range testCase.Inputs.ContextFiles {
		testCase.Inputs.ContextFiles[key], err = r.copyInput(env, contextFile, "context-files")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.ObservedResources != "" {
		testCase.Inputs.ObservedResources, err = r.copyInput(env, testCase.Inputs.ObservedResources, "observed-resources")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.ExtraResources != "" {
		testCase.Inputs.ExtraResources, err = r.copyInput(env, testCase.Inputs.ExtraResources, "extra-resources")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.FunctionCredentials != "" {
		testCase.Inputs.FunctionCredentials, err = r.copyInput(env, testCase.Inputs.FunctionCredentials, "function-credentials")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Patches.XRD != "" {
		testCase.Patches.XRD, err = r.copyInput(env, testCase.Patches.XRD, "xrd")
		if err != nil {
			return result.Fail(err)
		}
	}

	return nil
}

// runPreTestHooks runs the pre-test hooks of the test case, if it has any. It returns the failed result when one
// fails, to end the run of the test case, and nil to go on.
func (r *Runner) runPreTestHooks(testCase api.TestCase, needsResults map[string]*engine.TestCaseResult, result *engine.TestCaseResult) *engine.TestCaseResult {
	if testCase.HasPreTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		var err error

		result.PreTestHooksResults, err = hookExecutor.executeHooks(testCase.Hooks.PreTest, "pre-test", testCase.Inputs, nil, needsResults)
		result.ProcessPreTestHooksOutput()

		if err != nil {
			return result.Fail(nil)
		}
	}

	return nil
}

// renderTestCase turns the test case into the XR to render (converting its Claim, or taking its XR, and patching it),
// renders it, and writes the rendered resources to the outputs directory of its environment. It returns the
// failed result when any of that fails, to end the run of the test case, and nil to go on.
//
//nolint:gocognit // The steps of the conversion, the patching and the render, one after the other
func (r *Runner) renderTestCase(env *testCaseEnv, testCase api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	var err error

	// Handle XR input - either convert Claim to XR or use provided XR file
	var inputXR string
	if testCase.HasXR() {
		// Use provided XR file directly
		inputXR = testCase.Inputs.XR
		if r.Debug {
			utils.DebugPrintf("Using provided XR file: %s\n", inputXR)
		}
	} else {
		// Convert Claim to XR, reusing a previously-generated XR name from an existing full-render
		// golden file if the test case has one, so the conversion reproduces that render.
		goldenPath := ""

		if rel := firstFullRenderGolden(testCase.Assertions); rel != "" {
			if expanded, expandErr := r.expandPathRelativeToTestSuiteFile(r.testSuite.File, rel); expandErr == nil {
				goldenPath = expanded
			}
		}

		inputXR, err = r.convertClaimToXRFunc(r, testCase.Inputs.Claim, testCase.Patches.XRD, goldenPath, env.inputsDir)
		if err != nil {
			return result.Fail(fmt.Errorf("failed to convert Claim: %w", err))
		}
	}

	// Patch XR if needed (XRD and/or connection secret).
	// On v2/v2legacy, XRD defaults are applied natively by render's --xrd flag (added to renderArgs
	// below), so skip the Go-based pre-render patch for XRD on those tiers. v1 always uses the Go
	// patch because crossplane render does not support --xrd.
	if testCase.HasPatches() {
		patchesToApply := testCase.Patches
		if r.XPCLI.SupportsRenderXRD() {
			patchesToApply.XRD = ""
		}
		// HasPatches() is re-evaluated on the modified copy: if XRD was the only patch and we just
		// cleared it, we skip patchXR entirely. Connection-secret fields (including invalid combos)
		// still pass through so CheckConnectionSecret() inside patchXR can return a proper error.
		if patchesToApply.HasPatches() {
			inputXR, err = r.patchXRFunc(r, inputXR, env.inputsDir, patchesToApply)
			if err != nil {
				return result.Fail(fmt.Errorf("failed to patch XR: %w", err))
			}
		}
	}

	renderArgs := make([]string, 0, len(r.Render)+4)

	renderArgs = append(renderArgs, r.Render...)
	if r.CrossplaneVersion != "" && r.XPCLI.SupportsRenderVersion() {
		renderArgs = append(renderArgs, "--crossplane-version="+r.CrossplaneVersion)
	}

	if r.Containers != nil && r.XPCLI.SupportsRenderVersion() {
		// crossplane's dockerized render engine otherwise creates a new Docker network per
		// render call, stranding a reused container on an earlier call's network — see
		// Coordinator.NetworkName.
		networkName, err := r.Containers.NetworkName()
		if err != nil {
			return result.Fail(fmt.Errorf("failed to create docker network for container reuse: %w", err))
		}

		renderArgs = append(renderArgs, "--crossplane-docker-network="+networkName)
	}

	renderArgs = append(renderArgs, inputXR, testCase.Inputs.Composition, testCase.Inputs.Functions)

	if testCase.Patches.XRD != "" && r.XPCLI.SupportsRenderXRD() {
		renderArgs = append(renderArgs, "--xrd="+testCase.Patches.XRD)
	}

	// Add context files if specified (map[string]string)
	for key, contextFile := range testCase.Inputs.ContextFiles {
		renderArgs = append(renderArgs, "--context-files", fmt.Sprintf("%s=%s", key, contextFile))
	}

	// Add context values if specified (map[string]string)
	for key, contextValue := range testCase.Inputs.ContextValues {
		renderArgs = append(renderArgs, "--context-values", fmt.Sprintf("%s=%s", key, contextValue))
	}

	// Add observed resources if specified (single string)
	if testCase.Inputs.ObservedResources != "" {
		renderArgs = append(renderArgs, "--observed-resources", testCase.Inputs.ObservedResources)
	}

	// Add extra resources if specified (single string)
	if testCase.Inputs.ExtraResources != "" {
		renderArgs = append(renderArgs, "--extra-resources", testCase.Inputs.ExtraResources)
	}

	// Add function credentials if specified (single string)
	if testCase.Inputs.FunctionCredentials != "" {
		renderArgs = append(renderArgs, "--function-credentials", testCase.Inputs.FunctionCredentials)
	}

	// Run crossplane render command
	if r.Debug {
		utils.DebugPrintf("Running render command: %s %s\n", r.Dependencies["crossplane"], strings.Join(renderArgs, " "))
	}

	result.RawRenderOutput, err = r.runCommand(r.Dependencies["crossplane"], renderArgs...)
	if err != nil {
		return result.FailRender()
	}

	// Write rendered output to the outputs directory
	result.Outputs.Render = filepath.Join(env.outputsDir, "rendered.yaml")
	if err := afero.WriteFile(r.fs, result.Outputs.Render, result.RawRenderOutput, 0o600); err != nil {
		return result.Fail(fmt.Errorf("failed to write rendered output to temporary file: %w", err))
	}

	if r.Debug {
		utils.DebugPrintf("Wrote rendered output to: %s\n", result.Outputs.Render)
	}

	// Process render output - this sets RenderedResources and FormattedRenderOutput
	if err := result.ProcessRenderOutput(result.RawRenderOutput); err != nil {
		return result.Fail(fmt.Errorf("failed to process render output: %w", err))
	}

	result.Outputs.RenderCount = len(result.RenderedResources)

	if len(result.RenderedResources) > 0 {
		// Create separate XR file with just the first resource
		result.Outputs.XR = filepath.Join(env.outputsDir, "xr.yaml")

		xrYAML, err := yaml.Marshal(result.RenderedResources[0])
		if err != nil {
			return result.Fail(fmt.Errorf("failed to marshal XR resource: %w", err))
		}

		if err := afero.WriteFile(r.fs, result.Outputs.XR, xrYAML, 0o600); err != nil {
			return result.Fail(fmt.Errorf("failed to write XR file: %w", err))
		}
	}

	// Process all resources for Rendered map (including XR)
	for i, resource := range result.RenderedResources {
		kind := resource.GetKind()
		name := resource.GetName()

		// Create filename: rendered-{kind}-{name}.yaml
		filename := fmt.Sprintf("rendered-%s-%s.yaml", strings.ToLower(kind), name)
		filepath := filepath.Join(env.outputsDir, filename)

		// Marshal and write
		resourceYAML, err := yaml.Marshal(resource)
		if err != nil {
			return result.Fail(fmt.Errorf("failed to marshal rendered resource %d: %w", i+1, err))
		}

		if err := afero.WriteFile(r.fs, filepath, resourceYAML, 0o600); err != nil {
			return result.Fail(fmt.Errorf("failed to write rendered resource %d: %w", i+1, err))
		}

		// Add to Rendered map with string key containing slash
		result.Outputs.Rendered[fmt.Sprintf("%s/%s", kind, name)] = filepath
	}

	return nil
}

// validateTestCase validates the rendered resources of the test case against its CRDs, if it has any, and writes
// the output to the outputs directory of its environment. A failed validation is in the result, and does not
// end the run of the test case: only a failure to write the output does, and the failed result is returned then.
// It returns nil to go on.
func (r *Runner) validateTestCase(env *testCaseEnv, testCase api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	var err error

	if len(testCase.Inputs.CRDs) >= 1 {
		validateArgs := make([]string, 0, len(r.Validate)+4)

		validateArgs = append(validateArgs, r.Validate...)
		if r.CrossplaneVersion != "" {
			validateArgs = append(validateArgs, "--crossplane-image=xpkg.crossplane.io/crossplane/crossplane:"+r.CrossplaneVersion)
		}

		validateArgs = append(validateArgs, filepath.Join(env.inputsDir, "crds"), result.Outputs.Render)
		// Run crossplane beta validate command
		if r.Debug {
			utils.DebugPrintf("Running validate command: %s %s\n", r.Dependencies["crossplane"], strings.Join(validateArgs, " "))
		}

		result.RawValidateOutput, err = r.runCommand(r.Dependencies["crossplane"], validateArgs...)
		if err != nil {
			_ = result.MarkValidateFailed()
		}

		result.ProcessValidateOutput()

		// Write validation output to the outputs directory
		validateOutputFile := filepath.Join(env.outputsDir, "validate.txt")
		if err := afero.WriteFile(r.fs, validateOutputFile, result.RawValidateOutput, 0o600); err != nil {
			return result.Fail(fmt.Errorf("failed to write validation output to file: %w", err))
		}

		result.Outputs.Validate = &validateOutputFile

		if r.Debug {
			utils.DebugPrintf("Wrote validation output to: %s\n", validateOutputFile)
		}
	} else { //nolint:gocritic // keep the else block for visibility
		if r.Debug {
			utils.DebugPrintf("Skipped validate command \"%s %s\" because no CRDs were specified\n", r.Dependencies["crossplane"], strings.Join(r.Validate, " "))
		}
	}

	return nil
}

// runAssertions runs the assertions of the test case, if it has any, and writes the output to the outputs
// directory of its environment. A failed assertion is in the result, and does not end the run of the test case:
// only a failure to write the output does, and the failed result is returned then. It returns nil to go on.
func (r *Runner) runAssertions(env *testCaseEnv, testCase api.TestCase, result *engine.TestCaseResult) *engine.TestCaseResult {
	// Execute assertions if any are defined (collect errors but don't fail immediately)
	if testCase.HasAssertions() {
		exec := newAssertionExecutor(
			r.fs,
			&result.Outputs,
			r.Debug,
			r.testSuite.File,
			r.expandPathRelativeToTestSuiteFile,
			r.Color,
		)

		result.AssertionsResults = nil

		if testCase.HasAssertionsXprin() {
			if r.Debug {
				utils.DebugPrintf("Executing %d xprin assertions for test case '%s'\n", len(testCase.Assertions.Xprin), testCase.Name)
			}

			result.AssertionsResults = append(result.AssertionsResults, exec.executeAssertionsXprin(testCase.Assertions.Xprin)...)
		}

		if testCase.HasAssertionsDiff() {
			if r.Debug {
				utils.DebugPrintf("Executing %d diff assertions for test case '%s'\n", len(testCase.Assertions.Diff), testCase.Name)
			}

			result.AssertionsResults = append(result.AssertionsResults, exec.executeAssertionsDiff(testCase.Assertions.Diff)...)
		}

		if testCase.HasAssertionsDyff() {
			if r.Debug {
				utils.DebugPrintf("Executing %d dyff assertions for test case '%s'\n", len(testCase.Assertions.Dyff), testCase.Name)
			}

			result.AssertionsResults = append(result.AssertionsResults, exec.executeAssertionsDyff(testCase.Assertions.Dyff)...)
		}

		// Format assertions output and set hasFailedAssertions
		result.ProcessAssertionsOutput()

		if result.HasFailedAssertions {
			_ = result.MarkAssertionsFailed()
		}

		if r.Debug {
			utils.DebugPrintf("Assertions executed\n")
		}

		// Write raw assertion results to assertions.txt (raw == all assertions, regardless of the Verbose or ShowAssertions flags)
		assertionsFile := filepath.Join(env.outputsDir, "assertions.txt")
		if err := afero.WriteFile(r.fs, assertionsFile, []byte(result.RawAssertionsOutput), 0o600); err != nil {
			return result.Fail(fmt.Errorf("failed to write assertions output to file: %w", err))
		}

		result.Outputs.Assertions = &assertionsFile

		if r.Debug {
			utils.DebugPrintf("Wrote assertions output to: %s\n", assertionsFile)
		}
	}

	return nil
}

// runPostTestHooks runs the post-test hooks of the test case, if it has any, after its assertions. A failed hook
// does not end the run of the test case: it is in the result.
func (r *Runner) runPostTestHooks(testCase api.TestCase, needsResults map[string]*engine.TestCaseResult, result *engine.TestCaseResult) {
	if testCase.HasPostTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		result.PostTestHooksResults, _ = hookExecutor.executeHooks(testCase.Hooks.PostTest, "post-test", testCase.Inputs, &result.Outputs, needsResults)
		result.ProcessPostTestHooksOutput()
		// On post-test hook failure, section shows failed hooks; HasPipelineFailure() is true from results
	}
}

// saveOutputs saves the outputs of a test case that has an ID as internal artifacts, and points its Outputs
// there, so that the test cases that use them (`.Tests`) find them after the temporary
// directory of this one is removed.
func (r *Runner) saveOutputs(testCase api.TestCase, env *testCaseEnv, result *engine.TestCaseResult) error {
	if testCase.ID == "" {
		return nil
	}

	testCaseDir := filepath.Join(r.testSuiteSavedOutputsDir, testCase.ID)
	if err := r.copy(env.outputsDir, testCaseDir); err != nil {
		return fmt.Errorf("failed to save outputs as internal artifacts: %w", err)
	}

	if r.Debug {
		utils.DebugPrintf("Saved outputs as internal artifacts: %s\n", testCaseDir)
	}

	// Update Outputs paths to point to the internal artifacts for cross-test references
	result.Outputs.Render = filepath.Join(testCaseDir, "rendered.yaml")

	result.Outputs.XR = filepath.Join(testCaseDir, "xr.yaml")
	if result.Outputs.Validate != nil {
		*result.Outputs.Validate = filepath.Join(testCaseDir, "validate.txt")
	}

	if result.Outputs.Assertions != nil {
		*result.Outputs.Assertions = filepath.Join(testCaseDir, "assertions.txt")
	}

	// Update Rendered map paths to point to the internal artifacts
	for key, path := range result.Outputs.Rendered {
		filename := filepath.Base(path)
		result.Outputs.Rendered[key] = filepath.Join(testCaseDir, filename)
	}

	return nil
}
