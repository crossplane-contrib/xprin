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

package processor

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestRecursiveDirs(t *testing.T) {
	fs := afero.NewMemMapFs()
	root := "/root"
	sub1 := "/root/sub1"
	sub2 := "/root/sub1/sub2"

	require.NoError(t, fs.MkdirAll(sub1, 0o755))
	require.NoError(t, fs.MkdirAll(sub2, 0o755))

	dirs, err := recursiveDirs(fs, root)
	require.NoError(t, err)
	assert.Contains(t, dirs, root)
	assert.Contains(t, dirs, sub1)
	assert.Contains(t, dirs, sub2)
}

func TestRecursiveDirs_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Test with a non-existent root to simulate an error
	_, err := recursiveDirs(fs, "/nonexistent")
	require.Error(t, err)
}

func TestFindTestSuiteFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := "/testdir"
	subDir1 := "/testdir/dir1"
	subDir2 := "/testdir/dir2"

	require.NoError(t, fs.MkdirAll(tempDir, 0o755))
	require.NoError(t, fs.MkdirAll(subDir1, 0o755))
	require.NoError(t, fs.MkdirAll(subDir2, 0o755))

	// Table-driven tests for file pattern validation
	tests := []struct {
		name        string
		filename    string
		shouldMatch bool
		reason      string
	}{
		// Valid file patterns
		{"exact match", "xprin.yaml", true, "exact pattern match"},
		{"suffix match with prefix", "test1_xprin.yaml", true, "valid suffix pattern with prefix"},
		{"suffix match with different prefix", "whatever_xprin.yaml", true, "valid suffix pattern with different prefix"},
		{"suffix match containing 'not'", "test_not_xprin.yaml", true, "valid suffix pattern containing 'not'"},
		{"long name with suffix", "really_long_name_xprin.yaml", true, "valid suffix pattern with long name"},

		// Invalid file patterns
		{"wrong extension", "xprin.yml", false, "wrong file extension (.yml instead of .yaml)"},
		{"extra character after extension", "xprin.yaml1", false, "extra character after .yaml extension"},
		{"missing underscore", "whateverxprin.yaml", false, "missing underscore before xprin"},
		{"underscore with empty prefix", "_xprin.yaml", false, "underscore but nothing before it"},
		{"missing dot before extension", "test_xprin_yaml", false, "missing dot before yaml extension"},
		{"extra character in middle", "test_xprinTyaml", false, "extra character in the middle of extension"},
	}

	// Create all test files in the main directory
	for _, tt := range tests {
		require.NoError(t, afero.WriteFile(fs, tempDir+"/"+tt.filename, []byte("content"), 0o644))
	}

	// Add some valid files to subdirectories for integration testing
	require.NoError(t, afero.WriteFile(fs, subDir1+"/sub1_xprin.yaml", []byte("content"), 0o644))
	require.NoError(t, afero.WriteFile(fs, subDir1+"/xprin.yaml", []byte("content"), 0o644))
	require.NoError(t, afero.WriteFile(fs, subDir2+"/sub2_xprin.yaml", []byte("content"), 0o644))

	// Test file pattern matching in main directory
	t.Run("file pattern validation", func(t *testing.T) {
		files, err := findTestSuiteFiles(fs, tempDir)
		require.NoError(t, err)

		// Convert to base filenames for easier assertions
		baseNames := make(map[string]bool)
		for _, file := range files {
			baseNames[filepath.Base(file)] = true
		}

		// Test each pattern
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.shouldMatch {
					assert.True(t, baseNames[tt.filename], "File %s should be found: %s", tt.filename, tt.reason)
				} else {
					assert.False(t, baseNames[tt.filename], "File %s should not be found: %s", tt.filename, tt.reason)
				}
			})
		}

		// Verify we found exactly the expected number of valid files
		expectedValidCount := 0

		for _, tt := range tests {
			if tt.shouldMatch {
				expectedValidCount++
			}
		}

		assert.Len(t, files, expectedValidCount, "Should find exactly %d valid files in main directory", expectedValidCount)
	})

	// Test finding files in specific subdirectory
	t.Run("subdirectory search", func(t *testing.T) {
		files, err := findTestSuiteFiles(fs, subDir1)
		require.NoError(t, err)
		assert.Len(t, files, 2, "Should find 2 valid files in subDir1")
	})

	// Test finding specific file
	t.Run("specific file search", func(t *testing.T) {
		files, err := findTestSuiteFiles(fs, tempDir+"/test1_xprin.yaml")
		require.NoError(t, err)
		assert.Len(t, files, 1, "Should find 1 file when specifying exact file path")
		assert.Contains(t, files[0], "test1_xprin.yaml")
	})

	// Test non-existent pattern
	t.Run("non-existent pattern", func(t *testing.T) {
		_, err := findTestSuiteFiles(fs, tempDir+"/nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no test files found")
	})

	// Test invalid pattern syntax
	t.Run("invalid pattern", func(t *testing.T) {
		// afero.Glob returns an error for patterns with syntax errors
		_, err := findTestSuiteFiles(fs, "[]") // Invalid pattern for afero.Glob
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to match pattern")
	})
}

func TestIsValidTestSuiteFileName(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "exact match xprin.yaml",
			filename: "xprin.yaml",
			want:     true,
		},
		{
			name:     "path with xprin.yaml",
			filename: "/path/to/xprin.yaml",
			want:     true,
		},
		{
			name:     "valid prefix with underscore",
			filename: "test_xprin.yaml",
			want:     true,
		},
		{
			name:     "valid prefix with path",
			filename: "/path/to/test_xprin.yaml",
			want:     true,
		},
		{
			name:     "long valid prefix",
			filename: "very_long_prefix_xprin.yaml",
			want:     true,
		},
		{
			name:     "invalid - empty prefix",
			filename: "_xprin.yaml",
			want:     false,
		},
		{
			name:     "invalid - wrong extension",
			filename: "xprin.yml",
			want:     false,
		},
		{
			name:     "invalid - wrong filename",
			filename: "test.yaml",
			want:     false,
		},
		{
			name:     "invalid - suffix after yaml",
			filename: "xprin.yaml.bak",
			want:     false,
		},
		{
			name:     "invalid - suffix after yaml with underscore",
			filename: "test_xprin.yaml1",
			want:     false,
		},
		{
			name:     "invalid - substring of xprin.yaml",
			filename: "xprin-yaml",
			want:     false,
		},
		{
			name:     "invalid - case sensitivity",
			filename: "XPRIN.YAML",
			want:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidTestSuiteFileName(tc.filename)
			assert.Equal(t, tc.want, got)
		})
	}
}
