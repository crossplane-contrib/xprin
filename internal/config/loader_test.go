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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// TestLoad tests loading and validating configuration files.
func TestLoad(t *testing.T) {
	// Helper to create string pointer for test cases
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name       string
		configPath string
		configData *string // nil = don't write file, "" = write empty file, "data" = write file with data
		setupFunc  func()
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name:       "non-existent file returns error",
			configPath: "/path/to/nonexistent/file_xprin.yaml",
			wantErr:    true,
		},
		{
			name:       "invalid yaml",
			configPath: "/invalid.yaml",
			configData: strPtr(`invalid: yaml: :`),
			wantErr:    true,
		},
		{
			name:       "empty config",
			configPath: "/empty.yaml",
			configData: strPtr(``),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Dependencies == nil {
					t.Error("Dependencies map should be initialized")
				}

				if cfg.Repositories == nil {
					t.Error("Repositories slice should be initialized")
				}
			},
		},
		{
			name:       "valid config with all fields",
			configPath: "/valid-full.yaml",
			configData: strPtr(`dependencies:
  crossplane: go
repositories:
  test-repo: /fake/path/to/repo
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if len(cfg.Dependencies) != 1 {
					t.Errorf("Expected 1 dependency, got %d", len(cfg.Dependencies))
				}

				if len(cfg.Repositories) != 1 {
					t.Errorf("Expected 1 repository, got %d", len(cfg.Repositories))
				}

				if path, ok := cfg.Repositories["test-repo"]; !ok || path != "/fake/path/to/repo" {
					t.Errorf("Expected repository path for 'test-repo' to be %q, got %q", "/fake/path/to/repo", path)
				}
			},
		},
		{
			name:       "config with only dependencies",
			configPath: "/deps-only.yaml",
			configData: strPtr(`dependencies:
  crossplane: /usr/local/bin/crossplane
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if len(cfg.Dependencies) != 1 {
					t.Errorf("Expected 1 dependency, got %d", len(cfg.Dependencies))
				}

				if len(cfg.Repositories) != 0 {
					t.Errorf("Expected 0 repositories, got %d", len(cfg.Repositories))
				}
			},
		},
		{
			name:       "config with only repositories",
			configPath: "/repos-only.yaml",
			configData: strPtr(`repositories:
  repo1: /path/to/repo1
  repo2: /path/to/repo2
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if len(cfg.Dependencies) != 0 {
					t.Errorf("Expected 0 dependencies, got %d", len(cfg.Dependencies))
				}

				if len(cfg.Repositories) != 2 {
					t.Errorf("Expected 2 repositories, got %d", len(cfg.Repositories))
				}

				if _, ok := cfg.Repositories["repo1"]; !ok {
					t.Error("Expected repository 'repo1' to exist")
				}

				if _, ok := cfg.Repositories["repo2"]; !ok {
					t.Error("Expected repository 'repo2' to exist")
				}
			},
		},
		{
			name:       "config with null maps",
			configPath: "/null-maps.yaml",
			configData: strPtr(`dependencies: null
repositories: null
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Dependencies == nil {
					t.Error("Dependencies map should be initialized even when null")
				}

				if cfg.Repositories == nil {
					t.Error("Repositories slice should be initialized even when null")
				}
			},
		},
		{
			name:       "config with special characters",
			configPath: "/special-chars.yaml",
			configData: strPtr(`dependencies:
  "crossplane!@#": "path with spaces/crossplane"
repositories:
  "repo with spaces": "/path/to/repo with spaces"
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if v, ok := cfg.Dependencies["crossplane!@#"]; !ok || v != "path with spaces/crossplane" {
					t.Error("Special characters in dependency name/value not preserved")
				}

				if path, ok := cfg.Repositories["repo with spaces"]; !ok || path != "/path/to/repo with spaces" {
					t.Error("Spaces in repository name/path not preserved")
				}
			},
		},
		{
			name:       "config with empty values",
			configPath: "/empty-values.yaml",
			configData: strPtr(`dependencies:
  crossplane: ""
repositories:
  "": ""
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if len(cfg.Dependencies) != 1 {
					t.Error("Empty dependency values should be preserved")
				}

				if len(cfg.Repositories) != 1 {
					t.Error("Repository with empty values should be preserved")
				}

				if _, ok := cfg.Repositories[""]; !ok {
					t.Error("Empty repository key should be preserved")
				}
			},
		},
		{
			name:       "config with environment variables",
			configPath: "/env-vars.yaml",
			configData: strPtr(`dependencies:
  crossplane: "${HOME}/bin/crossplane"
repositories:
  env-repo: "${HOME}/repos/test"
`),
			setupFunc: func() {
				t.Setenv("HOME", "/test-home")
			},
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				// Environment variables should not be expanded during load
				if !strings.Contains(cfg.Dependencies["crossplane"], "${HOME}") {
					t.Error("Environment variables should not be expanded")
				}

				if path, ok := cfg.Repositories["env-repo"]; !ok || !strings.Contains(path, "${HOME}") {
					t.Error("Environment variables in repository path should not be expanded")
				}
			},
		},
		{
			name:       "config with missing parent directory",
			configPath: "/non-existent-dir/config.yaml",
			wantErr:    true,
		},
		{
			name:       "config with invalid repository entry",
			configPath: "/invalid-repo-entry.yaml",
			configData: strPtr(`repositories:
  - name: only-name-no-path
`),
			wantErr: true,
		},
		{
			name:       "config with invalid repository format",
			configPath: "/invalid-repo-format.yaml",
			configData: strPtr(`repositories:
  - not-a-map
  - also-not-a-map
`),
			wantErr: true,
		},
		{
			// Null values in YAML are unmarshaled as empty strings in Go
			name:       "config with nil repository values",
			configPath: "/nil-repo-values.yaml",
			configData: strPtr(`repositories:
  repo1: null
  repo2: /path2
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if len(cfg.Repositories) != 2 {
					t.Errorf("Expected 2 repositories, got %d", len(cfg.Repositories))
				}

				if v, ok := cfg.Repositories["repo1"]; !ok || v != "" {
					t.Errorf("Repository with null value should be present with empty string value")
				}

				if _, ok := cfg.Repositories["repo2"]; !ok {
					t.Errorf("Valid repository entry should be included")
				}
			},
		},
		{
			name:       "subcommands defaults when missing",
			configPath: "/subcommands-missing.yaml",
			configData: strPtr(`dependencies:
  crossplane: go
repositories: {}
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Subcommands == nil {
					t.Error("Subcommands struct should be initialized")
				}

				if cfg.Subcommands.Render != DefaultRenderCmd {
					t.Errorf("Expected default render, got %q", cfg.Subcommands.Render)
				}

				if cfg.Subcommands.Validate != DefaultValidateCmd {
					t.Errorf("Expected default validate, got %q", cfg.Subcommands.Validate)
				}
			},
		},
		{
			name:       "subcommands custom values",
			configPath: "/subcommands-custom.yaml",
			configData: strPtr(`dependencies:
  crossplane: go
subcommands:
  render: "render --foo"
  validate: "validate --bar"
repositories: {}
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Subcommands.Render != "render --foo" {
					t.Errorf("Expected custom render, got %q", cfg.Subcommands.Render)
				}

				if cfg.Subcommands.Validate != "validate --bar" {
					t.Errorf("Expected custom validate, got %q", cfg.Subcommands.Validate)
				}
			},
		},
		{
			name:       "subcommands partial (only render)",
			configPath: "/subcommands-partial.yaml",
			configData: strPtr(`dependencies:
  crossplane: go
subcommands:
  render: "render --foo"
repositories: {}
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Subcommands.Render != "render --foo" {
					t.Errorf("Expected custom render, got %q", cfg.Subcommands.Render)
				}

				if cfg.Subcommands.Validate != DefaultValidateCmd {
					t.Errorf("Expected default validate, got %q", cfg.Subcommands.Validate)
				}
			},
		},
		{
			name:       "subcommands partial (only validate)",
			configPath: "/subcommands-partial2.yaml",
			configData: strPtr(`dependencies:
  crossplane: go
subcommands:
  validate: "validate --bar"
repositories: {}
`),
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()

				if cfg.Subcommands.Render != DefaultRenderCmd {
					t.Errorf("Expected default render, got %q", cfg.Subcommands.Render)
				}

				if cfg.Subcommands.Validate != "validate --bar" {
					t.Errorf("Expected custom validate, got %q", cfg.Subcommands.Validate)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All tests can use in-memory filesystem
			// (Load() only parses YAML, doesn't validate repositories)
			fs := afero.NewMemMapFs()

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Write file if configData is provided (nil = don't write, empty string = write empty file)
			// We expand the path here to write the file to the in-memory filesystem.
			// Load() will expand it again when reading (this is expected behavior).
			// Note: We're not testing tilde expansion of the config file path here - we're just
			// using it to set up the test. Tilde expansion is tested in utils/pathutil_test.go.
			if tt.configData != nil {
				expandedPath, err := utils.ExpandTildeAbs(tt.configPath)
				require.NoError(t, err, "Failed to expand path for test file")
				require.NoError(t, afero.WriteFile(fs, expandedPath, []byte(*tt.configData), 0o644))
			}

			cfg, err := Load(fs, tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestFallback(t *testing.T) {
	tmpDir := t.TempDir()
	crossplanePath := filepath.Join(tmpDir, "crossplane")

	createBin := func(path string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("failed to create binary: %v", err)
		}
	}

	// Case 1: No mandatory dependencies -> fail
	t.Setenv("PATH", tmpDir)

	_, err := Fallback()
	if err == nil {
		t.Error("Expected error when no mandatory dependencies are present")
	}

	// Case 2: Only mandatory dependencies (crossplane) -> success
	createBin(crossplanePath)

	cfg, err := Fallback()
	if err != nil {
		t.Errorf("Expected no error when mandatory dependency is present, got: %v", err)
	}

	if cfg == nil || cfg.Dependencies["crossplane"] != "crossplane" {
		t.Error("Fallback config did not set crossplane dependency correctly")
	}
	// Only crossplane should be in dependencies since convert-claim-to-xr and patch-xr are now library dependencies
	if len(cfg.Dependencies) != 1 {
		t.Errorf("Expected only 1 dependency (crossplane), got %d: %v", len(cfg.Dependencies), cfg.Dependencies)
	}
}
