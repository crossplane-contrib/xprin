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

package test

import (
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

// TestNewOperation tests that NewOperation correctly initializes an Operation struct with the given config.
func TestCmd_Run(t *testing.T) {
	cfg := &internalcfg.Config{
		Dependencies: map[string]string{"dep1": "value1"},
		Repositories: map[string]string{"repo1": "path1"},
	}

	cmd := &Cmd{
		Targets: []string{},
		Config:  cfg,
	}

	// Test that Run method works
	ctx := &kong.Context{}
	err := cmd.Run(ctx)
	// Should not error even with empty targets
	assert.NoError(t, err)
}

// TestRun_WarningWithoutVerbose tests that a warning is printed when show-render flag is used without verbose.
func TestRun_WarningWithoutVerbose(t *testing.T) {
	// Setup test with properly initialized config
	cfg := &internalcfg.Config{
		Subcommands: &internalcfg.Subcommands{},
	}

	cmd := &Cmd{
		ShowRender:     true,
		ShowValidate:   true,
		ShowHooks:      true,
		ShowAssertions: true,
		Verbose:        false,
		Config:         cfg,
	}

	// Capture stderr during execution
	output := unittestsUtils.CaptureStderr(func() {
		ctx := &kong.Context{}
		err := cmd.Run(ctx)
		// We don't care about the error as long as we check the warning was output
		_ = err
	})

	// Verify the warning was output
	assert.Contains(t, output, "--show-render requires -v", "Should show warning when render flag is used without verbose")
	assert.Contains(t, output, "--show-validate requires -v", "Should show warning when validate flag is used without verbose")
	assert.Contains(t, output, "--show-hooks requires -v", "Should show warning when hooks flag is used without verbose")
	assert.Contains(t, output, "--show-assertions requires -v", "Should show warning when assertions flag is used without verbose")
}

// TestRun_NoWarningWithVerbose tests that no warning is printed when show-render flag is used with verbose.
func TestRun_NoWarningWithVerbose(t *testing.T) {
	// Setup test with properly initialized config
	cfg := &internalcfg.Config{
		Subcommands: &internalcfg.Subcommands{},
	}

	cmd := &Cmd{
		ShowRender:     true,
		ShowValidate:   true,
		ShowHooks:      true,
		ShowAssertions: true,
		Verbose:        true,
		Config:         cfg,
	}

	// Capture stderr during execution
	output := unittestsUtils.CaptureStderr(func() {
		ctx := &kong.Context{}
		err := cmd.Run(ctx)
		// We don't care about the error as long as we check the warning was output
		_ = err
	})

	// Verify no warning was output
	assert.NotContains(t, output, "--show-render requires -v", "Should not show warning when render flag is used with verbose")
	assert.NotContains(t, output, "--show-validate requires -v", "Should not show warning when validate flag is used with verbose")
	assert.NotContains(t, output, "--show-hooks requires -v", "Should not show warning when hooks flag is used with verbose")
	assert.NotContains(t, output, "--show-assertions requires -v", "Should not show warning when assertions flag is used with verbose")
}

func TestNewOptions(t *testing.T) {
	// Setup a config with specific render, validate and dependency values
	cfg := &internalcfg.Config{
		Dependencies: map[string]string{
			"crossplane": "custom-crossplane-path",
			"other-dep":  "other-path",
		},
		Subcommands: &internalcfg.Subcommands{
			Render:   "custom-render --flag1 --flag2",
			Validate: "custom-validate --flag3",
		},
		Repositories: map[string]string{
			"repo1": "path1",
			"repo2": "path2",
		},
	}

	// Create a test command
	cmd := &Cmd{
		ShowRender:     true,
		ShowValidate:   true,
		ShowHooks:      true,
		ShowAssertions: true,
		Verbose:        true,
		Debug:          false,
	}

	// Create options using the newOptions method
	options := cmd.newOptions(cfg)

	// Verify options were set from config
	assert.Equal(t, strings.Fields("custom-render --flag1 --flag2"), options.Render)
	assert.Equal(t, strings.Fields("custom-validate --flag3"), options.Validate)
	assert.Equal(t, cfg.Dependencies, options.Dependencies)
	assert.Equal(t, cfg.Repositories, options.Repositories)

	// Verify other options were set from command
	assert.Equal(t, cmd.ShowRender, options.ShowRender)
	assert.Equal(t, cmd.ShowValidate, options.ShowValidate)
	assert.Equal(t, cmd.ShowHooks, options.ShowHooks)
	assert.Equal(t, cmd.ShowAssertions, options.ShowAssertions)
	assert.Equal(t, cmd.Verbose, options.Verbose)
	assert.Equal(t, cmd.Debug, options.Debug)
}

// Test that NewOptions handles nil Subcommands gracefully.
func TestNewOptions_WithNilSubcommands(t *testing.T) {
	// Setup a config with nil Subcommands
	cfg := &internalcfg.Config{
		Dependencies: map[string]string{
			"crossplane": "path-to-crossplane",
		},
		Subcommands: nil, // explicitly nil
		Repositories: map[string]string{
			"repo1": "path1",
			"repo2": "path2",
		},
	}

	cmd := &Cmd{
		ShowRender:   true,
		ShowValidate: true,
		ShowHooks:    true,
		Verbose:      true,
	}

	// This should not panic
	options := cmd.newOptions(cfg)

	// Verify options were set correctly
	assert.Empty(t, options.Render)
	assert.Empty(t, options.Validate)
	assert.Equal(t, cfg.Dependencies, options.Dependencies)
	assert.Equal(t, cfg.Repositories, options.Repositories)
	assert.Equal(t, cmd.ShowRender, options.ShowRender)
	assert.Equal(t, cmd.ShowValidate, options.ShowValidate)
	assert.Equal(t, cmd.ShowHooks, options.ShowHooks)
	assert.Equal(t, cmd.ShowAssertions, options.ShowAssertions)
}
