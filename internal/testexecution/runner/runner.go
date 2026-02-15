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
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/gertd/go-pluralize"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// Runner handles test execution.
type Runner struct {
	*testexecutionUtils.Options

	fs               afero.Fs
	testSuiteSpec    *api.TestSuiteSpec
	testSuiteFile    string
	testSuiteFileDir string
	output           io.Writer
	// Directory paths
	inputsDir             string
	outputsDir            string
	testCaseTmpDir        string
	testSuiteArtifactsDir string
	// Mockable function fields
	runTestsFunc                      func() error
	runTestCaseFunc                   func(api.TestCase) *engine.TestCaseResult
	expandPathRelativeToTestSuiteFile func(base, path string) (string, error)
	verifyPathExists                  func(path string) error
	runCommand                        func(name string, args ...string) ([]byte, error)
	copy                              func(src, dest string, opts ...cp.Options) error
	convertClaimToXRFunc              func(r *Runner, claimPath, outputPath string) (string, error)
	patchXRFunc                       func(r *Runner, xrPath, outputPath string, patches api.Patches) (string, error)
}

// templateContext provides variables available in test suite templates.
type templateContext struct {
	// Repository variables (for compatibility with existing {{ .Repositories.name }} syntax)
	Repositories map[string]string // Repository name to path mapping
	// Input variables (available in hooks)
	Inputs api.Inputs
	// Output variables (available in post-test hooks)
	Outputs *engine.Outputs
	// Cross-test references (available in hooks)
	Tests map[string]*engine.TestCaseResult // Test ID to test case result mapping
}

// NewRunner creates a new test runner.
func NewRunner(options *testexecutionUtils.Options, testSuiteFile string, testSuiteSpec *api.TestSuiteSpec) *Runner {
	testSuiteFileDir := filepath.Dir(testSuiteFile)

	return &Runner{
		fs:               afero.NewOsFs(),
		output:           os.Stdout, // Default output to stdout
		Options:          options,
		testSuiteFile:    testSuiteFile,
		testSuiteFileDir: testSuiteFileDir,
		testSuiteSpec:    testSuiteSpec,
		// Initialize mockable function fields with default implementations
		runTestCaseFunc:                   nil, // will set default below
		expandPathRelativeToTestSuiteFile: testexecutionUtils.ExpandPathRelativeToTestSuiteFile,
		verifyPathExists:                  utils.VerifyPathExists,
		runCommand: func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = testSuiteFileDir

			var combined bytes.Buffer

			cmd.Stdout = &combined
			cmd.Stderr = &combined
			err := cmd.Run()

			return combined.Bytes(), err
		},
		copy:                 cp.Copy,
		convertClaimToXRFunc: (*Runner).convertClaimToXR,
		patchXRFunc:          (*Runner).patchXR,
	}
}

// newTemplateContext creates a new template context with the given parameters.
func newTemplateContext(repositories map[string]string, inputs api.Inputs, outputs *engine.Outputs, tests map[string]*engine.TestCaseResult) *templateContext {
	if repositories == nil {
		repositories = make(map[string]string)
	}

	return &templateContext{
		Repositories: repositories,
		Inputs:       inputs,
		Outputs:      outputs,
		Tests:        tests,
	}
}

// RunTests runs all tests in a test suite.
func (r *Runner) RunTests() error {
	if r.runTestsFunc != nil {
		return r.runTestsFunc()
	}

	// Validate that testSuiteFile and testSuiteSpec are set (they should be set in NewRunner)
	if r.testSuiteFile == "" {
		return fmt.Errorf("testsuite file path is required")
	}

	if r.testSuiteSpec == nil {
		return fmt.Errorf("testsuite specification is required")
	}

	// Create testsuite artifacts directory (always created, cleaned up when testsuite finishes)
	var err error

	r.testSuiteArtifactsDir, err = afero.TempDir(r.fs, "", "xprin-testsuite-artifacts-")
	if err != nil {
		return fmt.Errorf("failed to create testsuite artifacts directory: %w", err)
	}

	defer func() {
		_ = r.fs.RemoveAll(r.testSuiteArtifactsDir)
	}()

	if r.Debug {
		utils.DebugPrintf("Created testsuite artifacts directory: %s\n", r.testSuiteArtifactsDir)
	}

	if r.Debug {
		testSuiteFileDir := filepath.Dir(r.testSuiteFile)
		utils.DebugPrintf("Using testsuite file directory for relative path resolution: %s\n", testSuiteFileDir)

		// We do not expand or verify common fields here. They will be expanded and verified by the individual test cases that do not override them.
		if r.testSuiteSpec.HasCommon() {
			r.debugPrintCommon(r.testSuiteSpec.Common, "Found common configuration:")
		}

		plural := pluralize.NewClient()
		utils.DebugPrintf("Found %s\n", plural.Pluralize("test case", len(r.testSuiteSpec.Tests), true))
	}

	// Create test suite result
	testSuiteResult := engine.NewTestSuiteResult(r.testSuiteFile, r.Verbose)

	// Loop through all test cases and run them directly
	for _, testCase := range r.testSuiteSpec.Tests {
		// Run the test and let the engine handle everything
		testCaseResult := r.runTestCase(testCase, testSuiteResult)
		testCaseResult.Print(r.output) // Print immediately as test completes
		testSuiteResult.AddResult(testCaseResult)
	}

	// Complete the test suite result
	testSuiteResult.Complete()

	// Print only the file summary (not individual test results)
	testSuiteResult.Print(r.output)

	// Return error if any tests failed
	if testSuiteResult.HasFailures() {
		return fmt.Errorf("tests failed in testsuite %s", filepath.Base(r.testSuiteFile))
	}

	return nil
}

