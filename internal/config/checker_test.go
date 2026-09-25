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

package config

import (
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/crossplane-contrib/xprin/internal/xpcli"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// TestRunCheck_CLIDetectionOutput verifies that RunCheck prints the correct
// tier detection string for each CLI tier.
func TestRunCheck_CLIDetectionOutput(t *testing.T) {
	tests := []struct {
		name    string
		cli     xpcli.XPCLI
		wantStr string
	}{
		{
			name:    "v2",
			cli:     xpcli.InitXPCLI(xpcli.TierV2),
			wantStr: "v2+, current (>= v2.3.0)",
		},
		{
			name:    "v2legacy",
			cli:     xpcli.InitXPCLI(xpcli.TierV2Legacy),
			wantStr: "v2+, legacy (< v2.3.0)",
		},
		{
			name:    "v1",
			cli:     xpcli.InitXPCLI(xpcli.TierV1),
			wantStr: "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// "go" is always present and executable — used as a stand-in for a real crossplane binary.
			cfg := &Config{
				Dependencies: map[string]string{
					CrossplaneCmd: "go",
				},
				Subcommands: &Subcommands{
					Render:   DefaultRenderCmd,
					Validate: DefaultValidateCmd,
				},
				Repositories: map[string]string{},
				XPCLI:        tt.cli,
			}

			output := unittestsUtils.CaptureStdout(func() {
				err := RunCheck(cfg, "", false)
				require.NoError(t, err)
			})

			assert.Contains(t, output, "Crossplane CLI detected as "+tt.wantStr)
		})
	}
}

// TestRunCheck_QuietSuppressesOutput verifies that quiet mode skips all non-error output.
func TestRunCheck_QuietSuppressesOutput(t *testing.T) {
	cfg := &Config{
		Dependencies: map[string]string{
			CrossplaneCmd: "go",
		},
		Subcommands: &Subcommands{
			Render:   DefaultRenderCmd,
			Validate: DefaultValidateCmd,
		},
		Repositories: map[string]string{},
	}

	output := unittestsUtils.CaptureStdout(func() {
		err := RunCheck(cfg, "", true)
		require.NoError(t, err)
	})

	assert.Empty(t, output)
}

// TestRunCheck_ConfigPathPrinted verifies that a non-empty configPath is printed in non-quiet mode.
func TestRunCheck_ConfigPathPrinted(t *testing.T) {
	cfg := &Config{
		Dependencies: map[string]string{
			CrossplaneCmd: "go",
		},
		Subcommands: &Subcommands{
			Render:   DefaultRenderCmd,
			Validate: DefaultValidateCmd,
		},
		Repositories: map[string]string{},
	}

	output := unittestsUtils.CaptureStdout(func() {
		err := RunCheck(cfg, "/some/config.yaml", false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Configuration file: /some/config.yaml")
}

// TestRunCheck_MissingCrossplaneErrors verifies that a missing crossplane dep returns an error.
func TestRunCheck_MissingCrossplaneErrors(t *testing.T) {
	cfg := &Config{
		Dependencies: map[string]string{},
		Subcommands:  &Subcommands{},
		Repositories: map[string]string{},
	}

	err := RunCheck(cfg, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing mandatory dependencies")
}
