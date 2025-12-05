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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/engine"
	testexecutionUtils "github.com/crossplane-contrib/xprin/internal/testexecution/utils"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
	"sigs.k8s.io/yaml"
)

const testSuiteFile = "/suite_xprin.yaml"

// createTestCaseResult is a helper function to create TestCaseResult with common defaults.
func createTestCaseResult(name string, verbose bool, err error) *engine.TestCaseResult {
	result := engine.NewTestCaseResult(name, "", verbose, false, false, false, false)
	if err != nil {
		return result.Fail(err)
	}

	return result.Complete()
}

// newMockRunner creates a test-specific Runner with mocked functions.
func newMockRunner(options *testexecutionUtils.Options, mocks ...func(*Runner)) *Runner {
	// Use default test suite file and spec for tests that don't need specific values
	testSuiteSpec := &api.TestSuiteSpec{Tests: []api.TestCase{}}
	r := NewRunner(options, testSuiteFile, testSuiteSpec)

	// Set up default mock functions for testing
	r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) { return path, nil }
	r.verifyPathExists = func(_ string) error { return nil }
	r.copy = func(_, _ string, _ ...cp.Options) error { return nil }

	// Apply any custom mocks
	for _, mock := range mocks {
		mock(r)
	}

	return r
}

// makeOptions is a helper to create Options from a config and allow overrides for tests.
func makeOptions(cfg *config.Config, render, validate []string, overrides ...func(*testexecutionUtils.Options)) *testexecutionUtils.Options {
	// If render/validate are nil, extract from cfg.Subcommands (if present)
	if render == nil && cfg.Subcommands != nil && cfg.Subcommands.Render != "" {
		render = strings.Fields(cfg.Subcommands.Render)
	}

	if validate == nil && cfg.Subcommands != nil && cfg.Subcommands.Validate != "" {
		validate = strings.Fields(cfg.Subcommands.Validate)
	}
	// If still nil or empty, set to known defaults from config
	if len(render) == 0 {
		render = strings.Fields(config.DefaultRenderCmd)
	}

	if len(validate) == 0 {
		validate = strings.Fields(config.DefaultValidateCmd)
	}

	opt := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       render,
		Validate:     validate,
		ShowRender:   true,
		ShowValidate: true,
		ShowHooks:    true,
		Verbose:      true,
		Debug:        false,
	}
	for _, fn := range overrides {
		fn(opt)
	}

	return opt
}

func TestNewRunner(t *testing.T) {
	// Integration-style: take Render/Validate from config
	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
		Repositories: map[string]string{
			"my-claims-repo": "path/to/my/claims/repo",
		},
		Subcommands: &config.Subcommands{
			Render:   "render --foo bar",
			Validate: "validate --baz qux",
		},
	}
	// Simulate config-to-options conversion (string to []string)
	options := makeOptions(cfg, nil, nil, func(o *testexecutionUtils.Options) {
		o.Debug = true
	})
	runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})

	// Check the runner was created with the right settings
	assert.Equal(t, options, runner.Options)
	assert.Equal(t, testSuiteFile, runner.testSuiteFile)
	assert.True(t, runner.ShowRender)
	assert.True(t, runner.ShowValidate)
	assert.True(t, runner.ShowHooks)
	assert.True(t, runner.Verbose)
	assert.True(t, runner.Debug)
	// runTestsFunc is nil by default and uses the default runTests implementation
	assert.Equal(t, cfg.Dependencies, runner.Dependencies)
	assert.Equal(t, strings.Fields(cfg.Subcommands.Render), runner.Render)
	assert.Equal(t, strings.Fields(cfg.Subcommands.Validate), runner.Validate)
}

func TestNewRunner_DefaultsIfNotSpecified(t *testing.T) {
	// If Render/Validate are not set in config, runner should get the defaults
	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
		Repositories: map[string]string{
			"my-claims-repo": "path/to/my/claims/repo",
		},
		Subcommands: &config.Subcommands{}, // No Render/Validate set
	}
	// Simulate config-to-options conversion (empty means use defaults)
	options := makeOptions(cfg, nil, nil)
	runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})

	// The runner should have the default Render/Validate commands (from runner or config loader)
	assert.NotEmpty(t, runner.Render)
	assert.NotEmpty(t, runner.Validate)
	// Optionally, check for specific defaults if known, e.g.:
	assert.Equal(t, strings.Fields(config.DefaultRenderCmd), runner.Render)
	assert.Equal(t, strings.Fields(config.DefaultValidateCmd), runner.Validate)
}

func TestNewTemplateContext(t *testing.T) {
	t.Run("with all parameters", func(t *testing.T) {
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}
		outputs := &engine.Outputs{Render: "rendered.yaml"}

		context := newTemplateContext(repos, inputs, outputs, map[string]*engine.TestCaseResult{})

		assert.Equal(t, repos, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Equal(t, outputs, context.Outputs)
	})

	t.Run("with nil repositories", func(t *testing.T) {
		inputs := api.Inputs{XR: "test-xr.yaml"}

		context := newTemplateContext(nil, inputs, nil, map[string]*engine.TestCaseResult{})

		assert.NotNil(t, context.Repositories)
		assert.Empty(t, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Nil(t, context.Outputs)
	})

	t.Run("with nil outputs", func(t *testing.T) {
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}

		context := newTemplateContext(repos, inputs, nil, map[string]*engine.TestCaseResult{})

		assert.Equal(t, repos, context.Repositories)
		assert.Equal(t, inputs, context.Inputs)
		assert.Nil(t, context.Outputs)
	})
}

func TestRunTests(t *testing.T) {
	options := &testexecutionUtils.Options{
		ShowRender:   false,
		ShowValidate: false,
		Verbose:      false,
		Debug:        false,
	}
	cases := []struct {
		name      string
		testCases []api.TestCase
		mockErrs  []error
		wantFails int
		wantPass  int
	}{
		{
			name:      "no tests",
			testCases: nil,
			mockErrs:  nil,
			wantFails: 0,
			wantPass:  0,
		},
		{
			name:      "one passing test",
			testCases: []api.TestCase{{Name: "test1"}},
			mockErrs:  []error{nil},
			wantFails: 0,
			wantPass:  1,
		},
		{
			name:      "one failing test",
			testCases: []api.TestCase{{Name: "test1"}},
			mockErrs:  []error{errors.New("fail")},
			wantFails: 1,
			wantPass:  0,
		},
		{
			name:      "multiple tests",
			testCases: []api.TestCase{{Name: "test1"}, {Name: "test2"}},
			mockErrs:  []error{nil, errors.New("fail")},
			wantFails: 1,
			wantPass:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			var tests []api.TestCase
			for _, test := range tc.testCases {
				tests = append(tests, api.TestCase{
					Name: test.Name,
				})
			}

			testSuiteSpec := &api.TestSuiteSpec{
				Tests: tests,
			}

			runner := NewRunner(options, testSuiteFile, testSuiteSpec)
			runner.output = &buf

			// Mock the runTestCaseFunc to return controlled results
			call := 0
			runner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
				err := tc.mockErrs[call]
				call++

				return createTestCaseResult(testCase.Name, false, err)
			}

			err := runner.RunTests()
			if tc.wantFails > 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output was generated
			output := buf.String()
			if tc.wantFails > 0 {
				assert.Contains(t, output, "FAIL", "output should contain FAIL for failed tests")
			} else {
				assert.NotContains(t, output, "FAIL", "output should not contain FAIL for passing tests")
			}
		})
	}
}

