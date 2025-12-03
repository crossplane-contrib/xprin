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

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
)

func TestCheckRepositories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test paths
	validRepo := filepath.Join(tmpDir, "valid-repo")
	emptyRepo := filepath.Join(tmpDir, "empty-repo")
	nonExistentRepo := filepath.Join(tmpDir, "non-existent")
	homeRepo := filepath.Join("~/", "test-repo")

	// Create test repositories
	unittestsUtils.CreateTestDir(t, validRepo, 0o755)
	unittestsUtils.CreateTestDir(t, emptyRepo, 0o755)

	// Initialize git repo in validRepo
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path:      validRepo,
		RemoteURL: "git@github.com:myorg/valid-repo.git",
	})

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid repository",
			cfg: &Config{
				Repositories: map[string]string{
					"valid-repo": validRepo,
				},
			},
		},
		{
			name: "non-existent repository",
			cfg: &Config{
				Repositories: map[string]string{
					"non-existent": nonExistentRepo,
				},
			},
			wantErr: "directory does not exist",
		},
		{
			name: "not a git repository",
			cfg: &Config{
				Repositories: map[string]string{
					"empty-repo": emptyRepo,
				},
			},
			wantErr: "not a valid git repository",
		},
		{
			name: "multiple repositories with one invalid",
			cfg: &Config{
				Repositories: map[string]string{
					"valid-repo":   validRepo,
					"non-existent": nonExistentRepo,
				},
			},
			wantErr: "directory does not exist",
		},
		{
			name: "repository with tilde path",
			cfg: &Config{
				Repositories: map[string]string{
					"test-repo": homeRepo,
				},
			},
			wantErr: "directory does not exist",
		},
		{
			name: "repository name mismatch",
			cfg: &Config{
				Repositories: map[string]string{
					"wrong-name": validRepo,
				},
			},
			wantErr: "repository name mismatch",
		},
		{
			name: "empty repository list",
			cfg: &Config{
				Repositories: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.CheckRepositories()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckRepositories() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckRepositories() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckRepositories() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckRepositoriesWithRemotes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test paths
	sshRepo := filepath.Join(tmpDir, "ssh-repo")
	httpsRepo := filepath.Join(tmpDir, "https-repo")
	invalidURLRepo := filepath.Join(tmpDir, "invalid-url-repo")
	noRemoteRepo := filepath.Join(tmpDir, "no-remote-repo")

	// Create test repositories
	unittestsUtils.CreateTestDir(t, sshRepo, 0o755)
	unittestsUtils.CreateTestDir(t, httpsRepo, 0o755)
	unittestsUtils.CreateTestDir(t, invalidURLRepo, 0o755)
	unittestsUtils.CreateTestDir(t, noRemoteRepo, 0o755)

	// Initialize git repos with different remote URLs
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path:      sshRepo,
		RemoteURL: "git@github.com:myorg/ssh-repo.git",
	})
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path:      httpsRepo,
		RemoteURL: "https://github.com/myorg/https-repo.git",
	})
	// Invalid URL repo
	unittestsUtils.CreateTestDir(t, invalidURLRepo, 0o755)
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path:      invalidURLRepo,
		RemoteURL: ":::invalid-url:::",
	})

	// No remote repo
	unittestsUtils.CreateTestDir(t, noRemoteRepo, 0o755)
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path: noRemoteRepo,
	})

	// Empty remote URL repo
	emptyRemotePath := unittestsUtils.CreateTestDir(t, filepath.Join(tmpDir, "empty-remote-repo"), 0o755)
	unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
		Path:      emptyRemotePath,
		RemoteURL: "",
	})

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid SSH URL",
			cfg: &Config{
				Repositories: map[string]string{
					"ssh-repo": sshRepo,
				},
			},
		},
		{
			name: "valid HTTPS URL",
			cfg: &Config{
				Repositories: map[string]string{
					"https-repo": httpsRepo,
				},
			},
		},
		{
			name: "invalid remote URL format",
			cfg: &Config{
				Repositories: map[string]string{
					"invalid-url": invalidURLRepo,
				},
			},
			wantErr: "invalid remote URL format",
		},
		{
			name: "no origin remote",
			cfg: &Config{
				Repositories: map[string]string{
					"no-remote": noRemoteRepo,
				},
			},
			wantErr: "failed to get origin remote",
		},
		{
			name: "mixed valid and invalid",
			cfg: &Config{
				Repositories: map[string]string{
					"ssh-repo":    sshRepo,
					"invalid-url": invalidURLRepo,
				},
			},
			wantErr: "invalid remote URL format",
		},
		{
			name: "repository with empty remote URL",
			cfg: &Config{
				Repositories: map[string]string{
					"empty-remote-repo": emptyRemotePath,
				},
			},
			wantErr: "failed to get origin remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.CheckRepositories()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckRepositories() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckRepositories() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckRepositories() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckRepositoriesWithSpecialPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test paths with spaces and special characters
	repoWithSpaces := filepath.Join(tmpDir, "repo with spaces")
	repoWithSpecialChars := filepath.Join(tmpDir, "repo-$pecial#chars")

	// Create test repositories
	for _, path := range []string{repoWithSpaces, repoWithSpecialChars} {
		unittestsUtils.CreateTestDir(t, path, 0o755)
		unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
			Path:      path,
			RemoteURL: "git@github.com:myorg/repo.git",
		})
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "repository path with spaces",
			cfg: &Config{
				Repositories: map[string]string{
					"repo": repoWithSpaces,
				},
			},
		},
		{
			name: "repository path with special characters",
			cfg: &Config{
				Repositories: map[string]string{
					"repo": repoWithSpecialChars,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.CheckRepositories()
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("CheckRepositories() error = nil, wantErr %q", tt.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("CheckRepositories() error = %v, wantErr %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckRepositories() error = %v, wantErr nil", err)
			}
		})
	}
}

