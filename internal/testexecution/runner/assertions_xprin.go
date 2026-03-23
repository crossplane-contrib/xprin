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

// Package runner provides test execution functionality including assertion evaluation.
package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/crossplane-contrib/xprin/internal/api"
	"github.com/crossplane-contrib/xprin/internal/engine"
	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// executeAssertionsXprin executes all xprin assertions for a test case.
func (e *assertionExecutor) executeAssertionsXprin(assertions []api.AssertionXprin) []engine.AssertionResult {
	results := make([]engine.AssertionResult, 0, len(assertions))
	for _, assertion := range assertions {
		assertionResults, _ := e.executeAssertionXprin(assertion)
		results = append(results, assertionResults...)
	}

	return results
}

// executeAssertionXprin executes a single xprin assertion.
func (e *assertionExecutor) executeAssertionXprin(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	switch assertion.Type {
	case "Count":
		return e.executeCountAssertion(assertion)
	case "Exists":
		return e.executeExistsAssertion(assertion)
	case "NotExists":
		return e.executeNotExistsAssertion(assertion)
	case "FieldType":
		return e.executeFieldTypeAssertion(assertion)
	case "FieldExists":
		return e.executeFieldExistsAssertion(assertion)
	case "FieldNotExists":
		return e.executeFieldNotExistsAssertion(assertion)
	case "FieldValue":
		return e.executeFieldValueAssertion(assertion)
	default:
		return []engine.AssertionResult{engine.NewAssertionResult(
			assertion.Name,
			engine.StatusError(),
			fmt.Sprintf("unsupported assertion type: %s", assertion.Type),
		)}, nil
	}
}

// executeCountAssertion executes a count assertion.
func (e *assertionExecutor) executeCountAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Get the expected count from the assertion value
	expectedCount, ok := assertion.Value.(int)
	if !ok {
		// Try to convert from float64 (YAML numbers)
		if floatVal, ok := assertion.Value.(float64); ok {
			expectedCount = int(floatVal)
		} else {
			return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("count assertion value must be a number, got %T", assertion.Value))}, nil
		}
	}
	if expectedCount < 0 {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("count assertion value must be non-negative, got %d", expectedCount))}, nil
	}

	var actualCount int
	var resources []*unstructured.Unstructured
	if assertion.Resource == "" {
		// Count the number of all resources in the rendered output
		actualCount = len(e.outputs.Rendered)
	} else {
		// Count only the resources that match a certain pattern
		var err error
		resources, err = e.findResources(assertion.Resource)
		if err != nil {
			return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
		}
		actualCount = len(resources)
	}

	passed := actualCount == expectedCount

	var message string
	if passed {
		if len(resources) > 0 {
			message = fmt.Sprintf("found %d resources (as expected): %s", actualCount, toIdentifiersString(resources))
		} else {
			message = fmt.Sprintf("found %d resources (as expected)", actualCount)
		}
	} else {
		if len(resources) > 0 {
			message = fmt.Sprintf("expected %d resources, got %d resources: %s", expectedCount, actualCount, toIdentifiersString(resources))
		} else {
			message = fmt.Sprintf("expected %d resources, got %d", expectedCount, actualCount)
		}
	}

	status := engine.StatusFail()
	if passed {
		status = engine.StatusPass()
	}

	return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, status, message)}, nil
}

// executeExistsAssertion executes an exists assertion.
func (e *assertionExecutor) executeExistsAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Get the expected resource identifier from the assertion resource field
	resourceIdentifier := assertion.Resource
	if resourceIdentifier == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "exists assertion requires resource field")}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	var message string
	if len(resources) > 0 {
		message = fmt.Sprintf("resources found: %s", toIdentifiersString(resources))
	} else {
		message = fmt.Sprintf("resource %s not found", assertion.Resource)
	}

	status := engine.StatusFail()
	if len(resources) > 0 {
		status = engine.StatusPass()
	}

	return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, status, message)}, nil
}

// executeNotExistsAssertion executes a not exists assertion.
func (e *assertionExecutor) executeNotExistsAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Get the resource identifier from the assertion resource field
	resourceIdentifier := assertion.Resource
	if resourceIdentifier == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "not exists assertion requires resource field")}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	var message string
	if len(resources) == 0 {
		message = fmt.Sprintf("resource %s not found (as expected)", assertion.Resource)
	} else {
		message = fmt.Sprintf("resources found (should not exist): %s", toIdentifiersString(resources))
	}

	status := engine.StatusFail()
	if len(resources) == 0 {
		status = engine.StatusPass()
	}

	return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, status, message)}, nil
}

