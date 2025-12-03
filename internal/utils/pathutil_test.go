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

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/spf13/afero"
)

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "path without tilde",
			path: "/absolute/path",
			want: "/absolute/path",
		},
		{
			name: "path with tilde",
			path: "~/relative/path",
			want: filepath.Join(os.Getenv("HOME"), "relative/path"),
		},
		{
			name: "tilde not at start",
			path: "/path/with/~/tilde",
			want: "/path/with/~/tilde",
		},
		{
			name: "tilde without slash",
			path: "~",
			want: "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTilde(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTilde() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ExpandTilde() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandTildeWithoutHome(t *testing.T) {
	// Unset HOME to simulate UserHomeDir error
	t.Setenv("HOME", "")

	path := "~/test/path"

	_, err := ExpandTilde(path)
	if err == nil {
		t.Error("ExpandTilde() error = nil, want error about home directory")
	}

	_, err = ExpandTildeAbs(path)
	if err == nil {
		t.Error("ExpandTilde() error = nil, want error about home directory")
	}
}

func TestExpandAbs(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"absolute path", "/tmp/test", false},
		{"relative path", "../", false},
		{"empty path", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandAbs(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandAbs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				abs, _ := filepath.Abs(tt.path)
				if got != abs {
					t.Errorf("ExpandAbs() = %v, want %v", got, abs)
				}
			}
		})
	}
}

func TestExpandTildeAbs(t *testing.T) {
	// Set a fake HOME for tilde tests
	fakeHome := "/tmp/fakehome"
	t.Setenv("HOME", fakeHome)

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"absolute path", "/tmp/test", "/tmp/test", false},
		{"tilde path", "~/foo", filepath.Join(fakeHome, "foo"), false},
		{"relative path", "../", func() string { abs, _ := filepath.Abs("../"); return abs }(), false},
		{"empty path", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTildeAbs(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandTildeAbs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("ExpandTildeAbs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExistsAndVerifyPathExists(t *testing.T) {
	// Use afero with OS filesystem to create real files for testing
	// (since Exists and VerifyPathExists use os.Stat which requires real filesystem)
	fs := afero.NewOsFs()

	tempDir, err := afero.TempDir(fs, "", "xprin-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	defer func() {
		_ = fs.RemoveAll(tempDir)
	}()

	tempFile := filepath.Join(tempDir, "file.txt")
	if err := afero.WriteFile(fs, tempFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Exists should return true for existing file
	if !Exists(tempFile) {
		t.Errorf("Exists() = false, want true for existing file")
	}
	// Exists should return false for non-existent file
	nonexistent := filepath.Join(tempDir, "nope.txt")
	if Exists(nonexistent) {
		t.Errorf("Exists() = true, want false for non-existent file")
	}

	// VerifyPathExists should return nil for existing file
	if err := VerifyPathExists(tempFile); err != nil {
		t.Errorf("VerifyPathExists() error = %v, want nil", err)
	}
	// VerifyPathExists should return error for non-existent file
	if err := VerifyPathExists(nonexistent); err == nil {
		t.Errorf("VerifyPathExists() error = nil, want error")
	}
}

func TestValidateYAML(t *testing.T) {
	// Test valid YAML
	validYAML := []byte("key: value\narray:\n  - item1\n  - item2")
	err := ValidateYAML(validYAML)
	assert.NoError(t, err)

	// Test invalid YAML
	invalidYAML := []byte("key: value\nbroken: [array")
	err = ValidateYAML(invalidYAML)
	assert.Error(t, err)
}