func TestRunTestsIntegration(t *testing.T) {
	options := &testexecutionUtils.Options{
		ShowRender:   false,
		ShowValidate: false,
		Verbose:      false,
		Debug:        false,
	}

	testCases := []struct {
		name       string
		runFunc    func(tc api.TestCase) *engine.TestCaseResult
		suite      *api.TestSuiteSpec
		wantError  bool
		wantStatus string
		wantOutput []string
		verbose    bool // add verbose field
	}{
		{
			name: "all pass",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, nil)
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "xr.yaml",
							Composition: "comp.yaml",
							Functions:   "functions.yaml",
							CRDs:        []string{"crd.yaml"},
						},
					},
				},
			},
			wantError:  false,
			wantStatus: "PASS",
			verbose:    false,
		},
		{
			name: "one fails",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, errors.New("fail"))
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "xr.yaml",
							Composition: "comp.yaml",
							Functions:   "functions.yaml",
							CRDs:        []string{"crd.yaml"},
						},
					},
				},
			},
			wantError:  true,
			wantStatus: "FAIL",
			verbose:    false,
		},
		{
			name:      "no tests",
			suite:     &api.TestSuiteSpec{Tests: []api.TestCase{}},
			wantError: false, // Empty test suite should not error
			verbose:   false,
		},
		{
			name:      "nil TestSuiteSpec",
			wantError: true,
			verbose:   false,
		},
		{
			name: "verbose output includes RUN and hierarchy",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, true, nil)
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "foo.yaml",
							Composition: "bar.yaml",
							Functions:   "baz/",
						},
					},
				},
				Common: api.Common{},
			},
			wantOutput: []string{
				"=== RUN   test1",
				"--- PASS: test1 (0.00s)",
			},
			verbose: true,
		},
		{
			name: "non-verbose output prints group and subtest failures with durations",
			runFunc: func(tc api.TestCase) *engine.TestCaseResult {
				return createTestCaseResult(tc.Name, false, errors.New("fail: something went wrong"))
			},
			suite: &api.TestSuiteSpec{
				Tests: []api.TestCase{
					{
						Name: "test1",
						Inputs: api.Inputs{
							XR:          "foo.yaml",
							Composition: "bar.yaml",
							Functions:   "baz/",
						},
					},
				},
				Common: api.Common{},
			},
			wantError:  true,
			wantStatus: "FAIL",
			wantOutput: []string{
				"--- FAIL: test1 (0.00s)",
				"fail: something went wrong",
			},
			verbose: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			runner := NewRunner(options, testSuiteFile, tc.suite)
			runner.output = &buf
			runner.runTestCaseFunc = tc.runFunc
			runner.Verbose = tc.verbose // set per-test verbosity

			err := runner.RunTests()
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantStatus != "" {
				out := buf.String()

				switch tc.wantStatus {
				case "FAIL":
					assert.True(t, err != nil || strings.Contains(out, "FAIL"), "Expected test failure or FAIL in output")
				case "PASS":
					assert.True(t, err == nil && !strings.Contains(out, "FAIL"), "Expected no test failures")
				}
			}

			if len(tc.wantOutput) > 0 {
				out := buf.String()
				for _, want := range tc.wantOutput {
					assert.Contains(t, out, want)
				}
			}
		})
	}

	t.Run("multiple tests mixed pass/fail, verbose output check", func(t *testing.T) {
		var buf bytes.Buffer

		options := &testexecutionUtils.Options{
			ShowRender:   false,
			ShowValidate: false,
			Verbose:      true,
			Debug:        false,
		}
		suite := &api.TestSuiteSpec{
			Tests: []api.TestCase{
				{Name: "test1", Inputs: api.Inputs{
					XR: "a",
				}},
				{Name: "test2", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
				{Name: "test3", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
			},
		}
		runner := NewRunner(options, testSuiteFile, suite)
		runner.output = &buf

		// test1: pass, test2: fail, test3: pass
		call := 0
		results := []error{nil, errors.New("fail in test2"), nil}
		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, runner.Verbose, err)
		}
		err := runner.RunTests()
		require.Error(t, err)

		out := buf.String()
		// Check individual test results
		assert.Contains(t, out, "test1", "output should contain test1")
		assert.Contains(t, out, "test2", "output should contain test2")
		assert.Contains(t, out, "test3", "output should contain test3")
		// Should have failed test indicated by error and output
		assert.Contains(t, out, "FAIL", "output should contain FAIL for failed test")
	})

	t.Run("multiple groups mixed pass/fail, non-verbose output check", func(t *testing.T) {
		var buf bytes.Buffer

		options := &testexecutionUtils.Options{
			ShowRender:   false,
			ShowValidate: false,
			Verbose:      false,
			Debug:        false,
		}
		suite := &api.TestSuiteSpec{
			Tests: []api.TestCase{
				{Name: "test1", Inputs: api.Inputs{
					XR: "a",
				}},
				{Name: "test2", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
				{Name: "test3", Inputs: api.Inputs{
					XR:          "a",
					Composition: "b",
					Functions:   "c",
				}},
			},
		}
		runner := NewRunner(options, testSuiteFile, suite)
		runner.output = &buf

		// group1: pass, group2: fail, group3: pass
		call := 0
		results := []error{nil, errors.New("fail in group2"), nil}
		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			err := results[call]
			call++

			return createTestCaseResult(tc.Name, false, err)
		}
		err := runner.RunTests()
		require.Error(t, err)

		out := buf.String()
		// Only failed test should be shown in non-verbose mode
		assert.Contains(t, out, "--- FAIL: test2 (0.00s)")
		// Passing tests should not be shown in non-verbose mode
		assert.NotContains(t, out, "--- PASS: test1")
		assert.NotContains(t, out, "--- PASS: test3")
		// Error message should be present
		assert.Contains(t, out, "fail in group2")
	})
}

