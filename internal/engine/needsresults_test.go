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

package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

func TestGetNeedsResults(t *testing.T) {
	failed := NewTestCaseResult("A", "a", false, false, false, false, false).Fail(errors.New("boom"))
	skipped := NewTestCaseResult("B", "b", false, false, false, false, false).Skip(errors.New(`needs failed test case "a"`))
	passed := NewTestCaseResult("C", "c", false, false, false, false, false)
	// Skipped without a reason (update-goldens, no golden files): it can't be waited for either
	skippedAlone := NewTestCaseResult("D", "d", false, false, false, false, false).Skip()

	completed := map[string]*TestCaseResult{"a": failed, "b": skipped, "c": passed, "d": skippedAlone}

	t.Run("only the ones listed", func(t *testing.T) {
		results, skipReasons := GetNeedsResults([]string{"c"}, completed)

		assert.Equal(t, map[string]*TestCaseResult{"c": passed}, results)
		assert.Empty(t, skipReasons)
	})

	t.Run("a testcase without needs gets nothing", func(t *testing.T) {
		results, skipReasons := GetNeedsResults(nil, completed)

		assert.Empty(t, results)
		assert.Empty(t, skipReasons)
	})

	t.Run("the ones that have not run are in neither", func(t *testing.T) {
		results, skipReasons := GetNeedsResults([]string{"c", "not-run"}, completed)

		assert.Equal(t, map[string]*TestCaseResult{"c": passed}, results)
		assert.Empty(t, skipReasons)
	})

	t.Run("a reason for each failed or skipped one, in the order of needs; the results have all of them", func(t *testing.T) {
		results, skipReasons := GetNeedsResults([]string{"b", "c", "a", "d"}, completed)

		assert.Len(t, results, 4)
		assert.EqualError(t,
			errors.Join(skipReasons...),
			"needs skipped test case \"b\"\nneeds failed test case \"a\"\nneeds skipped test case \"d\"")
	})
}
