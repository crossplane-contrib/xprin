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

package runner

import (
	"testing"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"  //nolint:depguard // testify is widely used for testing
	"github.com/stretchr/testify/require" //nolint:depguard // testify is widely used for testing
)

const (
	testResource1File = "resource1.yaml"
	testResource2File = "resource2.yaml"
)

func filterFailedAssertions(results []engine.AssertionResult) []engine.AssertionResult {
	var failed []engine.AssertionResult
	for _, r := range results {
		if r.Status == engine.StatusFail() {
			failed = append(failed, r)
		}
	}

	return failed
}

func TestAssertionExecutor_ExecuteAssertions(t *testing.T) {
	t.Run("executes multiple assertions", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resource1Path := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				testResource1File: resource1Path,
			},
		}

		// Create a test resource file
		err := afero.WriteFile(fs, resource1Path, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertions := []api.AssertionXprin{
			{Name: "count-1", Type: "Count", Value: 1},
			{Name: "exists-1", Type: "Exists", Resource: "Pod/test-pod"},
		}

		allResults := executor.executeAssertionsXprin(assertions)
		assert.Len(t, allResults, 2)
		assert.Equal(t, "count-1", allResults[0].Name)
		assert.Equal(t, "exists-1", allResults[1].Name)
		failed := filterFailedAssertions(allResults)
		assert.Empty(t, failed)
	})

	t.Run("continues execution when one assertion fails", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resource1Path := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				testResource1File: resource1Path,
			},
		}

		// Create a test resource file
		err := afero.WriteFile(fs, resource1Path, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertions := []api.AssertionXprin{
			{Name: "count-wrong", Type: "Count", Value: 999},              // Will fail
			{Name: "exists-ok", Type: "Exists", Resource: "Pod/test-pod"}, // Will pass
		}

		allResults := executor.executeAssertionsXprin(assertions)
		assert.Len(t, allResults, 2)
		assert.Equal(t, engine.StatusFail(), allResults[0].Status)
		assert.Equal(t, engine.StatusPass(), allResults[1].Status)
		failed := filterFailedAssertions(allResults)
		assert.Len(t, failed, 1)
		assert.Equal(t, "count-wrong", failed[0].Name)
	})

	t.Run("handles execution errors gracefully", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		assertions := []api.AssertionXprin{
			{Name: "invalid-type", Type: "InvalidType", Value: "test"},
		}

		allResults := executor.executeAssertionsXprin(assertions)
		assert.Len(t, allResults, 1)
		assert.Equal(t, engine.StatusError(), allResults[0].Status, "invalid/unsupported assertion type is reported as error")
		assert.Contains(t, allResults[0].Message, "unsupported assertion type")
		failed := filterFailedAssertions(allResults)
		assert.Empty(t, failed, "error assertions are not counted as failed")
	})

	t.Run("handles empty assertions list", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		allResults := executor.executeAssertionsXprin([]api.AssertionXprin{})
		assert.Empty(t, allResults)
		assert.Empty(t, filterFailedAssertions(allResults))
	})
}

func TestAssertionExecutor_executeCountAssertion(t *testing.T) {
	t.Run("passes when count matches", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resource1Path := testResource1File
		resource2Path := testResource2File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				testResource1File: resource1Path,
				testResource2File: resource2Path,
			},
		}

		// Create test resource files
		err := afero.WriteFile(fs, resource1Path, []byte("apiVersion: v1\nkind: Pod\n"), 0o644)
		require.NoError(t, err)
		err = afero.WriteFile(fs, resource2Path, []byte("apiVersion: v1\nkind: Pod\n"), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "count-test", Type: "Count", Value: 2}
		result, err := executor.executeCountAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
		assert.Contains(t, result.Message, "found 2 resources")
	})

	t.Run("fails when count does not match", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte("apiVersion: v1\nkind: Pod\n"), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "count-test", Type: "Count", Value: 5}
		result, err := executor.executeCountAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "expected 5 resources, got 1")
	})

	t.Run("handles float64 value from YAML", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte("apiVersion: v1\nkind: Pod\n"), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "count-test", Type: "Count", Value: float64(1)}
		result, err := executor.executeCountAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
	})

	t.Run("fails with invalid value type", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "count-test", Type: "Count", Value: "not-a-number"}
		result, err := executor.executeCountAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "count assertion value must be a number")
	})
}

func TestAssertionExecutor_executeExistsAssertion(t *testing.T) {
	t.Run("passes when resource exists by kind and name", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "exists-test", Type: "Exists", Resource: "Pod/test-pod"}
		result, err := executor.executeExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
		assert.Contains(t, result.Message, "found")
	})

	t.Run("fails when resource does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "exists-test", Type: "Exists", Resource: "Service/my-service"}
		result, err := executor.executeExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "not found")
	})

	t.Run("fails when resource field is missing", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "exists-test", Type: "Exists", Resource: ""}
		result, err := executor.executeExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires resource field")
	})

	t.Run("fails with invalid resource format", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "exists-test", Type: "Exists", Resource: "Pod/name/extra"}
		result, err := executor.executeExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "must be in format")
	})
}

func TestAssertionExecutor_executeNotExistsAssertion(t *testing.T) {
	t.Run("passes when resource does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "not-exists-test", Type: "NotExists", Resource: "Service/my-service"}
		result, err := executor.executeNotExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
		assert.Contains(t, result.Message, "not found (as expected)")
	})

	t.Run("fails when resource exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "not-exists-test", Type: "NotExists", Resource: "Pod/test-pod"}
		result, err := executor.executeNotExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "found (should not exist)")
	})

	t.Run("fails when resource field is missing", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		assertion := api.AssertionXprin{Name: "not-exists-test", Type: "NotExists", Resource: ""}
		result, err := executor.executeNotExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires resource field")
	})
}

