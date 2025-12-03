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
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

// Basic test for CreateGitRepo function.
func TestCreateGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	// Create a basic repo
	repoPath := CreateTestDir(t, filepath.Join(tempDir, "test-repo"), 0o755)

	// Test with minimal options
	CreateGitRepo(t, GitRepoOptions{
		Path: repoPath,
	})

	// Verify it's a git repository
	_, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("Failed to open git repository: %v", err)
	}

	// Test with remote URL
	remoteRepoPath := CreateTestDir(t, filepath.Join(tempDir, "remote-repo"), 0o755)

	CreateGitRepo(t, GitRepoOptions{
		Path:      remoteRepoPath,
		RemoteURL: "https://example.com/test/repo.git",
	})

	// Open the repo to verify
	repo, err := git.PlainOpen(remoteRepoPath)
	if err != nil {
		t.Fatalf("Failed to open git repository: %v", err)
	}

	// Check remote was created
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatalf("Failed to get origin remote: %v", err)
	}

	if len(remote.Config().URLs) == 0 || remote.Config().URLs[0] != "https://example.com/test/repo.git" {
		t.Errorf("Remote URL mismatch. Expected %q, got URLs: %v",
			"https://example.com/test/repo.git", remote.Config().URLs)
	}

	// Skip the CreateInitialCommit test for now as it requires creating files
	// Testing it properly would require more setup with staging a file first

	// Verify custom remote name
	customRemoteRepoPath := CreateTestDir(t, filepath.Join(tempDir, "custom-remote-repo"), 0o755)

	CreateGitRepo(t, GitRepoOptions{
		Path:       customRemoteRepoPath,
		RemoteName: "upstream",
		RemoteURL:  "https://example.com/test/upstream.git",
	})

	// Open the repo to verify
	repo, err = git.PlainOpen(customRemoteRepoPath)
	if err != nil {
		t.Fatalf("Failed to open git repository: %v", err)
	}

	// Check remote was created with custom name
	remote, err = repo.Remote("upstream")
	if err != nil {
		t.Fatalf("Failed to get upstream remote: %v", err)
	}

	if len(remote.Config().URLs) == 0 || remote.Config().URLs[0] != "https://example.com/test/upstream.git" {
		t.Errorf("Remote URL mismatch. Got URLs: %v", remote.Config().URLs)
	}
}
