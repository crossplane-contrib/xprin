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
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	internalcfg "github.com/crossplane-contrib/xprin/internal/config"
	unittestUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
)

func TestCmd_Run(t *testing.T) {
	configPath := "/test/config.yaml"
	cfg := &internalcfg.Config{}

	cmd := &Cmd{
		Check:      false,
		Config:     cfg,
		ConfigPath: configPath,
	}

	// Test that Run method works
	ctx := &kong.Context{}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestOperation_Run(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test repository
	repoPath := unittestUtils.CreateTestDir(t, filepath.Join(tmpDir, "test-repo"), 0o755)

	// Initialize git repository with remote
	unittestUtils.CreateGitRepo(t, unittestUtils.GitRepoOptions{
		Path:      repoPath,
		RemoteURL: "https://github.com/example/test-repo.git",
	})

	// Create test config
	cfg := &internalcfg.Config{
		Dependencies: map[string]string{
			"crossplane": "go", // using 'go' as a known command
		},
		Repositories: map[string]string{
			"test-repo": repoPath, // using the created test repository
		},
	}

	tests := []struct {
		name           string
		cmd            *Cmd
		config         *internalcfg.Config
		wantOutput     []string
		wantErr        bool
		wantErrContain string
	}{
		{
			name:   "without validation",
			cmd:    &Cmd{},
			config: cfg,
			wantOutput: []string{
				"Configuration file:",
				"Repositories:",
				"test-repo:",
				"Dependencies:",
				"crossplane:",
			},
		},
		{
			name: "check successful",
			cmd: &Cmd{
				Check: true,
			},
			config: cfg,
			wantOutput: []string{
				"Configuration file:",
				"Configuration check successful",
				"Repositories:",
				"test-repo:",
				"Dependencies:",
				"crossplane:",
			},
		},
		{
			name: "check fails - empty dependency",
			cmd: &Cmd{
				Check: true,
			},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "",
				},
			},
			wantOutput: []string{
				"Configuration file:",
			},
			wantErr:        true,
			wantErrContain: "invalid dependencies",
		},
		{
			name: "validation fails - invalid repository",
			cmd: &Cmd{
				Check: true,
			},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "go",
				},
				Repositories: map[string]string{
					"invalid-repo": "/nonexistent/path",
				},
			},
			wantOutput: []string{
				"Configuration file:",
			},
			wantErr:        true,
			wantErrContain: "invalid repositories",
		},
		{
			name: "check fails - combined errors",
			cmd: &Cmd{
				Check: true,
			},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "hello",
				},
				Repositories: map[string]string{
					"test-repo":    repoPath,            // valid repository
					"invalid-repo": "/nonexistent/path", // invalid repository
				},
			},
			wantOutput: []string{
				"Configuration file:",
			},
			wantErr:        true,
			wantErrContain: "invalid dependencies:\ncrossplane: hello not found in PATH\ninvalid repositories:\ninvalid-repo:",
		},
		{
			name: "with subcommands section",
			cmd:  &Cmd{},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "go",
				},
				Subcommands: &internalcfg.Subcommands{
					Render:   "render --foo",
					Validate: "validate --bar",
				},
			},
			wantOutput: []string{
				"Subcommands:",
				"- render: render --foo",
				"- validate: validate --bar",
			},
		},
		{
			name: "with only render subcommand",
			cmd:  &Cmd{},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "go",
				},
				Subcommands: &internalcfg.Subcommands{
					Render: "render --foo",
				},
			},
			wantOutput: []string{
				"Subcommands:",
				"- render: render --foo",
			},
		},
		{
			name: "with only validate subcommand",
			cmd:  &Cmd{},
			config: &internalcfg.Config{
				Dependencies: map[string]string{
					"crossplane": "go",
				},
				Subcommands: &internalcfg.Subcommands{
					Validate: "validate --bar",
				},
			},
			wantOutput: []string{
				"Subcommands:",
				"- validate: validate --bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set config in the command
			tt.cmd.Config = tt.config
			tt.cmd.ConfigPath = "/test/config.yaml"

			// Capture both stdout and stderr during Run
			var err error

			captured := unittestUtils.CaptureOutput(func() {
				ctx := &kong.Context{}
				err = tt.cmd.Run(ctx)
			})

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Operation.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message content if expected
			if tt.wantErr && tt.wantErrContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.wantErrContain) && !strings.Contains(captured.Stderr, tt.wantErrContain) {
					t.Errorf("Operation.Run() error message = %v, want to contain %v",
						err.Error(), tt.wantErrContain)
				}
			}

			// Check output contains expected strings
			for _, want := range tt.wantOutput {
				if !strings.Contains(captured.Stdout, want) {
					t.Errorf("Operation.Run() output = %v, want %v", captured.Stdout, want)
				}
			}
		})
	}
}