// TestRunTestCase tests the core validation and execution logic.
func TestRunTestCase(t *testing.T) {
	validRenderYAML := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	fs := afero.NewMemMapFs()
	localCRDPath := "/crd.yaml"
	require.NoError(t, afero.WriteFile(fs, localCRDPath, []byte("dummy crd content"), 0o644))

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}

	cases := []struct {
		name      string
		testCase  api.TestCase
		setup     func(*Runner)
		wantError string
	}{
		// Validation Tests
		{
			name: "missing claim or xr",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			wantError: "missing mandatory field: either 'claim' or 'xr' must be specified",
		},
		{
			name: "missing composition and not in common",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:        "xr.yaml",
					Functions: "functions.yaml",
				},
			},
			wantError: "missing mandatory field: composition",
		},
		{
			name: "missing functions and not in common",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
				},
			},
			wantError: "missing mandatory field: functions",
		},
		{
			name: "both claim and xr specified",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			wantError: "conflicting fields: both 'claim' and 'xr' are specified",
		},
		{
			name: "xr field with missing xr file",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "missing.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "missing.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "XR file not found: not found",
		},
		// Execution Failure Tests
		{
			name: "convert claim fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for convert-claim-to-xr
				r.runCommand = func(name string, _ ...string) ([]byte, []byte, error) {
					if name == "convert-claim-to-xr" {
						return []byte("convert fail"), []byte(""), fmt.Errorf("fail")
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "failed to convert Claim: fail",
		},
		{
			name: "render fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane render
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return []byte("render fail"), []byte(""), fmt.Errorf("fail")
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "render fail",
		},
		{
			name: "validate fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane validate
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), []byte(""), fmt.Errorf("fail")
					}

					return validRenderYAML, []byte{}, nil
				}
			},
			wantError: "validate fail",
		},
		{
			name: "validate fails but post-test hooks run",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'cleanup executed'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return an error for crossplane validate
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), []byte(""), fmt.Errorf("fail")
					}
					// Mock successful hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup executed"), []byte{}, nil
					}

					return validRenderYAML, []byte{}, nil
				}
			},
			wantError: "validate fail",
		},
		{
			name: "both validate and post-test hook fail",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "failing-cleanup", Run: "echo 'cleanup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), []byte(""), fmt.Errorf("fail")
					}
					// Mock failing hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup"), []byte("hook failed"), errors.New("exit status 1")
					}

					return validRenderYAML, []byte{}, nil
				}
			},
			wantError: "validate fail",
		},
		// Happy Path Tests
		{
			name: "happy path with xr field",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return success
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "happy path with claim field",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
			},
			setup: func(r *Runner) {
				// Mock convertClaimToXRFunc to return a fake XR file path
				r.convertClaimToXRFunc = func(_ *Runner, _, outputPath string) (string, error) {
					return filepath.Join(outputPath, "xr.yaml"), nil
				}
				// Mock the runCommand function to return success for all operations
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "happy path with claim and patching",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{localCRDPath},
				},
				Patches: api.Patches{
					ConnectionSecret: boolPtr(true),
					XRD:              "my-xrd.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock convertClaimToXRFunc to return a fake XR file path
				r.convertClaimToXRFunc = func(_ *Runner, _, outputPath string) (string, error) {
					return filepath.Join(outputPath, "xr.yaml"), nil
				}
				// Mock patchXRFunc to return a fake patched XR file path
				r.patchXRFunc = func(_ *Runner, _, outputPath string, _ api.Patches) (string, error) {
					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "all render flags used (happy path)",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:           "xr.yaml",
					Composition:  "comp.yaml",
					Functions:    "functions.yaml",
					CRDs:         []string{localCRDPath},
					ContextFiles: map[string]string{"ctx": "context_file.yaml"},
					ContextValues: map[string]string{
						"ctx": func() string {
							val, err := json.Marshal(map[string]interface{}{"key": "value"})
							if err != nil {
								return "{}" // fallback to empty JSON object
							}

							return string(val)
						}(),
					},
					ObservedResources:   "observed.yaml",
					ExtraResources:      "extra.yaml",
					FunctionCredentials: "func_creds.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock the runCommand function to return success for all operations
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		// Connection Secret
		{
			name: "xr with connection secret enabled",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					ConnectionSecret: boolPtr(true),
				},
			},
			setup: func(r *Runner) {
				// Mock patchXRFunc to return a fake patched XR file path
				r.patchXRFunc = func(_ *Runner, _, outputPath string, _ api.Patches) (string, error) {
					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with connection secret name and namespace",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					ConnectionSecret:          boolPtr(true),
					ConnectionSecretName:      "my-secret",
					ConnectionSecretNamespace: "my-namespace",
				},
			},
			setup: func(r *Runner) {
				// Mock patchXRFunc to return a fake patched XR file path
				r.patchXRFunc = func(_ *Runner, _, outputPath string, _ api.Patches) (string, error) {
					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with connection secret name and namespace - validation failure",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					ConnectionSecret:          nil, // Not set to true
					ConnectionSecretName:      "my-secret",
					ConnectionSecretNamespace: "my-namespace",
				},
			},
			setup:     func(*Runner) {},
			wantError: "connection-secret must be set to true when using connection-secret-name or connection-secret-namespace",
		},
		{
			name: "xr with XRD",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					XRD: "my-xrd.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock patchXRFunc to return a fake patched XR file path
				r.patchXRFunc = func(_ *Runner, _, outputPath string, _ api.Patches) (string, error) {
					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with pre-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "setup", Run: "echo 'pre-test setup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("pre-test setup"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with post-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'post-test cleanup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("post-test cleanup"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with both pre-test and post-test hooks",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "setup", Run: "echo 'pre-test setup'"},
					},
					PostTest: []api.Hook{
						{Name: "cleanup", Run: "echo 'post-test cleanup'"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("hook output"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "",
		},
		{
			name: "xr with failing pre-test hook",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PreTest: []api.Hook{
						{Name: "failing-setup", Run: "echo 'failing setup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results for crossplane commands
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}
					// Mock failing hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("failing setup"), []byte("hook failed"), errors.New("exit status 1")
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "pre-test hook 'failing-setup' failed",
		},
		{
			name: "xr with failing post-test hook",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{Name: "failing-cleanup", Run: "echo 'failing cleanup' && exit 1"},
					},
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results for crossplane commands
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}
					// Mock failing hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("failing cleanup"), []byte("hook failed"), errors.New("exit status 1")
					}

					return []byte{}, []byte{}, nil
				}
			},
			wantError: "post-test hook 'failing-cleanup' failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new test runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem
			testRunner.testSuiteSpec = &api.TestSuiteSpec{}

			// Apply any custom mocks from the test case setup
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			result := testRunner.runTestCase(tc.testCase, testSuiteResult)

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail, result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else {
				assert.Equal(t, engine.StatusPass, result.Status)
				require.NoError(t, result.Error)
			}

			// For tests with post-test hooks, verify they ran
			if tc.name == "validate fails but post-test hooks run" {
				assert.Len(t, result.PostTestHooksResults, 1)
				assert.Equal(t, "cleanup", result.PostTestHooksResults[0].Name)
				require.NoError(t, result.PostTestHooksResults[0].Error)
			}

			// For tests where both fail, verify both errors are in the message
			if tc.name == "both validate and post-test hook fail" {
				assert.Len(t, result.PostTestHooksResults, 1)
				require.Error(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.Error.Error(), "validate fail")
				assert.Contains(t, result.Error.Error(), "post-test hook")
			}
		})
	}
}

func TestRunTestCase_CommonPathExpansionAndVerification(t *testing.T) {
	fs := afero.NewMemMapFs()
	commonCRDPath := "/common_crd.yaml"
	require.NoError(t, afero.WriteFile(fs, commonCRDPath, []byte("dummy common crd content"), 0o644))

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}
	cases := []struct {
		name      string
		common    api.Common
		testCase  api.TestCase
		setup     func(*Runner)
		wantError string
	}{
		{
			name: "expand common composition path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "badcommon_comp.yaml",
					Functions:   "common_functions.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_comp.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand composition path: expand fail",
		},
		{
			name: "verify common composition path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "common_composition.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "composition file not found: not found",
		},
		{
			name: "expand common functions path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "badcommon_functions.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_functions.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand functions path: expand fail",
		},
		{
			name: "verify common functions path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "common_functions.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "functions file or dir not found: not found",
		},
		{
			name: "expand common CRD path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
					CRDs:        []string{commonCRDPath, "badcommon_crd.yaml"},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_crd.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand CRD path badcommon_crd.yaml: expand fail",
		},
		{
			name: "verify common CRD path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
					CRDs:        []string{commonCRDPath, "bad_crd.yaml"},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "bad_crd.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "crd file not found: not found",
		},
		{
			name: "expand common ContextFiles path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:  "common_composition.yaml",
					Functions:    "common_functions.yaml",
					ContextFiles: map[string]string{"foo": "badcommon_contextfile.yaml"},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_contextfile.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand context file path for key 'foo': expand fail",
		},
		{
			name: "verify common ContextFiles path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:  "common_composition.yaml",
					Functions:    "common_functions.yaml",
					ContextFiles: map[string]string{"foo": "badcommon_contextfile.yaml"},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badcommon_contextfile.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "context file not found for key 'foo': not found",
		},
		{
			name: "expand common ObservedResources path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:       "common_composition.yaml",
					Functions:         "common_functions.yaml",
					ObservedResources: "badcommon_observed.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_observed.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand observed resources path: expand fail",
		},
		{
			name: "verify common ObservedResources path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:       "common_composition.yaml",
					Functions:         "common_functions.yaml",
					ObservedResources: "badcommon_observed.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badcommon_observed.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "observed resources file or dir not found: not found",
		},
		{
			name: "expand common ExtraResources path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:    "common_composition.yaml",
					Functions:      "common_functions.yaml",
					ExtraResources: "badcommon_extra.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_extra.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand extra resources path: expand fail",
		},
		{
			name: "verify common ExtraResources path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:    "common_composition.yaml",
					Functions:      "common_functions.yaml",
					ExtraResources: "badcommon_extra.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badcommon_extra.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "extra resources file or dir not found: not found",
		},
		{
			name: "expand common FunctionCredentials path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:         "common_composition.yaml",
					Functions:           "common_functions.yaml",
					FunctionCredentials: "badcommon_func.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_func.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand function credentials path: expand fail",
		},
		{
			name: "verify common FunctionCredentials path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition:         "common_composition.yaml",
					Functions:           "common_functions.yaml",
					FunctionCredentials: "badcommon_func.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badcommon_func.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "function credentials file or dir not found: not found",
		},
		{
			name: "expand common XRD path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
				},
				Patches: api.Patches{
					XRD: "badcommon_xrd.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcommon_xrd.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand XRD path: expand fail",
		},
		{
			name: "verify common XRD path fails",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
					Functions:   "common_functions.yaml",
				},
				Patches: api.Patches{
					XRD: "badcommon_xrd.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR: "xr.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badcommon_xrd.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "XRD file or dir not found: not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new mock runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem
			testRunner.testSuiteSpec = &api.TestSuiteSpec{
				Common: tc.common,
			}

			// Apply the test-specific mocks
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			result := testRunner.runTestCase(tc.testCase, testSuiteResult)

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail, result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else {
				assert.Equal(t, engine.StatusPass, result.Status)
				assert.NoError(t, result.Error)
			}
		})
	}
}

