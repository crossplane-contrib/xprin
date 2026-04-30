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

package runner

import (
	"path/filepath"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestSanitizePathSegment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain name unchanged", input: "my-test", expected: "my-test"},
		{name: "unsafe chars replaced", input: `a/b\c:d`, expected: "a_b_c_d"},
		{name: "consecutive underscores collapsed", input: "a//b", expected: "a_b"},
		{name: "leading and trailing underscores trimmed", input: "/test/", expected: "test"},
		{name: "empty string returns unnamed", input: "", expected: "unnamed"},
		{name: "only unsafe chars returns unnamed", input: "///", expected: "unnamed"},
		{name: "spaces trimmed and internal spaces replaced", input: " a b ", expected: "a_b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizePathSegment(tt.input))
		})
	}
}

func TestIndexPadWidth(t *testing.T) {
	tests := []struct {
		n        int
		expected int
	}{
		{n: 1, expected: 2},
		{n: 99, expected: 2},
		{n: 100, expected: 3},
		{n: 999, expected: 3},
		{n: 1000, expected: 4},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, indexPadWidth(tt.n))
	}
}

func TestArtifactsRelPath(t *testing.T) {
	cwd := "/home/user/project"

	t.Run("suite inside cwd mirrors invocation path", func(t *testing.T) {
		suite := "/home/user/project/examples/aws/suite_xprin.yaml"
		got := artifactsRelPath(suite, cwd)
		assert.Equal(t, filepath.Join("examples", "aws", "suite_xprin"), got)
	})

	t.Run("suite at cwd root uses basename only", func(t *testing.T) {
		suite := "/home/user/project/suite_xprin.yaml"
		got := artifactsRelPath(suite, cwd)
		assert.Equal(t, "suite_xprin", got)
	})

	t.Run("suite outside cwd falls back to absolute path segments", func(t *testing.T) {
		suite := "/other/path/suite_xprin.yaml"
		got := artifactsRelPath(suite, cwd)
		assert.Equal(t, filepath.Join("other", "path", "suite_xprin"), got)
	})
}
