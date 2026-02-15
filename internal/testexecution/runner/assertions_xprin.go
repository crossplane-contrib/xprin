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
		assertionResult, _ := e.executeAssertionXprin(assertion)
		results = append(results, assertionResult)
	}

	return results
}

// executeAssertionXprin executes a single xprin assertion.
func (e *assertionExecutor) executeAssertionXprin(assertion api.AssertionXprin) (engine.AssertionResult, error) {
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
		return engine.NewAssertionResult(
			assertion.Name,
			engine.StatusError(),
			fmt.Sprintf("unsupported assertion type: %s", assertion.Type),
		), nil
	}
}

// executeCountAssertion executes a count assertion.
func (e *assertionExecutor) executeCountAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Get the expected count from the assertion value
	expectedCount, ok := assertion.Value.(int)
	if !ok {
		// Try to convert from float64 (YAML numbers)
		if floatVal, ok := assertion.Value.(float64); ok {
			expectedCount = int(floatVal)
		} else {
			return engine.NewAssertionResult(
				assertion.Name,
				engine.StatusError(),
				fmt.Sprintf("count assertion value must be a number, got %T", assertion.Value),
			), nil
		}
	}

	// Count the number of resources in the rendered output
	actualCount := len(e.outputs.Rendered)

	passed := actualCount == expectedCount

	var message string
	if passed {
		message = fmt.Sprintf("found %d resources (as expected)", actualCount)
	} else {
		message = fmt.Sprintf("expected %d resources, got %d", expectedCount, actualCount)
	}

	status := engine.StatusFail()
	if passed {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeExistsAssertion executes an exists assertion.
func (e *assertionExecutor) executeExistsAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Get the expected resource identifier from the assertion resource field
	resourceIdentifier := assertion.Resource
	if resourceIdentifier == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "exists assertion requires resource field"), nil
	}

	// Parse the resource identifier (format: "Kind/name" or "Kind")
	parts := strings.Split(resourceIdentifier, "/")
	if len(parts) != 2 {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("exists assertion value must be in format 'Kind/name', got '%s'", resourceIdentifier)), nil
	}

	expectedKind := parts[0]
	expectedName := parts[1]

	// Search for the resource in rendered outputs
	found := false

	for _, resourcePath := range e.outputs.Rendered {
		// Read the resource file to check its kind and name
		resourceData, err := afero.ReadFile(e.fs, resourcePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Parse the YAML to extract kind and name
		resource := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(resourceData, resource); err != nil {
			continue // Skip invalid YAML
		}

		// Check if this is the resource we're looking for
		if resource.GetKind() == expectedKind && resource.GetName() == expectedName {
			found = true
			break
		}
	}

	var message string
	if found {
		message = fmt.Sprintf("resource %s/%s found", expectedKind, expectedName)
	} else {
		message = fmt.Sprintf("resource %s/%s not found", expectedKind, expectedName)
	}

	status := engine.StatusFail()
	if found {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeNotExistsAssertion executes a not exists assertion.
//

func (e *assertionExecutor) executeNotExistsAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Get the resource identifier from the assertion resource field
	resourceIdentifier := assertion.Resource
	if resourceIdentifier == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "not exists assertion requires resource field"), nil
	}

	// Parse the resource identifier (format: "Kind" or "Kind/name")
	parts := strings.Split(resourceIdentifier, "/")

	var (
		expectedKind, expectedName string
		checkSpecificName          bool
	)

	switch len(parts) {
	case 1:
		// Format: "Kind" - check for any resource of this kind
		expectedKind = parts[0]
		checkSpecificName = false
	case 2:
		// Format: "Kind/name" - check for specific resource
		expectedKind = parts[0]
		expectedName = parts[1]
		checkSpecificName = true
	default:
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("not exists assertion value must be in format 'Kind' or 'Kind/name', got '%s'", resourceIdentifier)), nil
	}

	// Search for the resource in rendered outputs
	found := false

	var foundResources []string

	for _, resourcePath := range e.outputs.Rendered {
		// Read the resource file to check its kind and name
		resourceData, err := afero.ReadFile(e.fs, resourcePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Parse the YAML to extract kind and name
		resource := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(resourceData, resource); err != nil {
			continue // Skip invalid YAML
		}

		// Check if this matches the resource we're looking for
		if resource.GetKind() == expectedKind {
			if !checkSpecificName {
				// Just checking for kind - found a resource of this kind
				foundResources = append(foundResources, fmt.Sprintf("%s/%s", resource.GetKind(), resource.GetName()))
				found = true
			} else if resource.GetName() == expectedName {
				// Checking for specific name
				found = true
				break
			}
		}
	}

	var message string

	if found {
		if checkSpecificName {
			message = fmt.Sprintf("resource %s/%s found (should not exist)", expectedKind, expectedName)
		} else {
			message = fmt.Sprintf("found %d resource(s) of kind %s (should not exist): %s", len(foundResources), expectedKind, strings.Join(foundResources, ", "))
		}
	} else {
		if checkSpecificName {
			message = fmt.Sprintf("resource %s/%s not found (as expected)", expectedKind, expectedName)
		} else {
			message = fmt.Sprintf("no resources of kind %s found (as expected)", expectedKind)
		}
	}

	status := engine.StatusPass()
	if found {
		status = engine.StatusFail()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeFieldTypeAssertion executes a field type assertion.
func (e *assertionExecutor) executeFieldTypeAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires resource field"), nil
	}

	if assertion.Field == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires field"), nil
	}

	if assertion.Value == nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field type assertion requires value field"), nil
	}

	// Get expected type
	expectedType, ok := assertion.Value.(string)
	if !ok {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field type assertion value must be a string, got %T", assertion.Value)), nil
	}

	// Parse the resource identifier (format: "Kind/name")
	parts := strings.Split(assertion.Resource, "/")
	if len(parts) != 2 {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field type assertion resource must be in format 'Kind/name', got '%s'", assertion.Resource)), nil
	}

	expectedKind := parts[0]
	expectedName := parts[1]

	// Find the resource in rendered outputs
	resource, err := e.findResource(expectedKind, expectedName)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error()), nil
	}

	// Navigate to the field value
	fieldValue, err := e.getFieldValue(resource.UnstructuredContent(), assertion.Field)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to get field %s: %v", assertion.Field, err)), nil
	}

	// Check the type
	actualType := e.getGoType(fieldValue)
	passed := actualType == expectedType

	var message string
	if passed {
		message = fmt.Sprintf("field %s has expected type %s", assertion.Field, expectedType)
	} else {
		message = fmt.Sprintf("field %s has type %s, expected %s", assertion.Field, actualType, expectedType)
	}

	status := engine.StatusFail()
	if passed {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeFieldExistsAssertion executes a field exists assertion.
func (e *assertionExecutor) executeFieldExistsAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field exists assertion requires resource field"), nil
	}

	if assertion.Field == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field exists assertion requires field"), nil
	}

	// Parse the resource identifier (format: "Kind/name")
	parts := strings.Split(assertion.Resource, "/")
	if len(parts) != 2 {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field exists assertion resource must be in format 'Kind/name', got '%s'", assertion.Resource)), nil
	}

	expectedKind := parts[0]
	expectedName := parts[1]

	// Find the resource in rendered outputs
	resource, err := e.findResource(expectedKind, expectedName)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error()), nil
	}

	// Check if the field exists
	fieldExists, err := e.checkFieldExists(resource.UnstructuredContent(), assertion.Field)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to check field %s: %v", assertion.Field, err)), nil
	}

	var message string
	if fieldExists {
		message = fmt.Sprintf("field %s exists", assertion.Field)
	} else {
		message = fmt.Sprintf("field %s does not exist", assertion.Field)
	}

	status := engine.StatusFail()
	if fieldExists {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeFieldNotExistsAssertion executes a field not exists assertion.
func (e *assertionExecutor) executeFieldNotExistsAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field not exists assertion requires resource field"), nil
	}

	if assertion.Field == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field not exists assertion requires field"), nil
	}

	// Parse the resource identifier (format: "Kind/name")
	parts := strings.Split(assertion.Resource, "/")
	if len(parts) != 2 {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field not exists assertion resource must be in format 'Kind/name', got '%s'", assertion.Resource)), nil
	}

	expectedKind := parts[0]
	expectedName := parts[1]

	// Find the resource in rendered outputs
	resource, err := e.findResource(expectedKind, expectedName)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error()), nil
	}

	// Check if the field exists
	fieldExists, err := e.checkFieldExists(resource.UnstructuredContent(), assertion.Field)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to check field %s: %v", assertion.Field, err)), nil
	}

	// Pass if field does NOT exist
	passed := !fieldExists

	var message string
	if passed {
		message = fmt.Sprintf("field %s does not exist (as expected)", assertion.Field)
	} else {
		message = fmt.Sprintf("field %s exists (should not exist)", assertion.Field)
	}

	status := engine.StatusFail()
	if passed {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// executeFieldValueAssertion executes a field value assertion.
func (e *assertionExecutor) executeFieldValueAssertion(assertion api.AssertionXprin) (engine.AssertionResult, error) {
	// Validate required fields
	if assertion.Resource == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires resource field"), nil
	}

	if assertion.Field == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires field"), nil
	}

	if assertion.Operator == "" {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires operator field"), nil
	}

	if assertion.Value == nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), "field value assertion requires value field"), nil
	}

	// Parse the resource identifier (format: "Kind/name")
	parts := strings.Split(assertion.Resource, "/")
	if len(parts) != 2 {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("field value assertion resource must be in format 'Kind/name', got '%s'", assertion.Resource)), nil
	}

	expectedKind := parts[0]
	expectedName := parts[1]

	// Find the resource in rendered outputs
	resource, err := e.findResource(expectedKind, expectedName)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), err.Error()), nil
	}

	// Navigate to the field value
	fieldValue, err := e.getFieldValue(resource.UnstructuredContent(), assertion.Field)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to get field %s: %v", assertion.Field, err)), nil
	}

	// Compare the field value with the expected value
	passed, err := e.compareFieldValue(fieldValue, assertion.Operator, assertion.Value)
	if err != nil {
		return engine.NewAssertionResult(assertion.Name, engine.StatusError(), fmt.Sprintf("failed to compare field value: %v", err)), nil
	}

	var message string
	if passed {
		message = fmt.Sprintf("field %s %s %v", assertion.Field, assertion.Operator, assertion.Value)
	} else {
		message = fmt.Sprintf("field %s is %v, expected %s %v", assertion.Field, fieldValue, assertion.Operator, assertion.Value)
	}

	status := engine.StatusFail()
	if passed {
		status = engine.StatusPass()
	}

	return engine.NewAssertionResult(assertion.Name, status, message), nil
}