func TestAssertionExecutor_executeFieldTypeAssertion(t *testing.T) {
	t.Run("passes when field type matches", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		// YAML numbers are parsed as float64, which becomes "number" type
		assertion := api.AssertionXprin{
			Name:     "field-type-test",
			Type:     "FieldType",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
			Value:    "number",
		}
		result, err := executor.executeFieldTypeAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
	})

	t.Run("fails when field type does not match", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-type-test",
			Type:     "FieldType",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
			Value:    "string",
		}
		result, err := executor.executeFieldTypeAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "expected string")
	})

	t.Run("fails when required fields are missing", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		// Missing resource
		assertion := api.AssertionXprin{Name: "field-type-test", Type: "FieldType", Field: "spec.replicas", Value: "int"}
		result, err := executor.executeFieldTypeAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires resource field")

		// Missing field
		assertion = api.AssertionXprin{Name: "field-type-test", Type: "FieldType", Resource: "Pod/test", Value: "int"}
		result, err = executor.executeFieldTypeAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires field")
	})
}

func TestAssertionExecutor_executeFieldExistsAssertion(t *testing.T) {
	t.Run("passes when field exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-exists-test",
			Type:     "FieldExists",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
		}
		result, err := executor.executeFieldExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
		assert.Contains(t, result.Message, "exists")
	})

	t.Run("fails when field does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec: {}
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-exists-test",
			Type:     "FieldExists",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
		}
		result, err := executor.executeFieldExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "does not exist")
	})
}

func TestAssertionExecutor_executeFieldNotExistsAssertion(t *testing.T) {
	t.Run("passes when field does not exist", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec: {}
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-not-exists-test",
			Type:     "FieldNotExists",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
		}
		result, err := executor.executeFieldNotExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
		assert.Contains(t, result.Message, "does not exist (as expected)")
	})

	t.Run("fails when field exists", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-not-exists-test",
			Type:     "FieldNotExists",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
		}
		result, err := executor.executeFieldNotExistsAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "exists (should not exist)")
	})
}

func TestAssertionExecutor_executeFieldValueAssertion(t *testing.T) {
	t.Run("passes when field value matches with ==", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-value-test",
			Type:     "FieldValue",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
			Operator: "==",
			Value:    float64(3), // YAML numbers are parsed as float64
		}
		result, err := executor.executeFieldValueAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)
	})

	t.Run("fails when field value does not match with ==", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-value-test",
			Type:     "FieldValue",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
			Operator: "==",
			Value:    float64(5), // YAML numbers are parsed as float64
		}
		result, err := executor.executeFieldValueAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusFail(), result.Status)
		assert.Contains(t, result.Message, "expected ==")
	})

	t.Run("fails when required fields are missing", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		// Missing resource
		assertion := api.AssertionXprin{Name: "field-value-test", Type: "FieldValue", Field: "spec.replicas", Operator: "==", Value: float64(3)}
		result, err := executor.executeFieldValueAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires resource field")

		// Missing field
		assertion = api.AssertionXprin{Name: "field-value-test", Type: "FieldValue", Resource: "Pod/test", Operator: "==", Value: float64(3)}
		result, err = executor.executeFieldValueAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires field")

		// Missing operator
		assertion = api.AssertionXprin{Name: "field-value-test", Type: "FieldValue", Resource: "Pod/test", Field: "spec.replicas", Value: 3}
		result, err = executor.executeFieldValueAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires operator field")

		// Missing value
		assertion = api.AssertionXprin{Name: "field-value-test", Type: "FieldValue", Resource: "Pod/test", Field: "spec.replicas", Operator: "=="}
		result, err = executor.executeFieldValueAssertion(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "requires value field")
	})

	t.Run("fails with unsupported operator", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resourcePath := testResource1File
		outputs := &engine.Outputs{
			Rendered: map[string]string{
				"resource1.yaml": resourcePath,
			},
		}

		err := afero.WriteFile(fs, resourcePath, []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  replicas: 3
`), 0o644)
		require.NoError(t, err)

		executor := newAssertionExecutor(fs, outputs, false, "", nil, false)

		assertion := api.AssertionXprin{
			Name:     "field-value-test",
			Type:     "FieldValue",
			Resource: "Pod/test-pod",
			Field:    "spec.replicas",
			Operator: "unsupported",
			Value:    float64(3),
		}
		result, err := executor.executeFieldValueAssertion(assertion)

		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "unsupported operator")
	})
}

func TestAssertionExecutor_executeAssertionXprin(t *testing.T) {
	t.Run("routes to correct assertion type", func(t *testing.T) {
		outputs := &engine.Outputs{
			Rendered: make(map[string]string),
		}

		executor := newAssertionExecutor(afero.NewMemMapFs(), outputs, false, "", nil, false)

		// Test Count
		assertion := api.AssertionXprin{Name: "test", Type: "Count", Value: 0}
		result, err := executor.executeAssertionXprin(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusPass(), result.Status)

		// Test unsupported type
		assertion = api.AssertionXprin{Name: "test", Type: "Unsupported", Value: "test"}
		result, err = executor.executeAssertionXprin(assertion)
		require.NoError(t, err)
		assert.Equal(t, engine.StatusError(), result.Status)
		assert.Contains(t, result.Message, "unsupported assertion type")
	})
}
