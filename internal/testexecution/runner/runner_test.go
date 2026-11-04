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
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/config"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/crossplane-contrib/xprin/internal/templates/placeholder"
	testexecutionOptions "github.com/crossplane-contrib/xprin/internal/testexecution/options"
	"github.com/crossplane-contrib/xprin/internal/testsuite"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
	cp "github.com/otiai10/copy"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const testSuiteFile = "/suite_xprin.yaml"

// newMockRunner creates a test-specific Runner with mocked functions.
func newMockRunner(options *testexecutionOptions.Options, mocks ...func(*Runner)) *Runner {
	options.DebugLog = noDebug

	r := NewRunner(options)

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
func makeOptions(cfg *config.Config, render, validate []string, overrides ...func(*testexecutionOptions.Options)) *testexecutionOptions.Options {
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
		validate = strings.Fields(config.ValidateSubcommand + " " + config.ValidateFlags)
	}

	opt := &testexecutionOptions.Options{
		Dependencies: cfg.Dependencies,
		Render:       render,
		Validate:     validate,
		XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
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
	options := makeOptions(cfg, nil, nil, func(o *testexecutionOptions.Options) {
		o.Debug = true
	})
	runner := NewRunner(options)

	// Check the runner was created with the right settings
	assert.Equal(t, options, runner.Options)
	assert.True(t, runner.ShowRender)
	assert.True(t, runner.ShowValidate)
	assert.True(t, runner.ShowHooks)
	assert.True(t, runner.Verbose)
	assert.True(t, runner.Debug)
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
	runner := NewRunner(options)

	// The runner should have the default Render/Validate commands (from runner or config loader)
	assert.NotEmpty(t, runner.Render)
	assert.NotEmpty(t, runner.Validate)
	// Optionally, check for specific defaults if known, e.g.:
	assert.Equal(t, strings.Fields(config.DefaultRenderCmd), runner.Render)
	assert.Equal(t, strings.Fields(config.ValidateSubcommand+" "+config.ValidateFlags), runner.Validate)
}

func TestNewRunner_RunsCommandsInTheGivenDirectory(t *testing.T) {
	// The runner is for the whole invocation, so the directory is the one of the testsuite file of each call
	dir := t.TempDir()
	want, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)

	out, err := NewRunner(&testexecutionOptions.Options{}).runCommand(dir, "pwd")

	require.NoError(t, err)

	got, err := filepath.EvalSymlinks(strings.TrimSpace(string(out))) // pwd can print the path with or without a symlink
	require.NoError(t, err)
	assert.Equal(t, want, got)
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
	options := &testexecutionOptions.Options{
		Dependencies: cfg.Dependencies,
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return []byte("render fail"), fmt.Errorf("fail")
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Render failure: Error is nil, failure shown in section
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Validate failure: Error is nil, failure shown in section
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}
					// Mock successful hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup executed"), nil
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Validate failure: Error is nil, failure shown in section
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate fail"), fmt.Errorf("fail")
					}
					// Mock failing hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("cleanup\nhook failed"), errors.New("exit status 1")
					}

					return validRenderYAML, nil
				}
			},
			wantError: "", // Both validate and post-test fail: Error is nil, failures in sections
		},
		{
			name: "post-test hook template failure shows root cause",
			testCase: api.TestCase{
				Name: "test",
				Inputs: api.Inputs{
					XR:          "xr.yaml",
					Composition: "comp.yaml",
					Functions:   "functions.yaml",
				},
				Hooks: api.Hooks{
					PostTest: []api.Hook{
						{
							Name: "templated-cleanup",
							Run:  fmt.Sprintf("echo 'value: %s.Outputs.UnknownField%s'", placeholder.Open, placeholder.Close),
						},
					},
				},
			},
			setup: func(r *Runner) {
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Pipeline failure only: failure is shown in post-test hooks section
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
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
				r.convertClaimToXRFunc = func(_ *Runner, _, _, _, outputPath string) (string, error) {
					return filepath.Join(outputPath, "xr.yaml"), nil
				}
				// Mock the runCommand function to return success for all operations
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}

					return []byte{}, nil
				}
			},
			wantError: "",
		},
		// Connection Secret
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("pre-test setup"), nil
					}

					return []byte{}, nil
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("post-test cleanup"), nil
					}

					return []byte{}, nil
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock hook execution
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						return []byte("hook output"), nil
					}

					return []byte{}, nil
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock failing hook execution (use real *exec.ExitError so output shows [x])
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						err := exec.Command("sh", "-c", "exit 1").Run()
						return []byte("failing setup\nhook failed"), err
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Pre-test failure: Error is nil, failure shown in section
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
				r.runCommand = func(_, name string, args ...string) ([]byte, error) {
					if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
						return validRenderYAML, nil
					}

					if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
						return []byte("validate ok"), nil
					}
					// Mock failing hook execution (use real *exec.ExitError so output shows [x])
					if name == "sh" && len(args) > 0 && args[0] == "-c" {
						err := exec.Command("sh", "-c", "exit 1").Run()
						return []byte("failing cleanup\nhook failed"), err
					}

					return []byte{}, nil
				}
			},
			wantError: "", // Post-test failure: Error is nil, failure shown in section
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new test runner for each test case
			testRunner := newMockRunner(options)
			testRunner.fs = fs // Use in-memory filesystem

			// Apply any custom mocks from the test case setup
			if tc.setup != nil {
				tc.setup(testRunner)
			}

			testSuiteResult := engine.NewTestSuiteResult("test-suite.yaml", false)
			env := newTestEnv()
			env.Index = 1
			result := testRunner.RunTestCase(env.TestSuiteContext, tc.testCase, testSuiteResult.GetCompletedTests())

			if tc.wantError != "" {
				assert.Equal(t, engine.StatusFail(), result.Status)
				require.Error(t, result.Error)
				assert.Contains(t, result.Error.Error(), tc.wantError)
			} else if result.Status != engine.StatusFail() || result.Error != nil {
				assert.Equal(t, engine.StatusPass(), result.Status)
				require.NoError(t, result.Error)
			}
			// else: pipeline failure only (StatusFail, Error == nil); failure is shown in sections

			// For tests with post-test hooks, verify they ran
			if tc.name == "validate fails but post-test hooks run" {
				assert.Len(t, result.PostTestHooksResults, 1)
				assert.Equal(t, "cleanup", result.PostTestHooksResults[0].Name)
				require.NoError(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")
			}

			// Pipeline-only failures: verify failure is in formatted sections
			if tc.name == "render fails" {
				assert.Contains(t, result.FormattedRenderOutput, "render fail")
			}

			if tc.name == "validate fails" {
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")
			}

			if tc.name == "xr with failing pre-test hook" {
				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "[x] failing-setup")
			}

			if tc.name == "xr with failing post-test hook" {
				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "[x] failing-cleanup")
			}

			if tc.name == "both validate and post-test hook fail" {
				assert.Len(t, result.PostTestHooksResults, 1)
				require.Error(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.FormattedValidateOutput, "validate fail")

				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "Post-test Hooks")
			}

			if tc.name == "post-test hook template failure shows root cause" {
				assert.Len(t, result.PostTestHooksResults, 1)
				require.Error(t, result.PostTestHooksResults[0].Error)
				assert.Contains(t, result.PostTestHooksResults[0].Error.Error(), "failed to render hook template")
				assert.Contains(t, result.PostTestHooksResults[0].Error.Error(), "can't evaluate field UnknownField")

				var buf bytes.Buffer
				result.Print(&buf)
				assert.Contains(t, buf.String(), "Post-test Hooks:")
				assert.Contains(t, buf.String(), "[!] templated-cleanup")
				assert.Contains(t, buf.String(), "error: failed to render hook template")
				assert.Contains(t, buf.String(), "can't evaluate field UnknownField")
				assert.NotContains(t, buf.String(), "exit code")
			}
		})
	}
}

