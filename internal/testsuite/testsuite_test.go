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

package testsuite

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/stretchr/testify/assert" //nolint:depguard // testify is widely used for testing
)

func TestNew(t *testing.T) {
	t.Run("a spec: the graph is the one of its test cases", func(t *testing.T) {
		// "one" is listed first, but needs "two"
		spec := &api.TestSuiteSpec{Tests: []api.TestCase{
			{Name: "One", ID: "one", Needs: []string{"two"}},
			{Name: "Two", ID: "two"},
		}}

		testSuite := New("/tests/a_xprin.yaml", spec)

		assert.Equal(t, "/tests/a_xprin.yaml", testSuite.File)
		assert.Same(t, spec, testSuite.Spec)
		assert.Equal(t, []int{1, 0}, testSuite.Graph.GetOrder())
	})

	t.Run("no spec: no graph", func(t *testing.T) {
		testSuite := New("/tests/a_xprin.yaml", nil)

		assert.Nil(t, testSuite.Spec)
		assert.Nil(t, testSuite.Graph)
	})
}

func TestTestSuite_GetDir(t *testing.T) {
	assert.Equal(t, "/tests/aws", New("/tests/aws/a_xprin.yaml", nil).GetDir())
	assert.Equal(t, ".", New("a_xprin.yaml", nil).GetDir(), "a file in the current directory")
}
