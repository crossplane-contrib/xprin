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
)

func TestWriteTestFile(t *testing.T) {
	tempDir := t.TempDir()

	// Test file path
	filePath := filepath.Join(tempDir, "test-file.txt")

	// Test content
	content := "This is a test file content"

	// Call the function we're testing
	path := WriteTestFile(t, filePath, content)

	// Verify the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("File was not created at %s", path)
	}

	// Verify the content is correct
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch. Got %q, want %q", string(data), content)
	}
}

func TestCreateTestDir(t *testing.T) {
	parentDir := t.TempDir()
	dirPath := filepath.Join(parentDir, "nested", "test-dir")
	path := CreateTestDir(t, dirPath, 0o755)

	// Verify the directory exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Fatalf("Directory was not created at %s", path)
	}

	if !info.IsDir() {
		t.Errorf("Created path is not a directory: %s", path)
	}

	// Verify permissions (this might not work exactly as expected on Windows)
	// Just check it's not 0000
	if info.Mode().Perm()&0o700 == 0 {
		t.Errorf("Directory permissions don't include read/write/execute for owner")
	}
}