// The common inputs of the testsuite file are merged into the test case before it runs: the inputs it does not set
// come from them, and the ones it sets are kept. How each input is merged is tested in api.
func TestRunTestCase_MergesTheCommonInputs(t *testing.T) {
	common := api.Common{
		Inputs: api.Inputs{
			Composition:         "common_comp.yaml",
			Functions:           "common_functions.yaml",
			ContextFiles:        map[string]string{"ctx": "common_context.yaml"},
			ObservedResources:   "common_observed.yaml",
			ExtraResources:      "common_extra.yaml",
			FunctionCredentials: "common_credentials.yaml",
		},
		Patches: api.Patches{XRD: "common_xrd.yaml"},
	}

	// What render gets after the functions, from the common inputs
	wantCommon := []string{
		"common_xrd.yaml",
		"--context-files", "common_context.yaml",
		"--observed-resources", "common_observed.yaml",
		"--extra-resources", "common_extra.yaml",
		"--function-credentials", "common_credentials.yaml",
	}

	for name, tt := range map[string]struct {
		testCase api.TestCase
		want     []string // the XR, the composition and the functions that render gets
	}{
		"the inputs that the test case does not set come from the common inputs": {
			testCase: api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml"}},
			want:     []string{"xr.yaml", "common_comp.yaml", "common_functions.yaml"},
		},
		"the inputs that the test case sets are kept": {
			testCase: api.TestCase{Name: "test", Inputs: api.Inputs{XR: "xr.yaml", Composition: "own_comp.yaml"}},
			want:     []string{"xr.yaml", "own_comp.yaml", "common_functions.yaml"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			runner := newMockRunner(&testexecutionOptions.Options{
				Dependencies: map[string]string{"crossplane": config.CrossplaneCmd},
				Render:       []string{config.RenderSubcommand, config.RenderFlags},
				XPCLI:        xpcli.InitXPCLI(xpcli.TierV2),
			})
			runner.fs = afero.NewMemMapFs()

			var renderArgs []string

			runner.runCommand = func(_, _ string, args ...string) ([]byte, error) {
				if len(args) > 0 && args[0] == config.RenderSubcommand {
					renderArgs = args

					return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), nil
				}

				return []byte{}, nil
			}

			env := testEnv(testsuite.New(testSuiteFile, &api.TestSuiteSpec{Common: common}))
			env.Index = 1
			result := runner.RunTestCase(env.TestSuiteContext, tt.testCase, nil)

			require.Equal(t, engine.StatusPass(), result.Status)
			require.GreaterOrEqual(t, len(renderArgs), 3)

			baseNames := make([]string, 0, len(renderArgs))
			for _, arg := range renderArgs {
				baseNames = append(baseNames, filepath.Base(arg))
			}

			// After the render command and its flags, and with only the file names of the copies in the temporary directory
			assert.Equal(t, append(tt.want, wantCommon...), baseNames[2:])
		})
	}
}