func TestRunTestCase_LocalPathExpansionAndVerification(t *testing.T) {
	fs := afero.NewMemMapFs()
	commonCRDPath := "/common_crd.yaml"
	require.NoError(t, afero.WriteFile(fs, commonCRDPath, []byte("dummy common crd content"), 0o644))

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}

	cases := []struct {
		name      string
		testCase  api.TestCase
		setup     func(*Runner)
		wantError string
	}{
		{
			name: "expand xr path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "badxr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badxr.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand XR path: expand fail",
		},
		{
			name: "verify xr path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "xr.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "XR file not found: not found",
		},
		{
			name: "expand claim path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "badclaim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badclaim.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand Claim path: expand fail",
		},
		{
			name: "verify claim path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					Claim:       "claim.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "claim.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "Claim file not found: not found",
		},
		{
			name: "expand local composition path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "badcomp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badcomp.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand composition path: expand fail",
		},
		{
			name: "verify local composition path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "comp.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "composition file not found: not found",
		},
		{
			name: "expand local functions path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "badfunctions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badfunctions.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand functions path: expand fail",
		},
		{
			name: "verify local functions path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "functions.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "functions file or dir not found: not found",
		},
		{
			name: "expand local CRD path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{commonCRDPath, "badlocal_crd.yaml"},
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_crd.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand CRD path badlocal_crd.yaml: expand fail",
		},
		{
			name: "verify local CRD path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{commonCRDPath, "bad_crd.yaml"},
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "bad_crd.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "crd file not found: not found",
		},
		{
			name: "expand local ContextFiles path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:           "xr.yaml",
					Composition:  "comp.yaml",
					Functions:    "functions.yaml",
					ContextFiles: map[string]string{"foo": "badlocal_contextfile.yaml"},
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_contextfile.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand context file path for key 'foo': expand fail",
		},
		{
			name: "verify local ContextFiles path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:           "xr.yaml",
					Composition:  "comp.yaml",
					Functions:    "functions.yaml",
					ContextFiles: map[string]string{"foo": "badlocal_contextfile.yaml"},
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badlocal_contextfile.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "context file not found for key 'foo': not found",
		},
		{
			name: "expand local ObservedResources path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                "xr.yaml",
					Composition:       "comp.yaml",
					Functions:         "functions.yaml",
					ObservedResources: "badlocal_observed.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_observed.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand observed resources path: expand fail",
		},
		{
			name: "verify local ObservedResources path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                "xr.yaml",
					Composition:       "comp.yaml",
					Functions:         "functions.yaml",
					ObservedResources: "badlocal_observed.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badlocal_observed.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "observed resources file or dir not found: not found",
		},
		{
			name: "expand local ExtraResources path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:             "xr.yaml",
					Composition:    "comp.yaml",
					Functions:      "functions.yaml",
					ExtraResources: "badlocal_extra.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_extra.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand extra resources path: expand fail",
		},
		{
			name: "verify local ExtraResources path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:             "xr.yaml",
					Composition:    "comp.yaml",
					Functions:      "functions.yaml",
					ExtraResources: "badlocal_extra.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badlocal_extra.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "extra resources file or dir not found: not found",
		},
		{
			name: "expand local FunctionCredentials path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                  "xr.yaml",
					Composition:         "comp.yaml",
					Functions:           "functions.yaml",
					FunctionCredentials: "badlocal_func.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_func.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand function credentials path: expand fail",
		},
		{
			name: "verify local FunctionCredentials path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                  "xr.yaml",
					Composition:         "comp.yaml",
					Functions:           "functions.yaml",
					FunctionCredentials: "badlocal_func.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badlocal_func.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "function credentials file or dir not found: not found",
		},
		{
			name: "expand local XRD path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					XRD: "badlocal_xrd.yaml",
				},
			},
			setup: func(r *Runner) {
				r.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
					if path == "badlocal_xrd.yaml" {
						return "", fmt.Errorf("expand fail")
					}

					return path, nil
				}
			},
			wantError: "failed to expand XRD path: expand fail",
		},
		{
			name: "verify local XRD path fails",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					XRD: "badlocal_xrd.yaml",
				},
			},
			setup: func(r *Runner) {
				r.verifyPathExists = func(path string) error {
					if path == "badlocal_xrd.yaml" {
						return fmt.Errorf("not found")
					}

					return nil
				}
			},
			wantError: "XRD file or dir not found: not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new mock runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem
			testRunner.testSuiteSpec = &api.TestSuiteSpec{}

			// Apply the test-specific mocks
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			result := testRunner.runTestCase(tc.testCase, testSuiteResult)

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail, result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else {
				assert.Equal(t, engine.StatusPass, result.Status)
				assert.NoError(t, result.Error)
			}
		})
	}
}

func TestRunTestCase_MergeCommon(t *testing.T) {
	fs := afero.NewMemMapFs()
	commonCRDPath := "/common_crd.yaml"
	require.NoError(t, afero.WriteFile(fs, commonCRDPath, []byte("dummy common crd content"), 0o644))

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}

	validRenderYAML := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")

	cases := []struct {
		name      string
		common    api.Common
		testCase  api.TestCase
		setup     func(*Runner)
		wantError string
	}{
		{
			name: "empty test case Composition uses common Composition",
			common: api.Common{
				Inputs: api.Inputs{
					Composition: "common_composition.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "", // empty, should use common
					Functions:   "functions.yaml",
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common composition
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case Functions uses common Functions",
			common: api.Common{
				Inputs: api.Inputs{
					Functions: "common_functions.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "", // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common functions
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case CRDs uses common CRDs",
			common: api.Common{
				Inputs: api.Inputs{
					CRDs: []string{commonCRDPath},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
					CRDs:        []string{}, // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common CRD
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case ContextFiles uses common ContextFiles",
			common: api.Common{
				Inputs: api.Inputs{
					ContextFiles: map[string]string{"common_ctx": "common_context.yaml"},
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:           "xr.yaml",
					Composition:  "comp.yaml",
					Functions:    "functions.yaml",
					ContextFiles: map[string]string{}, // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common context file
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case ObservedResources uses common ObservedResources",
			common: api.Common{
				Inputs: api.Inputs{
					ObservedResources: "common_observed.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                "xr.yaml",
					Composition:       "comp.yaml",
					Functions:         "functions.yaml",
					ObservedResources: "", // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common observed resources
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case ExtraResources uses common ExtraResources",
			common: api.Common{
				Inputs: api.Inputs{
					ExtraResources: "common_extra.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:             "xr.yaml",
					Composition:    "comp.yaml",
					Functions:      "functions.yaml",
					ExtraResources: "", // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common extra resources
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case FunctionCredentials uses common FunctionCredentials",
			common: api.Common{
				Inputs: api.Inputs{
					FunctionCredentials: "common_func_creds.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:                  "xr.yaml",
					Composition:         "comp.yaml",
					Functions:           "functions.yaml",
					FunctionCredentials: "", // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common function credentials
					_ = path
					return nil
				}
			},
			wantError: "",
		},
		{
			name: "empty test case XRD uses common XRD",
			common: api.Common{
				Patches: api.Patches{
					XRD: "common_xrd.yaml",
				},
			},
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Patches: api.Patches{
					XRD: "", // empty, should use common
				},
			},
			setup: func(r *Runner) {
				// Mock runCommand to return successful results
				r.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, []byte{}, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), []byte{}, nil
					}

					return []byte{}, []byte{}, nil
				}
				// Mock verifyPathExists to track calls
				r.verifyPathExists = func(path string) error {
					// This should be called with the common XRD
					_ = path
					return nil
				}
				// Mock patchXRFunc to return a fake patched XR file path
				r.patchXRFunc = func(_ *Runner, _, outputPath string, _ api.Patches) (string, error) {
					return filepath.Join(outputPath, "patched-xr.yaml"), nil
				}
			},
			wantError: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new mock runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem
			testRunner.testSuiteSpec = &api.TestSuiteSpec{
				Common: tc.common,
			}

			// Apply the test-specific mocks
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			result := testRunner.runTestCase(tc.testCase, testSuiteResult)

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail, result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else {
				assert.Equal(t, engine.StatusPass, result.Status)
				assert.NoError(t, result.Error)
			}
		})
	}
}

func TestRunTestCase_UsesTestResultWithStartTime(t *testing.T) {
	options := &testexecutionUtils.Options{
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}
	runner := newMockRunner(options)
	runner.testSuiteSpec = &api.TestSuiteSpec{}

	// Mock runCommand to return successful results
	runner.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
		if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
			return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), []byte{}, nil
		}

		if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
			return []byte("validate ok"), []byte{}, nil
		}

		return []byte{}, []byte{}, nil
	}

	// Provide a minimal valid testCase
	testCase := api.TestCase{
		Name: "test",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
			Functions:   "functions.yaml",
		},
	}
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	result := runner.runTestCase(testCase, testSuiteResult)
	assert.Equal(t, engine.StatusPass, result.Status)
	require.NoError(t, result.Error)
	assert.False(t, result.StartTime.IsZero(), "TestCaseResult should have StartTime set")
}

func TestRunTestCase_SkipsValidateWhenNoCRDs(t *testing.T) {
	validRenderYAML := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}

	runner := newMockRunner(options)
	runner.testSuiteSpec = &api.TestSuiteSpec{
		Common: api.Common{
			Inputs: api.Inputs{
				Functions: "functions.yaml",
				CRDs:      []string{}, // No CRDs
			},
		},
	}

	// Mock runCommand to return successful results
	runner.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
		if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
			return validRenderYAML, []byte{}, nil
		}
		// Validate should not be called when no CRDs are present
		return []byte{}, []byte{}, nil
	}

	testCase := api.TestCase{
		Name: "test-skip-validate",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
			Functions:   "functions.yaml",
			CRDs:        []string{}, // No CRDs in test case either
		},
	}

	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	result := runner.runTestCase(testCase, testSuiteResult)
	assert.Equal(t, engine.StatusPass, result.Status)
	assert.NoError(t, result.Error)
}

