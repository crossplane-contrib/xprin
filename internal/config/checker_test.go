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
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// TestRunCheck_CLIDetectionOutput verifies that RunCheck prints the correct
// IsV1CLI and IsLegacyCLI detection strings for all four tier combinations.
func TestRunCheck_CLIDetectionOutput(t *testing.T) {
	tests := []struct {
		name         string
		isV1CLI      bool
		isLegacyCLI  bool
		wantV1Label  string
		wantLegLabel string
	}{
		{
			name:         "v2+ and current CLI",
			isV1CLI:      false,
			isLegacyCLI:  false,
			wantV1Label:  "v2+",
			wantLegLabel: "current (>= v2.3.0)",
		},
		{
			name:         "v1 and current CLI",
			isV1CLI:      true,
			isLegacyCLI:  false,
			wantV1Label:  "v1",
			wantLegLabel: "current (>= v2.3.0)",
		},
		{
			name:         "v2+ and legacy CLI",
			isV1CLI:      false,
			isLegacyCLI:  true,
			wantV1Label:  "v2+",
			wantLegLabel: "legacy (< v2.3.0)",
		},
		{
			name:         "v1 and legacy CLI",
			isV1CLI:      true,
			isLegacyCLI:  true,
			wantV1Label:  "v1",
			wantLegLabel: "legacy (< v2.3.0)",
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
				IsV1CLI:      tt.isV1CLI,
				IsLegacyCLI:  tt.isLegacyCLI,
			}

			output := unittestsUtils.CaptureStdout(func() {
				err := RunCheck(cfg, "", false)
				require.NoError(t, err)
			})

			assert.Contains(t, output, "Crossplane CLI detected as "+tt.wantV1Label)
			assert.Contains(t, output, tt.wantLegLabel)
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