// A test case that the mode has no use for is skipped, whatever it needs, and none of its phases runs.
func TestRunTestCase_ASkippedByTheMode(t *testing.T) {
	spec := &api.TestSuiteSpec{Tests: []api.TestCase{{Name: "plain", ID: "plain"}}}
	r := NewRunner(&testexecutionOptions.Options{UpdateGoldens: true})
	r.runTestCaseFunc = func(api.TestCase) *engine.TestCaseResult {
		t.Fatal("a test case that the mode skips must not run")

		return nil
	}

	result := r.executeTestCase(TestSuiteContext{Index: 1, TestSuite: testsuite.New(testSuiteFile, spec)}, spec.Tests[0], nil)

	assert.Equal(t, engine.StatusSkip(), result.Status)
}

func TestRunTestCase_UsesTestResultWithStartTime(t *testing.T) {
	options := &testexecutionOptions.Options{
		Render:       []string{config.RenderSubcommand, config.RenderFlags},
		Validate:     []string{config.ValidateSubcommand},
		ShowRender:   true,
		ShowValidate: true,
		Verbose:      false,
		Debug:        false,
	}
	runner := newMockRunner(options)

	// Mock runCommand to return successful results
	runner.runCommand = func(_, name string, args ...string) ([]byte, error) {
		if name == config.CrossplaneCmd && len(args) > 0 && args[0] == config.RenderSubcommand {
			return []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n"), nil
		}

		if name == config.CrossplaneCmd && len(args) > 1 && args[0] == config.ValidateSubcommand {
			return []byte("validate ok"), nil
		}

		return []byte{}, nil
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
	env := newTestEnv()
	env.Index = 1
	result := runner.RunTestCase(env.TestSuiteContext, testCase, testSuiteResult.GetCompletedTests())
	assert.Equal(t, engine.StatusPass(), result.Status)
	require.NoError(t, result.Error)
	assert.False(t, result.StartTime.IsZero(), "TestCaseResult should have StartTime set")
}

// newTestEnv is the env of a test case of a testsuite file without test cases, which is the default of the tests that
// don't need a specific one.
func newTestEnv() *testCaseEnv {
	return testEnv(testsuite.New(testSuiteFile, &api.TestSuiteSpec{Tests: []api.TestCase{}}))
}

// testEnv is the env of a test case of testSuite, for the tests of the parts that take it.
func testEnv(testSuite *testsuite.TestSuite) *testCaseEnv {
	return &testCaseEnv{TestSuiteContext: TestSuiteContext{TestSuite: testSuite}}
}
