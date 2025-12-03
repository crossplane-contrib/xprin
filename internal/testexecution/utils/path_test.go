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

func TestExpandPathRelativeToTestSuiteFile(t *testing.T) {
	// Set up a fake HOME for tilde expansion (using in-memory path)
	fakeHome := "/fake/home"
	t.Setenv("HOME", fakeHome)

	baseDir := "/base/dir"
	testSuiteFile := filepath.Join(baseDir, "testsuite_xprin.yaml")

	// Absolute path containing tilde (should not expand tilde unless leading)
	absWithTilde := filepath.Join(string(os.PathSeparator), "path", "to", "~", "tests")
	wantAbsWithTilde := absWithTilde // Expect no expansion

	got, err := ExpandPathRelativeToTestSuiteFile(testSuiteFile, absWithTilde)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != wantAbsWithTilde {
		t.Errorf("ExpandPathRelativeToTestSuiteFile() = %v, want %v", got, wantAbsWithTilde)
	}

	// Absolute path
	absPath := filepath.Join(baseDir, "absdir")

	got, err = ExpandPathRelativeToTestSuiteFile(testSuiteFile, absPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != absPath {
		t.Errorf("ExpandPathRelativeToTestSuiteFile() = %v, want %v", got, absPath)
	}

	// Tilde path
	tildePath := "~/foo"
	wantTilde := filepath.Join(fakeHome, "foo")

	got, err = ExpandPathRelativeToTestSuiteFile(testSuiteFile, tildePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != wantTilde {
		t.Errorf("ExpandPathRelativeToTestSuiteFile() = %v, want %v", got, wantTilde)
	}

	// Relative path
	relPath := "rel/dir"
	wantRel := filepath.Join(baseDir, "rel/dir")

	got, err = ExpandPathRelativeToTestSuiteFile(testSuiteFile, relPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != wantRel {
		t.Errorf("ExpandPathRelativeToTestSuiteFile() = %v, want %v", got, wantRel)
	}

	// Empty path
	_, err = ExpandPathRelativeToTestSuiteFile(testSuiteFile, "")
	if err == nil || err.Error() != "empty path" {
		t.Errorf("expected error 'empty path', got %v", err)
	}
}
