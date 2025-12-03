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

// Package utils from internal/unittests provides helper functions for unit tests.
package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteTestFile writes content to a file, creating parent directories if needed.
// The path can be either:
// - A full path: WriteTestFile(t, "/path/to/file.yaml", "content")
// - A directory and name: WriteTestFile(t, filepath.Join(dir, "file.yaml"), "content").
func WriteTestFile(t *testing.T, path, content string) string {
	t.Helper()

	CreateTestDir(t, filepath.Dir(path), 0o750)

	// Write the file content
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}

	return path
}

// CreateTestDir creates a directory at the specified path, including any necessary parent
// directories. Unlike t.TempDir(), this creates a directory at exactly the path specified,
// not in the system's temporary directory. It does not register for automatic cleanup.
// Example:
//
//	repoPath := testutils.CreateTestDir(t, filepath.Join(tmpDir, "test-repo"), 0750)
func CreateTestDir(t *testing.T, path string, perm os.FileMode) string {
	t.Helper()

	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("Failed to create directory %s: %v", path, err)
	}

	return path
}
