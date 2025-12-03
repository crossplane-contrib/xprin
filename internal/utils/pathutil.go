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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExpandTilde replaces leading ~ with the user's home directory in a path.
// It handles paths like "~/path". If the path doesn't start with a tilde, it returns the path unchanged.
func ExpandTilde(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

// ExpandAbs returns the absolute path for the given path (does not expand tilde).
func ExpandAbs(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}

	return filepath.Abs(path)
}

// ExpandTildeAbs expands tilde and then returns the absolute path.
func ExpandTildeAbs(path string) (string, error) {
	tildeExpanded, err := ExpandTilde(path)
	if err != nil {
		return "", err
	}

	return ExpandAbs(tildeExpanded)
}

// Exists returns true if the given path exists (file or directory), false otherwise.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// VerifyPathExists returns nil if the path exists, or an error if it does not.
func VerifyPathExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	return nil
}

// ValidateYAML checks if the output is valid YAML.
func ValidateYAML(yamlData []byte) error {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &doc); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return nil
}