// runTestCase executes a single test case and returns a complete TestCaseResult
//
//nolint:gocognit // Complex test case execution with multiple validation and execution phases
func (r *Runner) runTestCase(testCase api.TestCase, testSuiteResult *engine.TestSuiteResult) *engine.TestCaseResult {
	if r.runTestCaseFunc != nil {
		return r.runTestCaseFunc(testCase)
	}

	if r.Debug {
		utils.DebugPrintf("Starting test case '%s'\n", testCase.Name)
	}

	result := engine.NewTestCaseResult(testCase.Name, testCase.ID, r.Verbose, r.ShowRender, r.ShowValidate, r.ShowHooks, r.ShowAssertions)
	// Create a temporary directory for the test case (with inputs and outputs subdirectories)
	var err error

	r.testCaseTmpDir, err = afero.TempDir(r.fs, "", "xprin-testcase-")
	if err != nil {
		return result.Fail(fmt.Errorf("failed to create temporary directory: %w", err))
	}

	defer func() {
		_ = r.fs.RemoveAll(r.testCaseTmpDir)
	}()

	// Create subdirectories for inputs and outputs
	r.inputsDir = filepath.Join(r.testCaseTmpDir, "inputs")

	r.outputsDir = filepath.Join(r.testCaseTmpDir, "outputs")
	if err := r.fs.MkdirAll(r.inputsDir, 0o750); err != nil {
		return result.Fail(fmt.Errorf("failed to create inputs directory: %w", err))
	}

	if err := r.fs.MkdirAll(r.outputsDir, 0o750); err != nil {
		return result.Fail(fmt.Errorf("failed to create outputs directory: %w", err))
	}

	if r.Debug {
		utils.DebugPrintf("Created temporary directory for test case: %s\n", r.testCaseTmpDir)
		utils.DebugPrintf("- Inputs: %s\n", r.inputsDir)
		utils.DebugPrintf("- Outputs: %s\n", r.outputsDir)
	}

	if r.testSuiteSpec.HasCommon() {
		testCase.MergeCommon(r.testSuiteSpec.Common)
	}

	// Process template variables for this test case
	if err := r.processTemplateVariables(&testCase, testSuiteResult); err != nil {
		return result.Fail(fmt.Errorf("failed to process template variables: %w", err))
	}

	if err := testCase.CheckMandatoryFields(); err != nil {
		return result.Fail(err)
	}

	if r.Debug {
		r.debugPrintTestCase(testCase, "Test specification:")
	}

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

		testCase.Inputs.XR, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.XR)
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

		testCase.Inputs.Claim, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.Claim)
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

	testCase.Inputs.Composition, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.Composition)
	if err != nil {
		failedExpandedPaths = append(failedExpandedPaths, fmt.Sprintf("failed to expand composition path: %v", err))
	}

	if err := r.verifyPathExists(testCase.Inputs.Composition); err != nil {
		unverifiedPaths = append(unverifiedPaths, fmt.Sprintf("composition file not found: %v", err))
	}

	if !filepath.IsAbs(testCase.Inputs.Functions) {
		anyPathExpanded = true
	}

	testCase.Inputs.Functions, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.Functions)
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

		testCase.Inputs.CRDs[i], err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, originalCRDPath)
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

		testCase.Inputs.ContextFiles[key], err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, originalContextFilePath)
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

		testCase.Inputs.ObservedResources, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.ObservedResources)
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

		testCase.Inputs.ExtraResources, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.ExtraResources)
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

		testCase.Inputs.FunctionCredentials, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Inputs.FunctionCredentials)
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

		testCase.Patches.XRD, err = r.expandPathRelativeToTestSuiteFile(r.testSuiteFile, testCase.Patches.XRD)
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
		r.debugPrintTestCase(testCase, "Test specification with expanded input paths:")
	}

	// Copy all inputs to the temporary inputs directory
	if testCase.HasXR() {
		testCase.Inputs.XR, err = r.copyInput(testCase.Inputs.XR, "xr")
		if err != nil {
			return result.Fail(err)
		}
	} else {
		testCase.Inputs.Claim, err = r.copyInput(testCase.Inputs.Claim, "claim")
		if err != nil {
			return result.Fail(err)
		}
	}

	testCase.Inputs.Composition, err = r.copyInput(testCase.Inputs.Composition, "composition")
	if err != nil {
		return result.Fail(err)
	}

	testCase.Inputs.Functions, err = r.copyInput(testCase.Inputs.Functions, "functions")
	if err != nil {
		return result.Fail(err)
	}

	crdsDir := filepath.Join(r.inputsDir, "crds")

	uniqueNames := uniqueBaseNamesForPaths(testCase.Inputs.CRDs)
	for i, crdPath := range testCase.Inputs.CRDs {
		dest := filepath.Join(crdsDir, uniqueNames[i])

		testCase.Inputs.CRDs[i], err = r.copyToPath(crdPath, dest)
		if err != nil {
			return result.Fail(err)
		}
	}

	for key, contextFile := range testCase.Inputs.ContextFiles {
		testCase.Inputs.ContextFiles[key], err = r.copyInput(contextFile, "context-files")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.ObservedResources != "" {
		testCase.Inputs.ObservedResources, err = r.copyInput(testCase.Inputs.ObservedResources, "observed-resources")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.ExtraResources != "" {
		testCase.Inputs.ExtraResources, err = r.copyInput(testCase.Inputs.ExtraResources, "extra-resources")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Inputs.FunctionCredentials != "" {
		testCase.Inputs.FunctionCredentials, err = r.copyInput(testCase.Inputs.FunctionCredentials, "function-credentials")
		if err != nil {
			return result.Fail(err)
		}
	}

	if testCase.Patches.XRD != "" {
		testCase.Patches.XRD, err = r.copyInput(testCase.Patches.XRD, "xrd")
		if err != nil {
			return result.Fail(err)
		}
	}

	// Execute pre-test hooks
	if testCase.HasPreTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		result.PreTestHooksResults, err = hookExecutor.executeHooks(testCase.Hooks.PreTest, "pre-test", testCase.Inputs, nil, testSuiteResult.GetCompletedTests())
		result.ProcessPreTestHooksOutput()

		if err != nil {
			return result.Fail(nil)
		}
	}

	// Handle XR input - either convert Claim to XR or use provided XR file
	var inputXR string
	if testCase.HasXR() {
		// Use provided XR file directly
		inputXR = testCase.Inputs.XR
		if r.Debug {
			utils.DebugPrintf("Using provided XR file: %s\n", inputXR)
		}
	} else {
		// Convert Claim to XR
		inputXR, err = r.convertClaimToXRFunc(r, testCase.Inputs.Claim, r.inputsDir)
		if err != nil {
			return result.Fail(fmt.Errorf("failed to convert Claim: %w", err))
		}
	}

	// Patch XR if needed (XRD and/or connection secret)
	if testCase.HasPatches() {
		inputXR, err = r.patchXRFunc(r, inputXR, r.inputsDir, testCase.Patches)
		if err != nil {
			return result.Fail(fmt.Errorf("failed to patch XR: %w", err))
		}
	}

	renderArgs := make([]string, 0, len(r.Render)+3)
	renderArgs = append(renderArgs, r.Render...)
	renderArgs = append(renderArgs, inputXR, testCase.Inputs.Composition, testCase.Inputs.Functions)

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
	result.Outputs.Render = filepath.Join(r.outputsDir, "rendered.yaml")
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
		result.Outputs.XR = filepath.Join(r.outputsDir, "xr.yaml")

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
		filepath := filepath.Join(r.outputsDir, filename)

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

	var finalError []string
	if len(testCase.Inputs.CRDs) >= 1 {
		validateArgs := make([]string, 0, len(r.Validate)+3)
		validateArgs = append(validateArgs, r.Validate...)
		validateArgs = append(validateArgs, filepath.Join(r.inputsDir, "crds"), result.Outputs.Render)
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
		validateOutputFile := filepath.Join(r.outputsDir, "validate.yaml")
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

	// Execute assertions if any are defined (collect errors but don't fail immediately)
	if testCase.HasAssertions() {
		exec := newAssertionExecutor(
			r.fs,
			&result.Outputs,
			r.Debug,
			r.testSuiteFile,
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
	}

	// Execute post-test hooks (after assertions)
	if testCase.HasPostTestHooks() {
		hookExecutor := newHookExecutor(r.Repositories, r.Debug, r.runCommand, r.renderTemplate)

		result.PostTestHooksResults, _ = hookExecutor.executeHooks(testCase.Hooks.PostTest, "post-test", testCase.Inputs, &result.Outputs, testSuiteResult.GetCompletedTests())
		result.ProcessPostTestHooksOutput()
		// On post-test hook failure, section shows failed hooks; HasPipelineFailure() is true from results
	}

	// Copy outputs to testsuite artifacts directory
	if testCase.ID != "" {
		artifactsDir := filepath.Join(r.testSuiteArtifactsDir, testCase.ID)
		if err := r.copy(r.outputsDir, artifactsDir); err != nil {
			return result.Fail(fmt.Errorf("failed to copy outputs to testsuite artifacts directory: %w", err))
		}

		if r.Debug {
			utils.DebugPrintf("Copied outputs to testsuite artifacts directory: %s\n", artifactsDir)
		}

		// Update Outputs paths to point to artifact paths for cross-test references
		result.Outputs.Render = filepath.Join(artifactsDir, "rendered.yaml")

		result.Outputs.XR = filepath.Join(artifactsDir, "xr.yaml")
		if result.Outputs.Validate != nil {
			*result.Outputs.Validate = filepath.Join(artifactsDir, "validate.yaml")
		}

		// Update Rendered map paths to point to artifact paths
		for key, path := range result.Outputs.Rendered {
			filename := filepath.Base(path)
			result.Outputs.Rendered[key] = filepath.Join(artifactsDir, filename)
		}
	}

	// Fail with infrastructure/non-section errors if any; otherwise fail with nil when pipeline failed
	if len(finalError) > 0 {
		return result.Fail(fmt.Errorf("%s", strings.Join(finalError, "\n")))
	}

	if result.HasPipelineFailure() {
		return result.Fail(nil)
	}

	// Complete the test case result
	if r.Debug {
		utils.DebugPrintf("Test case '%s' completed with status: %s\n", testCase.Name, result.Status.String())
	}

	return result.Complete()
}

// renderTemplate renders Go template syntax with the given context.
func (r *Runner) renderTemplate(content string, templateContext *templateContext, templateName string) (string, error) {
	// Parse and execute template
	tmpl, err := template.New(templateName).Option("missingkey=error").Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, templateContext); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// processTemplateVariables processes template variables for a test case.
func (r *Runner) processTemplateVariables(testCase *api.TestCase, testSuiteResult *engine.TestSuiteResult) error {
	// Check if there are any template variables by converting to YAML temporarily
	yamlData, err := yaml.Marshal(testCase)
	if err != nil {
		return fmt.Errorf("failed to marshal test case to YAML: %w", err)
	}

	content := string(yamlData)

	// Check if there are any template variables
	if !strings.Contains(content, testexecutionUtils.PlaceholderOpen) {
		return nil // No template variables to process
	}

	// Preserve hooks from testCase before removeHooks modifies them
	originalHooks := testCase.Hooks

	content, err = r.removeHooks(testCase)
	if err != nil {
		return fmt.Errorf("failed to remove hooks from YAML: %w", err)
	}

	content = testexecutionUtils.RestoreTemplateVars(content)

	// Render template
	templateContext := newTemplateContext(r.Repositories, testCase.Inputs, nil, testSuiteResult.GetCompletedTests())

	content, err = r.renderTemplate(content, templateContext, "testcase")
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Parse the processed YAML back to test case and restore hooks
	if err := r.restoreHooks(content, testCase, originalHooks); err != nil {
		return fmt.Errorf("failed to restore hooks: %w", err)
	}

	return nil
}

// removeHooks removes all hooks from the test case.
func (r *Runner) removeHooks(testCase *api.TestCase) (string, error) {
	if testCase.Hooks.PreTest != nil {
		testCase.Hooks.PreTest = nil
	}

	if testCase.Hooks.PostTest != nil {
		testCase.Hooks.PostTest = nil
	}

	yamlData, err := yaml.Marshal(testCase)
	if err != nil {
		return "", fmt.Errorf("failed to marshal test case to YAML: %w", err)
	}

	return string(yamlData), nil
}

// restoreHooks parses the processed YAML back to test case and restores hooks.
func (r *Runner) restoreHooks(content string, testCase *api.TestCase, originalHooks api.Hooks) error {
	// Parse the processed YAML back to test case
	if err := yaml.Unmarshal([]byte(content), testCase); err != nil {
		return fmt.Errorf("failed to parse processed YAML: %w", err)
	}

	// Restore hooks (they will be processed in executeHooks with the correct context)
	testCase.Hooks = originalHooks

	return nil
}
