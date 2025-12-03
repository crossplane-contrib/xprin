# Assertions

Assertions provide declarative validation of rendered resources. They allow you to validate the structure and content of rendered manifests without writing custom scripts.

## Overview

Assertions are executed after Crossplane validation (if CRDs are provided) or after rendering (if no CRDs), and before post-test hooks. All assertions are evaluated even if some fail, allowing you to see all validation issues at once.

**Key Features:**
- Declarative validation without custom scripts
- Multiple assertion types (count, existence, field checks)
- Support for common and test-level assertions
- All assertions evaluated even if some fail
- Detailed error messages for debugging

For information about how assertions work internally, see [How It Works](how-it-works.md#assertions-execution).

## Field Reference

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | ✅ | string | Assertion name (descriptive identifier) |
| `type` | ✅ | string | Assertion type (see [Assertion Types](#assertion-types)) |
| `resource` | ✅* | string | Resource identifier (format: `Kind/name` or `Kind` depending on assertion type) |
| `field` | ✅* | string | Field path for field-based assertions (e.g., `metadata.name`, `spec.replicas`) |
| `operator` | ✅* | string | Operator for field value assertions (e.g., `==`, `is`) |
| `value` | ✅* | any | Expected value for count, type, or field value assertions |

*Required fields depend on assertion type (see [Assertion Types](#assertion-types))

## Assertion Types

### Count

Validates the total number of rendered resources.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"Count"`
- `value` - Expected resource count (number)

**Example:**
```yaml
assertions:
  - name: "renders-three-resources"
    type: "Count"
    value: 3
```

**Use Case:** Ensure a composition renders exactly the expected number of resources.

---

### Exists

Validates that a specific resource exists in the rendered output.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"Exists"`
- `resource` - Resource identifier in format `Kind/name` (e.g., `"Deployment/my-app"`)

**Example:**
```yaml
assertions:
  - name: "deployment-exists"
    type: "Exists"
    resource: "Deployment/my-app"
  - name: "service-exists"
    type: "Exists"
    resource: "Service/my-app"
```

**Use Case:** Verify that specific resources are created by the composition.

---

### NotExists

Validates that a resource does not exist in the rendered output.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"NotExists"`
- `resource` - Resource identifier in format `Kind/name` or `Kind` (e.g., `"Deployment/old-app"` or `"Pod"`)

**Example:**
```yaml
assertions:
  - name: "no-old-deployment"
    type: "NotExists"
    resource: "Deployment/old-app"
  - name: "no-pods"
    type: "NotExists"
    resource: "Pod"
```

**Use Case:** Ensure deprecated resources are not created, or verify that certain resource types are excluded.

---

### FieldType

Validates the type of a field in a resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldType"`
- `resource` - Resource identifier in format `Kind/name`
- `field` - Field path using dot notation (e.g., `"spec.replicas"`, `"metadata.labels.app"`)
- `value` - Expected type: `"string"`, `"number"`, `"boolean"`, `"array"`, `"object"`, or `"null"`

**Supported Types:**
- `string` - Text values
- `number` - Numeric values (integers and floats)
- `boolean` - True/false values
- `array` - List values
- `object` - Map/dict values
- `null` - Null/empty values

**Example:**
```yaml
assertions:
  - name: "replicas-is-number"
    type: "FieldType"
    resource: "Deployment/my-app"
    field: "spec.replicas"
    value: "number"
  - name: "name-is-string"
    type: "FieldType"
    resource: "Deployment/my-app"
    field: "metadata.name"
    value: "string"
  - name: "labels-is-object"
    type: "FieldType"
    resource: "Deployment/my-app"
    field: "metadata.labels"
    value: "object"
  - name: "ports-is-array"
    type: "FieldType"
    resource: "Service/my-app"
    field: "spec.ports"
    value: "array"
  - name: "enabled-is-boolean"
    type: "FieldType"
    resource: "Deployment/my-app"
    field: "spec.enabled"
    value: "boolean"
  - name: "optional-field-is-null"
    type: "FieldType"
    resource: "Deployment/my-app"
    field: "spec.optionalField"
    value: "null"
```

**Use Case:** Validate that fields have the correct data types, ensuring type safety in rendered manifests.

---

### FieldExists

Validates that a field exists at a given path in a resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldExists"`
- `resource` - Resource identifier in format `Kind/name`
- `field` - Field path using dot notation (e.g., `"spec.replicas"`, `"metadata.labels.app"`)

**Example:**
```yaml
assertions:
  - name: "has-replicas-field"
    type: "FieldExists"
    resource: "Deployment/my-app"
    field: "spec.replicas"
  - name: "has-selector"
    type: "FieldExists"
    resource: "Service/my-app"
    field: "spec.selector"
```

**Use Case:** Ensure required fields are present in rendered resources.

---

### FieldNotExists

Validates that a field does not exist at a given path in a resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldNotExists"`
- `resource` - Resource identifier in format `Kind/name`
- `field` - Field path using dot notation (e.g., `"spec.deprecated"`)

**Example:**
```yaml
assertions:
  - name: "no-deprecated-field"
    type: "FieldNotExists"
    resource: "Deployment/my-app"
    field: "spec.deprecated"
```

**Use Case:** Ensure deprecated or unwanted fields are not present in rendered resources.

---

### FieldValue

Validates the value of a field in a resource using comparison operators.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldValue"`
- `resource` - Resource identifier in format `Kind/name`
- `field` - Field path using dot notation (e.g., `"spec.replicas"`)
- `operator` - Comparison operator: `"=="` (equals) or `"is"` (string comparison)
- `value` - Expected value (type must match field type)

**Supported Operators:**
- `==` - Equality comparison (works for numbers, strings, booleans)
- `is` - Equality comparison (same as `==`, provided for readability)

**Example:**
```yaml
assertions:
  - name: "replicas-equals-three"
    type: "FieldValue"
    resource: "Deployment/my-app"
    field: "spec.replicas"
    operator: "=="
    value: 3
  - name: "engine-is-postgresql"
    type: "FieldValue"
    resource: "Cluster/my-db"
    field: "spec.forProvider.engine"
    operator: "is"
    value: "postgresql"
```

**Use Case:** Validate specific field values match expected values.

**Note:** YAML numbers are parsed as `float64`, so numeric comparisons should account for this (e.g., `value: 3` is treated as `float64(3)`).

---

## Complete Examples

### Basic Example

```yaml
tests:
  - name: "Application Deployment"
    inputs:
      xr: app-xr.yaml
      composition: app-composition.yaml
      functions: /path/to/functions
      crds:
        - /path/to/crds
    assertions:
      # Count validation
      - name: "renders-three-resources"
        type: "Count"
        value: 3
      
      # Resource existence
      - name: "deployment-exists"
        type: "Exists"
        resource: "Deployment/my-app"
      - name: "service-exists"
        type: "Exists"
        resource: "Service/my-app"
      
      # Field validation
      - name: "deployment-replicas"
        type: "FieldValue"
        resource: "Deployment/my-app"
        field: "spec.replicas"
        operator: "=="
        value: 3
      
      - name: "service-type"
        type: "FieldType"
        resource: "Service/my-app"
        field: "spec.type"
        value: "string"
      
      - name: "has-selector"
        type: "FieldExists"
        resource: "Service/my-app"
        field: "spec.selector"
```

### Comprehensive Example

```yaml
tests:
  - name: "Comprehensive Assertions Example"
    inputs:
      xr: xr.yaml
      composition: comp.yaml
      functions: /path/to/functions
      crds:
        - /path/to/crds
    assertions:
      # Count assertion
      - name: "renders-three-resources"
        type: "Count"
        value: 3
      
      # Resource existence
      - name: "deployment-exists"
        type: "Exists"
        resource: "Deployment/my-app"
      - name: "service-exists"
        type: "Exists"
        resource: "Service/my-app"
      
      # Resource non-existence
      - name: "no-old-deployment"
        type: "NotExists"
        resource: "Deployment/old-app"
      - name: "no-pods"
        type: "NotExists"
        resource: "Pod"
      
      # Field existence
      - name: "has-replicas-field"
        type: "FieldExists"
        resource: "Deployment/my-app"
        field: "spec.replicas"
      - name: "no-deprecated-field"
        type: "FieldNotExists"
        resource: "Deployment/my-app"
        field: "spec.deprecated"
      
      # Field type validation (all supported types)
      - name: "replicas-is-number"
        type: "FieldType"
        resource: "Deployment/my-app"
        field: "spec.replicas"
        value: "number"
      - name: "name-is-string"
        type: "FieldType"
        resource: "Deployment/my-app"
        field: "metadata.name"
        value: "string"
      - name: "labels-is-object"
        type: "FieldType"
        resource: "Deployment/my-app"
        field: "metadata.labels"
        value: "object"
      - name: "ports-is-array"
        type: "FieldType"
        resource: "Service/my-app"
        field: "spec.ports"
        value: "array"
      - name: "enabled-is-boolean"
        type: "FieldType"
        resource: "Deployment/my-app"
        field: "spec.enabled"
        value: "boolean"
      - name: "optional-field-is-null"
        type: "FieldType"
        resource: "Deployment/my-app"
        field: "spec.optionalField"
        value: "null"
      
      # Field value validation
      - name: "replicas-equals-three"
        type: "FieldValue"
        resource: "Deployment/my-app"
        field: "spec.replicas"
        operator: "=="
        value: 3
      - name: "engine-is-postgresql"
        type: "FieldValue"
        resource: "Cluster/my-db"
        field: "spec.forProvider.engine"
        operator: "is"
        value: "postgresql"
```

## Common vs Test-Level Assertions

Assertions can be defined in both the `common` section and at the test case level.

### Merging Behavior

- **If test case has assertions**: Test case assertions completely replace common assertions (no appending)
- **If test case has no assertions**: Common assertions are used

### Example

```yaml
common:
  assertions:
    - name: "common-count"
      type: "Count"
      value: 3
    - name: "common-exists"
      type: "Exists"
      resource: "Deployment/my-app"

tests:
  - name: "Test 1"
    # No assertions defined, so common assertions are used
    inputs:
      xr: xr1.yaml
      composition: comp.yaml
      functions: /path/to/functions
  
  - name: "Test 2"
    # Test case assertions replace common assertions
    inputs:
      xr: xr2.yaml
      composition: comp.yaml
      functions: /path/to/functions
    assertions:
      - name: "test2-count"
        type: "Count"
        value: 5
```

For detailed information about merging logic, see [How It Works](how-it-works.md#common-vs-test-level-configuration).

## Field Path Syntax

Field paths use dot notation to navigate nested structures:

- `metadata.name` - Top-level field
- `spec.replicas` - Nested field
- `metadata.labels.app` - Deeply nested field
- `spec.forProvider.engine` - Multiple levels of nesting

Field access handles:
- Missing fields (returns null/error)
- Null values (treated as `null` type)
- Array indexing (not directly supported, use array operations)

## Execution and Error Handling

**Execution Order:**
1. Assertions run after validation (if CRDs are provided) or after rendering (if no CRDs)
2. All assertions are evaluated sequentially
3. Results are collected (pass/fail with messages)
4. All results are reported at the end

**Error Handling:**
- All assertions are evaluated even if some fail
- Failed assertions are collected and reported together
- If assertions fail, the test continues to post-test hooks
- This allows cleanup and additional validation even when assertions fail

**Viewing Results:**
- Use `--show-assertions` with `--verbose` to see assertion results in output
- Failed assertions show the assertion name and failure message

For detailed information about execution and error handling, see [How It Works](how-it-works.md#error-handling-behavior).

## When to Use Assertions vs Hooks

**Use Assertions for:**
- Declarative validation (count, existence, field checks)
- Type validation
- Value validation
- Simple, repeatable checks

**Use Hooks for:**
- Complex operations
- External tool integration (Kyverno, UpTest, etc.)
- Custom validation logic that requires scripts
- Operations that need shell commands

Assertions and hooks complement each other - use assertions for simple validation, hooks for complex operations.

---

**Next Steps:**
- Understand how xprin works internally in [How It Works](how-it-works.md)