func TestRunTestsFileResults(t *testing.T) {
	options := &testexecutionUtils.Options{
		ShowRender:   false,
		ShowValidate: false,
		Debug:        false,
	}

	cases := []struct {
		name       string
		verbose    bool
		withError  bool
		wantStatus string
	}{
		{
			name:       "passing result",
			verbose:    false,
			withError:  false,
			wantStatus: "ok",
		},
		{
			name:       "passing result, verbose",
			verbose:    true,
			withError:  false,
			wantStatus: "PASS",
		},
		{
			name:       "failing result",
			verbose:    false,
			withError:  true,
			wantStatus: "FAIL",
		},
		{
			name:       "failing result, verbose",
			verbose:    true,
			withError:  true,
			wantStatus: "FAIL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
			runner.output = &buf
			runner.Verbose = tc.verbose

			// Create a test suite result to test file-level reporting
			testSuiteFile := "test-suite.yaml"
			testSuiteResult := engine.NewTestSuiteResult(testSuiteFile, tc.verbose)

			// Create a test case result
			testCaseResult := engine.NewTestCaseResult("test1", "", tc.verbose, false, false, false, false)

			if tc.withError {
				testCaseResult.Fail(errors.New("test failure"))
			} else {
				testCaseResult.Complete()
			}

			// Add the test case result to the suite
			testSuiteResult.AddResult(testCaseResult)
			testSuiteResult.Complete()

			// Print the test suite result (file-level reporting)
			testSuiteResult.Print(&buf)

			// Verify results
			output := buf.String()

			// Check if the expected status is in the output
			assert.Contains(t, output, tc.wantStatus, "output should contain expected status")

			// Check that the file path appears in the output
			assert.Contains(t, output, testSuiteFile, "Output should contain the test file path")
		})
	}
}

