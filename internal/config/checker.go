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

// Package config provides functions for checking the configuration file of the xprin tool.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/utils"
	"github.com/go-git/go-git/v5"
)

// CheckDependency checks if a dependency is valid.
func CheckDependency(dep string) error {
	// Check for empty or whitespace-only command
	trimmed := strings.TrimSpace(dep)
	if trimmed == "" {
		return fmt.Errorf("empty command")
	}

	// Check for leading/trailing whitespace
	if trimmed != dep {
		return fmt.Errorf("%s not found in PATH", dep)
	}

	// Split command by spaces to handle subcommands
	cmdParts := strings.Fields(dep)
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Case 1: Absolute path - validate file exists and is executable
	if filepath.IsAbs(cmdParts[0]) {
		if info, err := os.Stat(cmdParts[0]); err != nil || info.Mode()&0o111 == 0 {
			return fmt.Errorf("path %s is not executable", cmdParts[0])
		}

		return nil
	}

	// Case 2: Command in PATH
	path, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return fmt.Errorf("%s not found in PATH", cmdParts[0])
	}

	// Verify executable permissions
	if info, err := os.Stat(path); err != nil || info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", cmdParts[0])
	}

	return nil
}

// CheckDependencies checks if all required dependencies are present and valid.
func (c *Config) CheckDependencies() error {
	mandatoryDeps := []string{"crossplane"}

	var (
		missingDeps []string
		invalidDeps []string
	)

	// Check for missing mandatory dependencies

	for _, dep := range mandatoryDeps {
		if _, exists := c.Dependencies[dep]; !exists {
			missingDeps = append(missingDeps, dep)
		}
	}

	// Check all configured dependencies
	for dep, value := range c.Dependencies {
		if err := CheckDependency(value); err != nil {
			invalidDeps = append(invalidDeps, fmt.Sprintf("%s: %v", dep, err))
		}
	}

	if len(missingDeps) > 0 || len(invalidDeps) > 0 {
		var err strings.Builder
		if len(missingDeps) > 0 {
			err.WriteString("missing mandatory dependencies: ")
			err.WriteString(strings.Join(missingDeps, ", "))
			err.WriteString("\n")
		}

		if len(invalidDeps) > 0 {
			err.WriteString("invalid dependencies:\n")
			err.WriteString(strings.Join(invalidDeps, "\n"))
		}

		return fmt.Errorf("%s", err.String())
	}

	return nil
}

// CheckSubcommands checks if the subcommands.render and subcommands.validate are valid.
// They must start with "render" or "beta render" (for render), and "validate" or "beta validate" (for validate).
func (c *Config) CheckSubcommands() error {
	if c.Subcommands == nil {
		return nil // subcommands section is optional
	}

	var errs []string

	// Helper to check a command string
	checkSubCmd := func(subCmdStr, key string) {
		if subCmdStr == "" {
			return
		}

		parts := strings.Fields(subCmdStr)
		if len(parts) == 0 {
			errs = append(errs, fmt.Sprintf("subcommands.%s is empty", key))
			return
		}
		// Accept "render" or "beta render" for render, "validate" or "beta validate" for validate
		expected := key

		var startIdx int
		//nolint:gocritic // Complex conditions don't translate well to switch statement
		if parts[0] == "beta" && len(parts) > 1 && parts[1] == expected {
			startIdx = 2
		} else if parts[0] == expected {
			startIdx = 1
		} else {
			errs = append(errs, fmt.Sprintf("subcommands.%s must start with '%s' or 'beta %s'", key, expected, expected))
			return
		}
		// Check that all remaining parts are flags
		for i, arg := range parts[startIdx:] {
			if !strings.HasPrefix(arg, "-") {
				errs = append(errs, fmt.Sprintf("subcommands.%s: argument %d ('%s') must be a flag (start with '-' or '--')", key, i+startIdx, arg))
			}
		}
	}

	checkSubCmd(c.Subcommands.Render, "render")
	checkSubCmd(c.Subcommands.Validate, "validate")

	if len(errs) > 0 {
		return fmt.Errorf("invalid commands section:\n%s", strings.Join(errs, "\n"))
	}

	return nil
}

// CheckRepositories checks if all configured repositories are valid
//
//nolint:gocognit // Complex validation logic with multiple conditions
func (c *Config) CheckRepositories() error {
	var invalidRepos []string

	for name, path := range c.Repositories {
		// Expand tilde and make absolute in path
		expandedPath, err := utils.ExpandTildeAbs(path)
		if err != nil {
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: failed to expand path: %v", name, err))
			continue
		}

		// Check if directory exists
		if err := utils.VerifyPathExists(expandedPath); err != nil {
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: directory does not exist", name))
			continue
		}

		// Open and validate git repository
		gitRepo, err := git.PlainOpen(expandedPath)
		if err != nil {
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: not a valid git repository", name))
			continue
		}

		// Get remote URL
		remotes, err := gitRepo.Remotes()
		if err != nil {
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: failed to get repository remotes", name))
			continue
		}

		if len(remotes) == 0 {
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: failed to get origin remote", name))
			continue
		}

		// Find "origin" remote
		var originURL string

		for _, remote := range remotes {
			if remote.Config().Name == "origin" {
				if len(remote.Config().URLs) > 0 {
					originURL = remote.Config().URLs[0]
					break
				}
			}
		}

		switch {
		case originURL == "":
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: failed to get origin remote", name))
		case strings.Contains(originURL, "?"):
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: invalid remote URL format: query parameters are not allowed", name))
		case !strings.HasPrefix(originURL, "https://") && !strings.HasPrefix(originURL, "git@"):
			invalidRepos = append(invalidRepos, fmt.Sprintf("%s: invalid remote URL format: must be HTTPS or SSH. Got: %s", name, originURL))
		default:
			// Get the repository name from the origin URL
			repoName := ""

			if strings.HasPrefix(originURL, "git@") {
				parts := strings.Split(originURL, ":")
				if len(parts) == 2 {
					repoName = strings.TrimSuffix(filepath.Base(parts[1]), ".git")
				}
			} else {
				repoName = strings.TrimSuffix(filepath.Base(originURL), ".git")
			}

			if repoName != "" && repoName != name {
				invalidRepos = append(invalidRepos, fmt.Sprintf("%s: repository name mismatch", name))
			}
		}
	}

	if len(invalidRepos) > 0 {
		return fmt.Errorf("invalid repositories:\n%s", strings.Join(invalidRepos, "\n"))
	}

	return nil
}