// executeFieldTypeAssertion executes a field type assertion.
func (e *assertionExecutor) executeFieldTypeAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires resource field")}, nil
	}

	if assertion.Field == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires field")}, nil
	}

	if assertion.Value == nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires value field")}, nil
	}

	// Get expected type
	expectedType, ok := assertion.Value.(string)
	if !ok {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field type assertion value must be a string, got %T", assertion.Value))}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	if len(resources) == 0 {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("no rendered resource matched the given name %s", assertion.Resource))}, nil
	}

	results := make([]engine.AssertionResult, len(resources))
	for i, resource := range resources {
		// Navigate to the field value
		fieldValue, err := e.getFieldValue(resource.UnstructuredContent(), assertion.Field)
		if err != nil {
			results[i] = engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to get field %s: %v on resource %s", assertion.Field, err, resourceID(resource)))
			continue
		}

		// Check the type
		actualType := e.getGoType(fieldValue)
		passed := actualType == expectedType

		var message string
		if passed {
			message = fmt.Sprintf("field %s has expected type %s on resource %s", assertion.Field, expectedType, resourceID(resource))
		} else {
			message = fmt.Sprintf("field %s has type %s, expected %s on resource %s", assertion.Field, actualType, expectedType, resourceID(resource))
		}

		status := engine.StatusFail()
		if passed {
			status = engine.StatusPass()
		}

		results[i] = engine.NewAssertionResult(assertion.Name, status, message)
	}

	return results, nil
}

// executeFieldExistsAssertion executes a field exists assertion.
func (e *assertionExecutor) executeFieldExistsAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field exists assertion requires resource field")}, nil
	}

	if assertion.Field == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field exists assertion requires field")}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	if len(resources) == 0 {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("no rendered resource matched the given name %s", assertion.Resource))}, nil
	}

	results := make([]engine.AssertionResult, len(resources))
	for i, resource := range resources {
		// Check if the field exists
		fieldExists, err := e.checkFieldExists(resource.UnstructuredContent(), assertion.Field)
		if err != nil {
			results[i] = engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to check field %s: %v on resource %s", assertion.Field, err, resourceID(resource)))
			continue
		}

		var message string
		if fieldExists {
			message = fmt.Sprintf("field %s exists on resource %s", assertion.Field, resourceID(resource))
		} else {
			message = fmt.Sprintf("field %s does not exist on resource %s", assertion.Field, resourceID(resource))
		}

		status := engine.StatusFail()
		if fieldExists {
			status = engine.StatusPass()
		}

		results[i] = engine.NewAssertionResult(assertion.Name, status, message)
	}

	return results, nil
}

// executeFieldNotExistsAssertion executes a field not exists assertion.
func (e *assertionExecutor) executeFieldNotExistsAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field not exists assertion requires resource field")}, nil
	}

	if assertion.Field == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field not exists assertion requires field")}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	if len(resources) == 0 {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("no rendered resource matched the given name %s", assertion.Resource))}, nil
	}

	results := make([]engine.AssertionResult, len(resources))
	for i, resource := range resources {
		// Check if the field exists
		fieldExists, err := e.checkFieldExists(resource.UnstructuredContent(), assertion.Field)
		if err != nil {
			results[i] = engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to check field %s: %v on resource %s", assertion.Field, err, resourceID(resource)))
			continue
		}

		// Pass if field does NOT exist
		passed := !fieldExists

		var message string
		if passed {
			message = fmt.Sprintf("field %s does not exist (as expected) on resource %s", assertion.Field, resourceID(resource))
		} else {
			message = fmt.Sprintf("field %s exists (should not exist) on resource %s", assertion.Field, resourceID(resource))
		}

		status := engine.StatusFail()
		if passed {
			status = engine.StatusPass()
		}

		results[i] = engine.NewAssertionResult(assertion.Name, status, message)
	}

	return results, nil
}