func TestCheckRepositoriesSpecialURLs(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup test repositories with various URL patterns
	testCases := []struct {
		name      string
		repoName  string
		remoteURL string
		wantErr   string
	}{
		{
			name:      "SSH URL with port",
			repoName:  "repo1",
			remoteURL: "git@github.com:2222/myorg/repo1.git",
		},
		{
			name:      "HTTPS URL with username",
			repoName:  "repo2",
			remoteURL: "https://user@github.com/myorg/repo2.git",
		},
		{
			name:      "HTTPS URL with username and password",
			repoName:  "repo3",
			remoteURL: "https://user:pass@github.com/myorg/repo3.git",
		},
		{
			name:      "SSH URL with custom hostname",
			repoName:  "repo4",
			remoteURL: "git@custom.gitlab.com:myorg/repo4.git",
		},
		{
			name:      "HTTPS URL with query parameters",
			repoName:  "repo5",
			remoteURL: "https://github.com/myorg/repo5.git?token=abc123",
			wantErr:   "invalid remote URL format",
		},
		{
			name:      "File protocol URL",
			repoName:  "repo6",
			remoteURL: "file:///path/to/repo6.git",
			wantErr:   "invalid remote URL format",
		},
	}

	// Create test repositories
	repos := make(map[string]string, len(testCases))
	for _, tc := range testCases {
		repoPath := filepath.Join(tmpDir, tc.repoName)
		unittestsUtils.CreateTestDir(t, repoPath, 0o755)
		unittestsUtils.CreateGitRepo(t, unittestsUtils.GitRepoOptions{
			Path:      repoPath,
			RemoteURL: tc.remoteURL,
		})
		repos[tc.repoName] = repoPath
	}

	// Test each case individually
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Repositories: map[string]string{
					tc.repoName: filepath.Join(tmpDir, tc.repoName),
				},
			}

			err := cfg.CheckRepositories()
			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("CheckRepositories() error = nil, wantErr %q", tc.wantErr)
					return
				}

				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("CheckRepositories() error = %v, wantErr %q", err, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Errorf("CheckRepositories() error = %v, wantErr nil", err)
			}
		})
	}

	// Test all repositories together
	t.Run("AllRepositoriesTogether", func(t *testing.T) {
		cfg := &Config{
			Repositories: repos,
		}

		err := cfg.CheckRepositories()
		if err == nil {
			t.Error("CheckRepositories() error = nil, want error for invalid URL formats")
		}
	})
}
