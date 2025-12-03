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

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
)

func TestCheckDependency(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file with no execute permissions
	noExecPath := filepath.Join(tmpDir, "no-exec-file")
	unittestsUtils.WriteTestFile(t, noExecPath, "#!/bin/sh\necho test")

	if err := os.Chmod(noExecPath, 0o644); err != nil { // read/write but not executable
		t.Fatalf("Failed to set permissions: %v", err)
	}

	tests := []struct {
		name    string
		dep     string
		wantErr string
	}{
		{
			name:    "empty command",
			dep:     "",
			wantErr: "empty command",
		},
		{
			name:    "non-existent command",
			dep:     "non-existent-command-xyz",
			wantErr: "not found in PATH",
		},
		{
			name: "command exists in PATH",
			dep:  "go", // assuming go is installed
		},
		{
			name:    "non-existent absolute path",
			dep:     "/non/existent/path",
			wantErr: "not executable",
		},
		{
			name:    "all whitespace command",
			dep:     "   \t   ",
			wantErr: "empty command",
		},
		{
			name:    "no execute permission file",
			dep:     noExecPath,
			wantErr: "not executable",
		},
		{
			name:    "command with multiple spaces",
			dep:     "go  version", // multiple spaces between parts
			wantErr: "",            // This should be valid as we split by spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDependency(tt.dep)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckDependency() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckDependency() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckDependency() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	executablePath := filepath.Join(tmpDir, "test-executable")
	unittestsUtils.WriteTestFile(t, executablePath, "#!/bin/sh\necho test")
	// Set executable permission
	if err := os.Chmod(executablePath, 0o755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid dependencies",
			cfg: &Config{
				Dependencies: map[string]string{
					"crossplane": "go", // using 'go' as a test command that exists
				},
			},
		},
		{
			name: "missing mandatory dependency",
			cfg: &Config{
				Dependencies: map[string]string{
					// missing crossplane
				},
			},
			wantErr: "missing mandatory dependencies",
		},
		{
			name: "invalid dependency command",
			cfg: &Config{
				Dependencies: map[string]string{
					"crossplane": "non-existent-command-xyz",
				},
			},
			wantErr: "invalid dependencies",
		},
		{
			name: "dependencies with absolute path",
			cfg: &Config{
				Dependencies: map[string]string{
					"crossplane": executablePath,
				},
			},
		},
		{
			name: "dependency with spaces in command",
			cfg: &Config{
				Dependencies: map[string]string{
					"crossplane": "go version",
				},
			},
		},
		{
			name:    "empty config",
			cfg:     &Config{},
			wantErr: "missing mandatory dependencies",
		},
		{
			name: "dependencies with empty values",
			cfg: &Config{
				Dependencies: map[string]string{
					"crossplane": "",
				},
			},
			wantErr: "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.CheckDependencies()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckDependencies() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckDependencies() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckDependencies() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckDependencyWithNonExecutableFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file without execute permissions
	nonExecutablePath := filepath.Join(tmpDir, "non-executable")
	unittestsUtils.WriteTestFile(t, nonExecutablePath, "test")
	// Ensure non-executable permissions (0644 is already default, but being explicit)
	if err := os.Chmod(nonExecutablePath, 0o644); err != nil {
		t.Fatalf("Failed to set non-executable permissions: %v", err)
	}

	validateErr := CheckDependency(nonExecutablePath)
	if validateErr == nil {
		t.Error("CheckDependency() error = nil, want error about non-executable file")
	} else if !strings.Contains(validateErr.Error(), "not executable") {
		t.Errorf("CheckDependency() error = %v, want error about non-executable file", validateErr)
	}
}

func TestCheckSymlinkedDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an executable file
	execPath := filepath.Join(tmpDir, "executable")
	unittestsUtils.WriteTestFile(t, execPath, "#!/bin/sh\necho test")
	// Set executable permissions
	if err := os.Chmod(execPath, 0o755); err != nil {
		t.Fatalf("Failed to set executable permissions: %v", err)
	}

	// Create a symlink to the executable
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(execPath, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create a broken symlink
	brokenSymlinkPath := filepath.Join(tmpDir, "broken")
	if err := os.Symlink(filepath.Join(tmpDir, "nonexistent"), brokenSymlinkPath); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	tests := []struct {
		name    string
		dep     string
		wantErr string
	}{
		{
			name: "valid symlink to executable",
			dep:  symlinkPath,
		},
		{
			name:    "broken symlink",
			dep:     brokenSymlinkPath,
			wantErr: "not executable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDependency(tt.dep)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckDependency() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckDependency() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckDependency() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckDependencyExtraSpaces(t *testing.T) {
	tests := []struct {
		name    string
		dep     string
		value   string
		wantErr string
	}{
		{
			name:    "leading spaces",
			dep:     "   go",
			wantErr: "not found in PATH",
		},
		{
			name:    "trailing spaces",
			dep:     "go   ",
			wantErr: "not found in PATH",
		},
		{
			name: "multiple arguments with extra spaces",
			dep:  "go  version   --short",
		},
		{
			name:    "only spaces",
			dep:     "     ",
			wantErr: "empty command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDependency(tt.dep)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckDependency() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckDependency() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckDependency() error = %v, wantErr nil", err)
			}
		})
	}
}
