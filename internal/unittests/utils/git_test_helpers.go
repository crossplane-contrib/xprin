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
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitRepoOptions defines options for creating a test git repository.
type GitRepoOptions struct {
	// Path to create the repository at (required)
	Path string

	// RemoteURL is the URL to use for the origin remote
	// If empty, no remote will be added
	RemoteURL string

	// RemoteName is the name to use for the remote (defaults to "origin")
	RemoteName string

	// UserName is the Git user name (defaults to "Test User")
	UserName string

	// UserEmail is the Git user email (defaults to "test@example.com")
	UserEmail string

	// CreateInitialCommit indicates whether to create an initial empty commit
	CreateInitialCommit bool
}

// CreateGitRepo creates a test git repository with the specified options.
func CreateGitRepo(t *testing.T, opts GitRepoOptions) {
	t.Helper()

	// Validate required fields
	if opts.Path == "" {
		t.Fatalf("GitRepoOptions.Path is required")
	}

	// Set defaults for optional fields
	if opts.RemoteName == "" {
		opts.RemoteName = "origin"
	}

	if opts.UserName == "" {
		opts.UserName = "Test User"
	}

	if opts.UserEmail == "" {
		opts.UserEmail = "test@example.com"
	}

	// Initialize repository
	repo, err := git.PlainInit(opts.Path, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure repository user
	cfg, err := repo.Config()
	if err != nil {
		t.Fatalf("Failed to get repo config: %v", err)
	}

	cfg.User.Name = opts.UserName
	cfg.User.Email = opts.UserEmail

	if err := repo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set repo config: %v", err)
	}

	// Add remote if URL is provided
	if opts.RemoteURL != "" {
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: opts.RemoteName,
			URLs: []string{opts.RemoteURL},
		})
		if err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
	}

	// Create initial commit if requested
	if opts.CreateInitialCommit {
		wt, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}

		_, err = wt.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  opts.UserName,
				Email: opts.UserEmail,
				When:  time.Now(),
			},
		})
		if err != nil {
			t.Fatalf("Failed to create initial commit: %v", err)
		}
	}
}
