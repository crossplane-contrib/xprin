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
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/utils"
)

// ExpandPathRelativeToTestSuiteFile expands a path relative to the directory of the given testsuite file.
// If the path is absolute, it is returned as an absolute path (with tilde expanded if present).
// If the path starts with ~, it is expanded to the user's home directory.
// Otherwise, the path is joined with the directory of the testsuite file and made absolute.
func ExpandPathRelativeToTestSuiteFile(testSuiteFile, path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}

	if strings.HasPrefix(path, "~") {
		return utils.ExpandTildeAbs(path)
	}

	if filepath.IsAbs(path) {
		return path, nil
	}

	baseDir := filepath.Dir(testSuiteFile)

	return utils.ExpandAbs(filepath.Join(baseDir, path))
}