// executeFieldValueAssertion executes a field value assertion.
func (e *assertionExecutor) executeFieldValueAssertion(assertion api.AssertionXprin) ([]engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires resource field")}, nil
	}

	if assertion.Field == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires field")}, nil
	}

	if assertion.Operator == "" {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires operator field")}, nil
	}

	if assertion.Value == nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires value field")}, nil
	}

	resources, err := e.findResources(assertion.Resource)
	if err != nil {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error())}, nil
	}

	if len(resources) == 0 {
		return []engine.AssertionResult{engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("no rendered resource matched the given name %s", assertion.Resource))}, nil
	}

	results := make([]engine.AssertionResult, len(resources))
	for i, resource := range resources {
		// Navigate to the field value
		fieldValue, err := e.getFieldValue(resource.UnstructuredContent(), assertion.Field)
		if err != nil {
			results[i] = engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to get field %s: %v on resource %s", assertion.Field, err, resourceID(resource)))
			continue
		}

		// Compare the field value with the expected value
		passed, err := e.compareFieldValue(fieldValue, assertion.Operator, assertion.Value)
		if err != nil {
			results[i] = engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to compare field value: %v on resource %s", err, resourceID(resource)))
			continue
		}

		var message string
		if passed {
			message = fmt.Sprintf("field %s %s %v on resource %s", assertion.Field, assertion.Operator, assertion.Value, resourceID(resource))
		} else {
			message = fmt.Sprintf("field %s is %v, expected %s %v on resource %s", assertion.Field, fieldValue, assertion.Operator, assertion.Value, resourceID(resource))
		}

		status := engine.StatusFail()
		if passed {
			status = engine.StatusPass()
		}
		results[i] = engine.NewAssertionResult(assertion.Name, status, message)
	}
	return results, nil
}

func (e *assertionExecutor) findResources(pattern string) ([]*unstructured.Unstructured, error) {
	matchedResources := []*unstructured.Unstructured{}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("resource pattern cannot be empty")
	}

	slashCnt := strings.Count(pattern, "/")
	if slashCnt > 1 {
		return nil, fmt.Errorf("the name pattern must be in format 'Kind' or 'Kind/Name'")
	}

	// If only the kind is specified, add '/*' to the pattern to match all resource names for that kind
	if slashCnt == 0 {
		pattern += "/*"
	}

	for resourceIdentifier, resourcePath := range e.outputs.Rendered {
		isMatched, err := filepath.Match(pattern, resourceIdentifier)
		if err != nil {
			return nil, fmt.Errorf("invalid resource pattern %q: %w", pattern, err)
		}
		if !isMatched {
			continue
		}

		resourceData, err := afero.ReadFile(e.fs, resourcePath)
		if err != nil {
			return nil, fmt.Errorf("could not read rendered data for %s resource from file %s", resourceIdentifier, resourcePath)
		}

		// Parse the YAML to extract kind and name
		resource := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(resourceData, resource); err != nil {
			return nil, fmt.Errorf("invalid YAML for resource %s", resourceIdentifier)
		}

		matchedResources = append(matchedResources, resource)
	}

	return matchedResources, nil
}

// getFieldValue navigates to a field value using dot notation (e.g., "metadata.name").
func (e *assertionExecutor) getFieldValue(obj map[string]any, fieldPath string) (any, error) {
	parts := strings.Split(fieldPath, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			if value, exists := current[part]; exists {
				return value, nil
			}

			return nil, fmt.Errorf("field %s not found", fieldPath)
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return nil, fmt.Errorf("field %s is not an object", strings.Join(parts[:i+1], "."))
		}
	}

	return nil, fmt.Errorf("field %s not found", fieldPath)
}

// checkFieldExists checks if a field exists using dot notation (e.g., "metadata.name").
func (e *assertionExecutor) checkFieldExists(obj map[string]any, fieldPath string) (bool, error) {
	parts := strings.Split(fieldPath, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - check if it exists
			_, exists := current[part]
			return exists, nil
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return false, fmt.Errorf("field %s is not an object", strings.Join(parts[:i+1], "."))
		}
	}

	return false, fmt.Errorf("field %s not found", fieldPath)
}

// getGoType returns the Go type name for a value.
func (e *assertionExecutor) getGoType(value any) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// compareFieldValue compares a field value with an expected value using the specified operator.
func (e *assertionExecutor) compareFieldValue(fieldValue any, operator string, expectedValue any) (bool, error) {
	switch operator {
	case "==", "is":
		return e.compareEqual(fieldValue, expectedValue)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareEqual compares two values for equality.
func (e *assertionExecutor) compareEqual(fieldValue, expectedValue any) (bool, error) {
	// Handle nil values
	if fieldValue == nil && expectedValue == nil {
		return true, nil
	}

	if fieldValue == nil || expectedValue == nil {
		return false, nil
	}

	// Convert both values to strings for comparison
	fieldStr := fmt.Sprintf("%v", fieldValue)
	expectedStr := fmt.Sprintf("%v", expectedValue)

	return fieldStr == expectedStr, nil
}

func resourceID(r *unstructured.Unstructured) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", r.GetKind(), r.GetName())
}

func toIdentifiersString(resources []*unstructured.Unstructured) string {
	identifiers := make([]string, len(resources))
	for i, r := range resources {
		identifiers[i] = resourceID(r)
	}
	return strings.Join(identifiers, ", ")
}