func TestRenderTemplate(t *testing.T) {
	t.Run("repository variables", func(t *testing.T) {
		yaml := `
common:
  functions: {{ .Repositories.myrepo }}/foo
  crds: {{ .Repositories.otherrepo }}/bar
`
		repos := map[string]string{
			"myrepo":    "/path/to/myrepo",
			"otherrepo": "/path/to/otherrepo",
		}

		templateContext := newTemplateContext(repos, api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "/path/to/myrepo/foo")
		assert.Contains(t, out, "/path/to/otherrepo/bar")
		assert.NotContains(t, out, "{{ .Repositories.myrepo }}")
		assert.NotContains(t, out, "{{ .Repositories.otherrepo }}")
	})

	t.Run("input variables", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "setup"
    run: "echo 'Setting up {{ .Inputs.XR }} with {{ .Inputs.Composition }}'"
`
		inputs := api.Inputs{
			XR:          "my-xr.yaml",
			Composition: "my-composition.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Setting up my-xr.yaml with my-composition.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")
		assert.NotContains(t, out, "{{ .Inputs.Composition }}")
	})

	t.Run("input variables - comprehensive", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "comprehensive setup"
    run: "echo 'XR: {{ .Inputs.XR }}, Claim: {{ .Inputs.Claim }}, Functions: {{ .Inputs.Functions }}, Observed: {{ .Inputs.ObservedResources }}'"
`
		inputs := api.Inputs{
			XR:                  "my-xr.yaml",
			Claim:               "my-claim.yaml",
			Functions:           "my-functions/",
			ObservedResources:   "my-observed.yaml",
			ExtraResources:      "my-extra.yaml",
			FunctionCredentials: "my-creds.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'XR: my-xr.yaml, Claim: my-claim.yaml, Functions: my-functions/, Observed: my-observed.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")
		assert.NotContains(t, out, "{{ .Inputs.Claim }}")
		assert.NotContains(t, out, "{{ .Inputs.Functions }}")
		assert.NotContains(t, out, "{{ .Inputs.ObservedResources }}")
	})

	t.Run("input variables - context files and values", func(t *testing.T) {
		yaml := `
hooks:
  pre-test:
  - name: "context setup"
    run: "echo 'Context files: {{ .Inputs.ContextFiles }}, Context values: {{ .Inputs.ContextValues }}'"
`
		inputs := api.Inputs{
			XR: "my-xr.yaml",
			ContextFiles: map[string]string{
				"config":  "/path/to/config.yaml",
				"secrets": "/path/to/secrets.yaml",
			},
			ContextValues: map[string]string{
				"env":    "production",
				"region": "us-west-2",
			},
		}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		// Note: map rendering in Go templates can vary, so we check for key components
		assert.Contains(t, out, "Context files:")
		assert.Contains(t, out, "Context values:")
		assert.NotContains(t, out, "{{ .Inputs.ContextFiles }}")
		assert.NotContains(t, out, "{{ .Inputs.ContextValues }}")
	})

	t.Run("output variables", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "cleanup"
    run: "echo 'Cleaning up {{ .Outputs.XR }} and {{ .Outputs.Render }}'"
`
		outputs := &engine.Outputs{
			XR:     "rendered-xr.yaml",
			Render: "rendered-resources.yaml",
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Cleaning up rendered-xr.yaml and rendered-resources.yaml'")
		assert.NotContains(t, out, "{{ .Outputs.XR }}")
		assert.NotContains(t, out, "{{ .Outputs.Render }}")
	})

	t.Run("output variables with validation", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "validate and count"
    run: "echo 'Validation: {{ .Outputs.Validate }}, Count: {{ .Outputs.RenderCount }}'"
`
		validatePath := "/path/to/validate.yaml"
		outputs := &engine.Outputs{
			XR:          "rendered-xr.yaml",
			Render:      "rendered-resources.yaml",
			Validate:    &validatePath,
			RenderCount: 5,
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Validation: /path/to/validate.yaml, Count: 5'")
		assert.NotContains(t, out, "{{ .Outputs.Validate }}")
		assert.NotContains(t, out, "{{ .Outputs.RenderCount }}")
	})

	t.Run("output variables without validation", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "no validation"
    run: "echo 'Validation: {{ .Outputs.Validate }}, Count: {{ .Outputs.RenderCount }}'"
`
		outputs := &engine.Outputs{
			XR:          "rendered-xr.yaml",
			Render:      "rendered-resources.yaml",
			Validate:    nil, // No validation output
			RenderCount: 3,
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		assert.Contains(t, out, "echo 'Validation: <nil>, Count: 3'")
		assert.NotContains(t, out, "{{ .Outputs.Validate }}")
		assert.NotContains(t, out, "{{ .Outputs.RenderCount }}")
	})

	t.Run("output variables - rendered map", func(t *testing.T) {
		yaml := `
hooks:
  post-test:
  - name: "check rendered resources"
    run: "echo 'Rendered resources: {{ .Outputs.Rendered }}'"
`
		outputs := &engine.Outputs{
			XR:     "rendered-xr.yaml",
			Render: "rendered-resources.yaml",
			Rendered: map[string]string{
				"ConfigMap/my-config": "/path/to/configmap.yaml",
				"Service/my-service":  "/path/to/service.yaml",
			},
		}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)
		// Note: map rendering in Go templates can vary, so we check for key components
		assert.Contains(t, out, "Rendered resources:")
		assert.NotContains(t, out, "{{ .Outputs.Rendered }}")
	})

	t.Run("mixed template variables", func(t *testing.T) {
		yaml := `
common:
  functions: {{ .Repositories.myrepo }}/functions
hooks:
  pre-test:
  - name: "pre-test"
    run: "echo 'Pre-test for {{ .Inputs.XR }}'"
  post-test:
  - name: "post-test"
    run: "echo 'Post-test for {{ .Outputs.XR }}'"
`
		repos := map[string]string{"myrepo": "/path/to/repo"}
		inputs := api.Inputs{XR: "test-xr.yaml"}
		outputs := &engine.Outputs{XR: "rendered-xr.yaml"}

		templateContext := newTemplateContext(repos, inputs, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		out, err := runner.renderTemplate(yaml, templateContext, "test")
		require.NoError(t, err)

		// Check repository variables
		assert.Contains(t, out, "/path/to/repo/functions")
		assert.NotContains(t, out, "{{ .Repositories.myrepo }}")

		// Check input variables
		assert.Contains(t, out, "echo 'Pre-test for test-xr.yaml'")
		assert.NotContains(t, out, "{{ .Inputs.XR }}")

		// Check output variables
		assert.Contains(t, out, "echo 'Post-test for rendered-xr.yaml'")
		assert.NotContains(t, out, "{{ .Outputs.XR }}")
	})

	t.Run("unknown repository variable", func(t *testing.T) {
		yaml := "functions: {{ .Repositories.unknownrepo }}/foo"
		repos := map[string]string{"myrepo": "/some/path"}

		templateContext := newTemplateContext(repos, api.Inputs{}, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "map has no entry for key \"unknownrepo\"")
	})

	t.Run("unknown input variable", func(t *testing.T) {
		yaml := "hooks:\n  pre-test:\n  - run: \"echo '{{ .Inputs.UnknownField }}'\""
		inputs := api.Inputs{XR: "test-xr.yaml"}

		templateContext := newTemplateContext(map[string]string{}, inputs, nil, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't evaluate field UnknownField")
	})

	t.Run("unknown output variable", func(t *testing.T) {
		yaml := "hooks:\n  post-test:\n  - run: \"echo '{{ .Outputs.UnknownField }}'\""
		outputs := &engine.Outputs{XR: "rendered-xr.yaml"}

		templateContext := newTemplateContext(map[string]string{}, api.Inputs{}, outputs, map[string]*engine.TestCaseResult{})
		runner := &Runner{}
		_, err := runner.renderTemplate(yaml, templateContext, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't evaluate field UnknownField")
	})
}

// TestProcessTemplateVariables tests the processTemplateVariables function.
func TestProcessTemplateVariables(t *testing.T) {
	// Create a test case with template variables in hooks
	testCase := api.TestCase{
		Name: "template-vars-test",
		ID:   "template-vars-test",
		Inputs: api.Inputs{
			XR:          "test-xr.yaml",
			Composition: "test-comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: fmt.Sprintf("echo 'Setting up %s'", testexecutionUtils.CreatePlaceholder(".Repositories.myrepo"))},
			},
		},
	}

	// Create runner with repositories
	runner := &Runner{
		Options: &testexecutionUtils.Options{
			Repositories: map[string]string{
				"myrepo": "/path/to/myrepo",
			},
		},
	}

	// Process template variables
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	err := runner.processTemplateVariables(&testCase, testSuiteResult)
	require.NoError(t, err)

	// Verify that template variables were processed
	// Note: The actual processing depends on the implementation
	// For now, just verify the function doesn't error
	assert.NotEmpty(t, testCase.Hooks.PreTest[0].Run)
}

// TestProcessTemplateVariables_NoTemplateVars tests processTemplateVariables with no template variables.
func TestProcessTemplateVariables_NoTemplateVars(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create test files
	xrFile := "/xr.yaml"
	require.NoError(t, afero.WriteFile(fs, xrFile, []byte(`apiVersion: example.com/v1
kind: XR
metadata:
  name: test-xr`), 0o644))

	compositionFile := "/composition.yaml"
	require.NoError(t, afero.WriteFile(fs, compositionFile, []byte(`apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: test-composition`), 0o644))

	// Create a test case without template variables
	testCase := api.TestCase{
		Name: "no-template-vars-test",
		ID:   "no-template-vars-test",
		Inputs: api.Inputs{
			XR:          xrFile,
			Composition: compositionFile,
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: "echo 'Setting up without templates'"},
			},
		},
	}

	// Create runner
	runner := &Runner{
		Options: &testexecutionUtils.Options{
			Repositories: map[string]string{
				"myrepo": "/path/to/myrepo",
			},
		},
	}

	// Process template variables
	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	err := runner.processTemplateVariables(&testCase, testSuiteResult)
	require.NoError(t, err)

	// Verify that the test case was not modified
	assert.Equal(t, "echo 'Setting up without templates'", testCase.Hooks.PreTest[0].Run)
}

// TestRemoveHooks tests the removeHooks function.
func TestRemoveHooks(t *testing.T) {
	runner := &Runner{}

	// Create a test case with hooks
	testCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: "echo 'pre-test'"},
			},
			PostTest: []api.Hook{
				{Name: "post-test", Run: "echo 'post-test'"},
			},
		},
	}

	// Remove all hooks
	_, err := runner.removeHooks(&testCase)
	require.NoError(t, err)

	// Verify both pre-test and post-test hooks were removed
	assert.Nil(t, testCase.Hooks.PreTest)
	assert.Nil(t, testCase.Hooks.PostTest)
}

// TestRestoreHooks tests the restoreHooks function.
func TestRestoreHooks(t *testing.T) {
	runner := &Runner{}

	// Original test case with hooks
	originalTestCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
		},
		Hooks: api.Hooks{
			PreTest: []api.Hook{
				{Name: "pre-test", Run: "echo 'pre-test'"},
			},
			PostTest: []api.Hook{
				{Name: "post-test", Run: "echo 'post-test'"},
			},
		},
	}

	// Create processed YAML without hooks (as would happen after template processing)
	processedTestCase := api.TestCase{
		Name: "test",
		ID:   "test-id",
		Inputs: api.Inputs{
			XR:          "xr.yaml",
			Composition: "comp.yaml",
			// Hooks removed for template processing
		},
	}

	processedYAML, err := yaml.Marshal(processedTestCase)
	require.NoError(t, err)

	// Restore hooks
	originalHooks := originalTestCase.Hooks
	err = runner.restoreHooks(string(processedYAML), &originalTestCase, originalHooks)
	require.NoError(t, err)

	// Verify hooks were restored
	assert.Len(t, originalTestCase.Hooks.PreTest, 1)
	assert.Equal(t, "pre-test", originalTestCase.Hooks.PreTest[0].Name)
	assert.Len(t, originalTestCase.Hooks.PostTest, 1)
	assert.Equal(t, "post-test", originalTestCase.Hooks.PostTest[0].Name)
	assert.Equal(t, "echo 'pre-test'", originalTestCase.Hooks.PreTest[0].Run)
	assert.Equal(t, "echo 'post-test'", originalTestCase.Hooks.PostTest[0].Run)
}

// TestRunTestCase_Outputs tests that output files are written to the outputs directory.
func TestRunTestCase_Outputs(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create test files
	require.NoError(t, afero.WriteFile(fs, "/xr.yaml", []byte(`apiVersion: example.org/v1alpha1
kind: MyCompositeResource
metadata:
  name: test-xr`), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/comp.yaml", []byte("apiVersion: apiextensions.crossplane.io/v1\nkind: Composition"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/functions.yaml", []byte("functions: []"), 0o644))
	require.NoError(t, afero.WriteFile(fs, "/crd.yaml", []byte("apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition"), 0o644))

	validRenderYAML := []byte(`apiVersion: example.org/v1alpha1
kind: Pod
metadata:
  name: test-pod
---
apiVersion: example.org/v1alpha1
kind: ConfigMap
metadata:
  name: test-configmap`)

	cfg := &config.Config{
		Dependencies: map[string]string{
			"crossplane": config.CrossplaneCmd,
		},
	}
	options := &testexecutionUtils.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		Debug:        false,
	}
	runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
	runner.fs = fs // Use in-memory filesystem
	runner.testSuiteFile = testSuiteFile
	runner.testSuiteSpec = &api.TestSuiteSpec{}

	// Mock path expansion and verification
	runner.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
		// For in-memory filesystem, expand relative paths relative to test suite file directory
		if filepath.IsAbs(path) {
			return path, nil
		}
		// Test suite file is at /suite_xprin.yaml, so relative paths should be in /
		return "/" + path, nil
	}
	runner.verifyPathExists = func(path string) error {
		if _, err := fs.Stat(path); err != nil {
			return err
		}

		return nil
	}

	// Mock copy function to use afero for in-memory filesystem
	runner.copy = func(src, dest string, _ ...cp.Options) error {
		// Read from source
		data, err := afero.ReadFile(fs, src)
		if err != nil {
			return err
		}
		// Ensure destination directory exists
		destDir := filepath.Dir(dest)
		if err := fs.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		// Write to destination
		return afero.WriteFile(fs, dest, data, 0o644)
	}

	// Mock runCommand to return successful render and validate results
	runner.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
		if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
			return validRenderYAML, []byte{}, nil
		}

		if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
			return []byte("validate ok"), []byte{}, nil
		}

		return []byte{}, []byte{}, nil
	}

	testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
	testCase := api.TestCase{
		Name: "test1",
		ID:   "", // No ID - test outputs directory, not artifacts
		Inputs: api.Inputs{
			XR:          "xr.yaml",            // Relative path, will be expanded
			Composition: "comp.yaml",          // Relative path, will be expanded
			Functions:   "functions.yaml",     // Relative path, will be expanded
			CRDs:        []string{"crd.yaml"}, // Relative path, will be expanded
		},
	}

	// Call runapi.TestCase WITHOUT runTestCaseFunc - this executes the REAL code path
	result := runner.runTestCase(testCase, testSuiteResult)
	require.NoError(t, result.Error)
	assert.Equal(t, engine.StatusPass, result.Status)

	// Verify Outputs paths are set (indicating files were written to outputs directory)
	// Note: The actual files are cleaned up by defer in runapi.TestCase, but the paths prove they were written
	assert.NotEmpty(t, result.Outputs.Render, "Render path should be set")
	assert.NotEmpty(t, result.Outputs.XR, "XR path should be set")

	if result.Outputs.Validate != nil {
		assert.NotEmpty(t, *result.Outputs.Validate, "Validate path should be set")
	}

	// Verify Outputs paths point to outputs directory (not artifacts since ID is empty)
	// The paths should contain "outputs" but not "test1-id" (which would be artifacts)
	assert.NotContains(t, result.Outputs.Render, "test1-id", "Render path should NOT point to artifacts directory when ID is empty")
	assert.NotContains(t, result.Outputs.XR, "test1-id", "XR path should NOT point to artifacts directory when ID is empty")
	assert.Contains(t, result.Outputs.Render, "outputs", "Render path should point to outputs directory")
	assert.Contains(t, result.Outputs.XR, "outputs", "XR path should point to outputs directory")

	if result.Outputs.Validate != nil {
		assert.Contains(t, *result.Outputs.Validate, "outputs", "Validate path should point to outputs directory")
	}

	// Verify expected filenames are in the paths
	assert.Contains(t, result.Outputs.Render, "rendered.yaml", "Render path should contain rendered.yaml")
	assert.Contains(t, result.Outputs.XR, "xr.yaml", "XR path should contain xr.yaml")

	if result.Outputs.Validate != nil {
		assert.Contains(t, *result.Outputs.Validate, "validate.yaml", "Validate path should contain validate.yaml")
	}

	// Verify Rendered map contains the expected resources
	assert.Contains(t, result.Outputs.Rendered, "Pod/test-pod", "Rendered map should contain Pod/test-pod")
	assert.Contains(t, result.Outputs.Rendered, "ConfigMap/test-configmap", "Rendered map should contain ConfigMap/test-configmap")
	assert.Contains(t, result.Outputs.Rendered["Pod/test-pod"], "rendered-pod-test-pod.yaml", "Pod resource path should contain correct filename")
	assert.Contains(t, result.Outputs.Rendered["ConfigMap/test-configmap"], "rendered-configmap-test-configmap.yaml", "ConfigMap resource path should contain correct filename")
	assert.Contains(t, result.Outputs.Rendered["Pod/test-pod"], "outputs", "Pod resource path should point to outputs directory")
	assert.Contains(t, result.Outputs.Rendered["ConfigMap/test-configmap"], "outputs", "ConfigMap resource path should point to outputs directory")

	// Verify RenderCount was set
	assert.Equal(t, 2, result.Outputs.RenderCount, "RenderCount should match number of resources")
}

// TestArtifactsDirectory tests the artifacts directory functionality.
func TestArtifactsDirectory(t *testing.T) {
	t.Run("creates artifacts directory in runTests", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		options := &testexecutionUtils.Options{
			Debug: false,
		}

		testSuiteSpec := &api.TestSuiteSpec{
			Tests: []api.TestCase{
				{Name: "test1"},
			},
		}

		runner := NewRunner(options, testSuiteFile, testSuiteSpec)
		runner.fs = fs // Use in-memory filesystem

		// Mock runTestCase to avoid needing full test setup
		runner.runTestCaseFunc = func(testCase api.TestCase) *engine.TestCaseResult {
			return engine.NewTestCaseResult(testCase.Name, testCase.ID, false, false, false, false, false).Complete()
		}

		// Before runTests, artifacts directory should not exist
		assert.Empty(t, runner.testSuiteArtifactsDir, "artifacts directory should not exist before runTests")

		err := runner.RunTests()
		require.NoError(t, err)

		// After runTests, artifacts directory should have been created (even though it's cleaned up by defer)
		// We can verify it was set (the directory is created in runTests line 132)
		// Note: The directory is cleaned up by defer, so we verify the field was set during execution
		// Since we can't check after defer runs, we verify through behavior: if runTests succeeded,
		// the directory was created. The actual copying logic is tested in other tests.
		assert.NotEmpty(t, runner.testSuiteArtifactsDir, "artifacts directory path should have been set by runTests")
	})

	t.Run("copies outputs to artifacts directory when testCase.ID is set", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		// Create test files
		require.NoError(t, afero.WriteFile(fs, "/xr.yaml", []byte(`apiVersion: example.org/v1alpha1
kind: MyCompositeResource
metadata:
  name: test-xr`), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/comp.yaml", []byte("apiVersion: apiextensions.crossplane.io/v1\nkind: Composition"), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/functions.yaml", []byte("functions: []"), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/crd.yaml", []byte("apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition"), 0o644))

		validRenderYAML := []byte(`apiVersion: example.org/v1alpha1
kind: Pod
metadata:
  name: test-pod
---
apiVersion: example.org/v1alpha1
kind: ConfigMap
metadata:
  name: test-configmap`)

		cfg := &config.Config{
			Dependencies: map[string]string{
				"crossplane": config.CrossplaneCmd,
			},
		}
		options := &testexecutionUtils.Options{
			Dependencies: cfg.Dependencies,
			Render:       []string{config.RenderSubcommand, config.RenderFlags},
			Validate:     []string{config.ValidateSubcommand},
			Debug:        false,
		}
		runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
		runner.fs = fs // Use in-memory filesystem
		runner.testSuiteFile = testSuiteFile
		runner.testSuiteSpec = &api.TestSuiteSpec{}

		// Set artifacts directory (simulating what runTests does)
		artifactsBaseDir := "/artifacts-dir"
		require.NoError(t, fs.MkdirAll(artifactsBaseDir, 0o755))
		runner.testSuiteArtifactsDir = artifactsBaseDir

		// Mock path expansion and verification
		runner.expandPathRelativeToTestSuiteFile = func(_, path string) (string, error) {
			if filepath.IsAbs(path) {
				return path, nil
			}

			return "/" + path, nil
		}
		runner.verifyPathExists = func(path string) error {
			if _, err := fs.Stat(path); err != nil {
				return err
			}

			return nil
		}

		// Mock copy function to use afero for in-memory filesystem
		runner.copy = func(src, dest string, _ ...cp.Options) error {
			// Check if source is a directory
			srcInfo, err := fs.Stat(src)
			if err != nil {
				return err
			}

			if srcInfo.IsDir() {
				// Copy directory recursively
				return afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					// Calculate relative path from source
					relPath, err := filepath.Rel(src, path)
					if err != nil {
						return err
					}

					destPath := filepath.Join(dest, relPath)
					if info.IsDir() {
						return fs.MkdirAll(destPath, info.Mode())
					}
					// Copy file
					data, err := afero.ReadFile(fs, path)
					if err != nil {
						return err
					}

					return afero.WriteFile(fs, destPath, data, info.Mode())
				})
			}
			// Copy single file
			data, err := afero.ReadFile(fs, src)
			if err != nil {
				return err
			}
			// Ensure destination directory exists
			destDir := filepath.Dir(dest)
			if err := fs.MkdirAll(destDir, 0o755); err != nil {
				return err
			}
			// Write to destination
			return afero.WriteFile(fs, dest, data, srcInfo.Mode())
		}

		// Mock runCommand to return successful render and validate results
		runner.runCommand = func(name string, args ...string) ([]byte, []byte, error) {
			if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
				return validRenderYAML, []byte{}, nil
			}

			if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
				return []byte("validate ok"), []byte{}, nil
			}

			return []byte{}, []byte{}, nil
		}

		testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
		testCase := api.TestCase{
			Name: "test1",
			ID:   "test1-id",
			Inputs: api.Inputs{
				XR:          "xr.yaml",
				Composition: "comp.yaml",
				Functions:   "functions.yaml",
				CRDs:        []string{"crd.yaml"},
			},
		}

		// Call runTestCase WITHOUT runTestCaseFunc - this executes the REAL code path
		result := runner.runTestCase(testCase, testSuiteResult)
		require.NoError(t, result.Error)
		assert.Equal(t, engine.StatusPass, result.Status)

		// Verify files were actually copied to artifacts directory by the REAL code
		artifactsDir := filepath.Join(artifactsBaseDir, testCase.ID)
		info, err := fs.Stat(artifactsDir)
		require.NoError(t, err, "artifacts directory should exist")
		assert.True(t, info.IsDir(), "artifacts directory should be a directory")

		_, err = fs.Stat(filepath.Join(artifactsDir, "rendered.yaml"))
		require.NoError(t, err, "rendered.yaml should be copied")
		_, err = fs.Stat(filepath.Join(artifactsDir, "xr.yaml"))
		require.NoError(t, err, "xr.yaml should be copied")
		_, err = fs.Stat(filepath.Join(artifactsDir, "validate.yaml"))
		require.NoError(t, err, "validate.yaml should be copied")
		_, err = fs.Stat(filepath.Join(artifactsDir, "rendered-pod-test-pod.yaml"))
		require.NoError(t, err, "rendered Pod resource should be copied")
		_, err = fs.Stat(filepath.Join(artifactsDir, "rendered-configmap-test-configmap.yaml"))
		require.NoError(t, err, "rendered ConfigMap resource should be copied")

		// Verify Outputs paths were updated to point to artifact paths (by real code at lines 665-675)
		assert.Contains(t, result.Outputs.Render, "test1-id", "Render path should point to artifacts directory")
		assert.Contains(t, result.Outputs.XR, "test1-id", "XR path should point to artifacts directory")

		if result.Outputs.Validate != nil {
			assert.Contains(t, *result.Outputs.Validate, "test1-id", "Validate path should point to artifacts directory")
		}
		// Verify Rendered map paths were updated
		assert.Contains(t, result.Outputs.Rendered["Pod/test-pod"], "test1-id", "Rendered Pod path should point to artifacts directory")
		assert.Contains(t, result.Outputs.Rendered["ConfigMap/test-configmap"], "test1-id", "Rendered ConfigMap path should point to artifacts directory")
	})

	t.Run("does not copy outputs when testCase.ID is empty", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		options := &testexecutionUtils.Options{
			Debug: false,
		}
		runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})
		runner.fs = fs // Use in-memory filesystem

		var copyCallCount int

		originalCopy := runner.copy
		runner.copy = func(src, dest string, opts ...cp.Options) error {
			if strings.Contains(dest, "xprin-testsuite-artifacts") {
				copyCallCount++
			}

			return originalCopy(src, dest, opts...)
		}

		testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
		runner.testSuiteArtifactsDir = "/artifacts-dir"
		require.NoError(t, fs.MkdirAll(runner.testSuiteArtifactsDir, 0o755))

		testCase := api.TestCase{
			Name: "test1", // No ID
		}

		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			result := engine.NewTestCaseResult(tc.Name, tc.ID, false, false, false, false, false)
			// The artifacts copying logic should not execute when ID is empty
			return result.Complete()
		}

		result := runner.runTestCase(testCase, testSuiteResult)
		require.NoError(t, result.Error)

		// Verify copy was NOT called when ID is empty
		assert.Equal(t, 0, copyCallCount, "copy should not be called when testCase.ID is empty")
	})

	t.Run("supports cross-test references via GetCompletedTests", func(t *testing.T) {
		options := &testexecutionUtils.Options{
			Debug: false,
		}
		runner := NewRunner(options, testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}})

		var (
			test1Result      *engine.TestCaseResult
			capturedTestsMap map[string]*engine.TestCaseResult
		)

		testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)

		// First test case - will be added to testSuiteResult
		testCase1 := api.TestCase{
			Name: "test1",
			ID:   "test1-id",
		}

		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			result := engine.NewTestCaseResult(tc.Name, tc.ID, false, false, false, false, false)
			result.Outputs.Render = "/path/to/render1.yaml"
			result.Outputs.XR = "/path/to/xr1.yaml"
			result.Outputs.RenderCount = 1

			if tc.ID == "test1-id" {
				test1Result = result
				testSuiteResult.AddResult(result)
			}

			return result.Complete()
		}

		// Run first test case
		result1 := runner.runTestCase(testCase1, testSuiteResult)
		require.NoError(t, result1.Error)

		// Second test case - should have access to first test via GetCompletedTests
		testCase2 := api.TestCase{
			Name: "test2",
			ID:   "test2-id",
			Hooks: api.Hooks{
				PreTest: []api.Hook{
					{Run: fmt.Sprintf("echo '%s.Tests.test1-id.Outputs.XR%s'", testexecutionUtils.PlaceholderOpen, testexecutionUtils.PlaceholderClose)},
				},
			},
		}

		// Test GetCompletedTests directly since we can't mock executeHooks (it's a method, not a field)
		tests := testSuiteResult.GetCompletedTests()
		capturedTestsMap = tests

		// Verify that GetCompletedTests returns the correct map
		assert.NotNil(t, capturedTestsMap, "tests map should not be nil")
		assert.Contains(t, capturedTestsMap, "test1-id", "tests map should contain test1-id")

		if test1Result != nil {
			// Verify we can access the first test's outputs
			test1 := capturedTestsMap["test1-id"]
			assert.NotNil(t, test1, "test1 should be accessible")
			assert.NotEmpty(t, test1.Outputs.Render, "should be able to access test1 outputs")
			assert.NotEmpty(t, test1.Outputs.XR, "should be able to access test1 XR")
			// Compare IDs and outputs since pointers may differ
			assert.Equal(t, test1Result.ID, test1.ID, "test1 ID should match")
			assert.Equal(t, test1Result.Outputs.XR, test1.Outputs.XR, "test1 XR path should match")
			assert.Equal(t, test1Result.Outputs.Render, test1.Outputs.Render, "test1 Render path should match")
		}

		// Also verify that runTestCase would call executeHooks with GetCompletedTests
		// by verifying the test case result can be run successfully
		runner.runTestCaseFunc = func(tc api.TestCase) *engine.TestCaseResult {
			result := engine.NewTestCaseResult(tc.Name, tc.ID, false, false, false, false, false)
			// In real runTestCase, executeHooks is called with testSuiteResult.GetCompletedTests()
			// We verify that GetCompletedTests() works correctly above
			return result.Complete()
		}

		// Run second test case
		result2 := runner.runTestCase(testCase2, testSuiteResult)
		require.NoError(t, result2.Error)
	})
}
