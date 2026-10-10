/*
Copyright 2026 The Crossplane Authors.

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

package discovery

import (
	"errors"
	"testing"

	unittestsUtils "github.com/crossplane-contrib/xprin/internal/unittests/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

// recorder returns a Visitor that records the files it is called with.
func recorder(visited *[]string, err error) Visitor {
	return func(testSuiteFile string) error {
		*visited = append(*visited, testSuiteFile)

		return err
	}
}

func TestWalkDirectory(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/testdir", 0o755))

		var (
			err     error
			visited []string
		)

		out := unittestsUtils.CaptureStderr(func() {
			err = walkDirectory(fs, "/testdir", Options{}, recorder(&visited, nil))
		})

		require.NoError(t, err, "no testsuite files is not an error")
		assert.Contains(t, out, "?   \t/testdir\t[no testsuite files]")
		assert.Empty(t, visited)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		var (
			err     error
			visited []string
		)

		out := unittestsUtils.CaptureStderr(func() {
			err = walkDirectory(afero.NewMemMapFs(), "/nonexistent", Options{}, recorder(&visited, nil))
		})

		require.NoError(t, err)
		assert.Contains(t, out, "?   \t/nonexistent\t[no testsuite files]")
		assert.Empty(t, visited)
	})

	t.Run("quiet suppresses [no testsuite files]", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, fs.MkdirAll("/somedir", 0o755))

		var (
			err     error
			visited []string
		)

		out := unittestsUtils.CaptureStderr(func() {
			err = walkDirectory(fs, "/somedir", Options{Quiet: true}, recorder(&visited, nil))
		})

		require.NoError(t, err)
		assert.NotContains(t, out, "[no testsuite files]")
	})

	t.Run("visits every testsuite file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/testdir/a_xprin.yaml", []byte("dummy"), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/testdir/b_xprin.yaml", []byte("dummy"), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/testdir/not_a_suite.yaml", []byte("dummy"), 0o644))

		var visited []string

		require.NoError(t, walkDirectory(fs, "/testdir", Options{}, recorder(&visited, nil)))
		assert.Equal(t, []string{"/testdir/a_xprin.yaml", "/testdir/b_xprin.yaml"}, visited)
	})

	t.Run("a failing visit makes the directory fail, but every file is still visited", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/testdir/a_xprin.yaml", []byte("dummy"), 0o644))
		require.NoError(t, afero.WriteFile(fs, "/testdir/b_xprin.yaml", []byte("dummy"), 0o644))

		var visited []string

		err := walkDirectory(fs, "/testdir", Options{}, recorder(&visited, errors.New("boom")))
		require.ErrorContains(t, err, "errors occurred processing files in directory /testdir")
		assert.Len(t, visited, 2)
	})
}

func TestWalk(t *testing.T) {
	newFS := func(t *testing.T) afero.Fs {
		t.Helper()

		fs := afero.NewMemMapFs()
		for _, f := range []string{
			"/tests/a_xprin.yaml",
			"/tests/not_a_suite.yaml",
			"/tests/sub/b_xprin.yaml",
			"/tests/sub/deeper/xprin.yaml",
		} {
			require.NoError(t, afero.WriteFile(fs, f, []byte("dummy"), 0o644))
		}

		require.NoError(t, fs.MkdirAll("/tests/empty", 0o755))

		return fs
	}

	tests := map[string]struct {
		targets []string
		want    []string
	}{
		"a testsuite file":                 {[]string{"/tests/a_xprin.yaml"}, []string{"/tests/a_xprin.yaml"}},
		"a file that is not a testsuite":   {[]string{"/tests/not_a_suite.yaml"}, nil},
		"a path that does not exist":       {[]string{"/tests/nope_xprin.yaml"}, nil},
		"a directory is not recursive":     {[]string{"/tests"}, []string{"/tests/a_xprin.yaml"}},
		"... is recursive":                 {[]string{"/tests/..."}, []string{"/tests/a_xprin.yaml", "/tests/sub/b_xprin.yaml", "/tests/sub/deeper/xprin.yaml"}},
		"several targets, in order":        {[]string{"/tests/sub/b_xprin.yaml", "/tests/a_xprin.yaml"}, []string{"/tests/sub/b_xprin.yaml", "/tests/a_xprin.yaml"}},
		"the same file named twice":        {[]string{"/tests/a_xprin.yaml", "/tests/a_xprin.yaml"}, []string{"/tests/a_xprin.yaml", "/tests/a_xprin.yaml"}},
		"a directory without suite files":  {[]string{"/tests/empty"}, nil},
		"recursive from a nested location": {[]string{"/tests/sub/..."}, []string{"/tests/sub/b_xprin.yaml", "/tests/sub/deeper/xprin.yaml"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var visited []string

			err := Walk(newFS(t), tc.targets, Options{Quiet: true}, recorder(&visited, nil))
			require.NoError(t, err)
			assert.Equal(t, tc.want, visited)
		})
	}

	t.Run("a failing visit fails the walk and prints FAIL, but the next target is still visited", func(t *testing.T) {
		var (
			err     error
			visited []string
		)

		out := unittestsUtils.CaptureOutput(func() {
			err = Walk(newFS(t), []string{"/tests/a_xprin.yaml", "/tests/sub/b_xprin.yaml"}, Options{}, recorder(&visited, errors.New("boom")))
		})

		require.ErrorContains(t, err, "processing completed with errors")
		assert.Contains(t, out.Stdout, "FAIL")
		assert.Len(t, visited, 2)
	})
}