// findResource finds a resource by kind and name in the rendered outputs.
func (e *assertionExecutor) findResource(expectedKind, expectedName string) (*unstructured.Unstructured, error) {
	for _, resourcePath := range e.outputs.Rendered {
		// Read the resource file to check its kind and name
		resourceData, err := afero.ReadFile(e.fs, resourcePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Parse the YAML to extract kind and name
		resource := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(resourceData, resource); err != nil {
			continue // Skip invalid YAML
		}

		// Check if this is the resource we're looking for
		if resource.GetKind() == expectedKind && resource.GetName() == expectedName {
			return resource, nil
		}
	}

	return nil, fmt.Errorf("resource %s/%s not found", expectedKind, expectedName)
}

// getFieldValue navigates to a field value using dot notation (e.g., "metadata.name").
func (e *assertionExecutor) getFieldValue(obj map[string]interface{}, fieldPath string) (interface{}, error) {
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
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil, fmt.Errorf("field %s is not an object", strings.Join(parts[:i+1], "."))
		}
	}

	return nil, fmt.Errorf("field %s not found", fieldPath)
}

// checkFieldExists checks if a field exists using dot notation (e.g., "metadata.name").
func (e *assertionExecutor) checkFieldExists(obj map[string]interface{}, fieldPath string) (bool, error) {
	parts := strings.Split(fieldPath, ".")
	current := obj

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - check if it exists
			_, exists := current[part]
			return exists, nil
		}

		// Navigate deeper
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return false, fmt.Errorf("field %s is not an object", strings.Join(parts[:i+1], "."))
		}
	}

	return false, fmt.Errorf("field %s not found", fieldPath)
}

// getGoType returns the Go type name for a value.
func (e *assertionExecutor) getGoType(value interface{}) string {
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
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// compareFieldValue compares a field value with an expected value using the specified operator.
func (e *assertionExecutor) compareFieldValue(fieldValue interface{}, operator string, expectedValue interface{}) (bool, error) {
	switch operator {
	case "==", "is":
		return e.compareEqual(fieldValue, expectedValue)
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// compareEqual compares two values for equality.
func (e *assertionExecutor) compareEqual(fieldValue, expectedValue interface{}) (bool, error) {
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
