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

package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

func TestParse(t *testing.T) {
	t.Run("a valid template", func(t *testing.T) {
		tmpl, err := Parse("t", "{{ .Name }}")
		require.NoError(t, err)
		assert.Equal(t, "t", tmpl.Name())
	})

	t.Run("a syntax error", func(t *testing.T) {
		_, err := Parse("t", "{{ if .Name }}")
		require.ErrorContains(t, err, "failed to parse template")
	})

	t.Run("an unknown function", func(t *testing.T) {
		_, err := Parse("t", "{{ nope .Name }}")
		require.Error(t, err)
	})

	t.Run("a hyphen in a dotted field", func(t *testing.T) {
		_, err := Parse("t", "{{ .Tests.test-case-1.Outputs.XR }}")
		require.ErrorContains(t, err, "bad character U+002D '-'")
	})
}
