# Assertions

Assertions provide declarative validation of rendered resources. They allow you to validate the structure and content of rendered manifests without writing custom scripts.

## Overview

Assertions are executed after Crossplane validation (if CRDs are provided) or after rendering (if no CRDs), and before post-test hooks. All assertions are evaluated even if some fail, allowing you to see all validation issues at once.

**Key Features:**
- Declarative validation without custom scripts
- Multiple assertion types (count, existence, field checks)
- Resource patterns (globbing) in xprin assertions: match by exact name, kind only, or shell-style pattern (e.g. `SecurityGroup/group-*`) so one assertion applies to all matching resources
- Golden-file comparison against full render or a single resource file
- Support for common and test-level assertions
- All assertions evaluated even if some fail
- Summary line reports total, successful, failed, and error counts
- Detailed error messages for debugging

Assertion results use the same [statuses and output symbols](how-it-works.md#statuses-and-output-symbols) as other phases ([✓] pass, [x] fail, [!] error).

For information about how assertions work internally, see [How It Works](how-it-works.md#assertions-execution).

## Structure

Assertions are organized by **assertion engine**. xprin supports three engines:

| Engine | Key | Description |
|--------|-----|-------------|
| **xprin** | `assertions.xprin` | In-process assertions: count, existence, field type/value checks on rendered resources. |
| **diff** | `assertions.diff` | Golden-file comparison using a **unified diff** (line-by-line, like `diff -u`). ([go-difflib](https://github.com/pmezard/go-difflib)) |
| **dyff** | `assertions.dyff` | Golden-file comparison using **dyff** (structural YAML diff, document-aware). ([dyff](https://github.com/homeport/dyff)) |

You can use one or more engines in the same test; all assertion results are collected and reported together.

A fourth group, `assertions.errors`, is not an engine: it asserts on a **failure**, where the engines assert on what was **rendered**.

| Asserts on | Key |
|--------|-----|
| What was rendered | `assertions.xprin`, `assertions.diff`, `assertions.dyff` |
| A phase that fails | `assertions.errors` (see [Error assertions (errors)](#error-assertions-errors)) |

```yaml
assertions:
  xprin:
  - name: "resource-count"
    type: "Count"
    value: 3
  diff:
  - name: "Full render matches golden"
    expected: golden_full_render.yaml
  dyff:
  - name: "Full render matches golden (structural)"
    expected: golden_full_render.yaml
```

- **xprin** assertions go under `assertions.xprin` (see [Assertion types (xprin)](#assertion-types-xprin)).
- **diff** and **dyff** assertions go under `assertions.diff` and `assertions.dyff` (see [Golden-file assertions (diff and dyff)](#golden-file-assertions-diff-and-dyff)).

## Golden-file assertions (diff and dyff)

**diff** and **dyff** compare the test’s actual output (full render or a single resource) to a **golden file** (expected YAML). Paths are relative to the test suite file.

### Fields (diff and dyff)

`name` and `expected` are required, `resource` is optional ([specification](testsuite-specification.md#assertionsdiff-and-assertionsdyff)).

### When to use diff vs dyff

- **diff** – Unified diff (line-by-line). Good when you care about exact formatting and line order; failure output looks like `diff -u`. Use when the golden file is hand-written or must match byte-for-byte.
- **dyff** – Structural YAML diff (document-aware, reorders keys). Good when you care about content equality and readability of the diff; reordering keys or formatting won’t fail. Use when you want to ignore formatting and focus on structure and values.

### Examples

**Full render vs golden file (diff and dyff):**

```yaml
assertions:
  diff:
  - name: "Full render matches golden (using diff)"
    expected: golden_full_render.yaml
  dyff:
  - name: "Full render matches golden (using dyff)"
    expected: golden_full_render.yaml
```

**Single resource vs golden file:**

```yaml
assertions:
  diff:
  - name: "Cluster resource matches golden (using diff)"
    expected: golden_single_resource.yaml
    resource: "Cluster/platform-aws-rds"
  dyff:
  - name: "Cluster resource matches golden (using dyff)"
    expected: golden_single_resource.yaml
    resource: "Cluster/platform-aws-rds"
```

When `resource` is set, the runner uses the path of that resource’s rendered file as **actual**; otherwise it uses the path of the full render output.

---

## Error assertions (errors)

`assertions.errors` asserts that a phase **fails**, and optionally why. It is for the negative test cases: an input that a composition has to refuse. A test case that has an assertion on the error of the render expects the render to fail: that failure does not fail the test case, and only these assertions run (the other assertions and the validate step do not, since nothing was rendered). The post-test hooks run after them, as always, but without outputs: `.Outputs` is empty. It passes if its assertions pass and its hooks succeed.

### Fields

`name` and `phase` are required, `operator` and `value` are optional ([specification](testsuite-specification.md#assertionserrors)).

`operator` and `value` go together. Without them, any failure of the phase passes. The output is the combined stdout and stderr of the command of the phase (`crossplane render`).

### Examples

A render that has to fail, for any reason:

```yaml
tests:
- name: "the existing resource stops the render"
  inputs:
    xr: xr.yaml
    composition: composition.yaml
    functions: functions.yaml
    observed-resources: existing-endpoint.yaml
  assertions:
    errors:
    - name: "the render fails"
      phase: render
```

A render that has to fail for a reason, with the message in `common` for all the test cases of the file:

```yaml
common:
  inputs:
    composition: composition.yaml
    functions: functions.yaml
    observed-resources: existing-endpoint.yaml
  assertions:
    errors:
    - name: "the existing resource could not be evaluated"
      phase: render
      operator: contains
      value: "could not be evaluated, abort"

tests:
- name: "attachment in another region"
  inputs:
    xr: attachment_in_other_region/xr.yaml
- name: "subnet in another region"
  inputs:
    xr: subnet_in_other_region/xr.yaml
```

### Behavior

- If the render succeeds, the assertion fails: `the render succeeded, but it was expected to fail`.
- If the render fails and the output is not what `operator` and `value` say, the assertion fails, and its message has the output.
- A test case whose render fails, and that has no assertion on the error of the render, fails as always.
- Without `operator` and `value`, the assertion also passes when the render could not run for another reason (the Docker daemon is not running, an image can't be pulled). Use them to check that it failed for the reason you expect.
- The post-test hooks run after the assertions on the error of the render, whatever they say, as they run after assertions that fail. A hook that uses `.Outputs` gets empty values, so it has to expect that, or the test case has none.
- The test case has no outputs, so a test case that [needs](how-it-works.md#dependencies-between-test-cases-needs) it has nothing of it to use with `.Tests`.
- Like the other groups, `assertions.errors` is merged per group: a test case that has its own list does not get the one of `common`.

---

## Assertion types (xprin)

### Fields

`name` and `type` are required, the others depend on the type ([specification](testsuite-specification.md#assertionsxprin)).

### Resource field: patterns (globbing)

For **xprin** assertions, the `resource` field supports **shell-style globbing (patterns)**. The assertion is applied to every resource that matches the pattern, so you can match resources with unpredictable names and avoid repeating the same assertion for multiple resources.

**Rules:**

- The `/` character cannot be matched by `*`. So you either specify **kind only** or a **full pattern** in the form `Kind/Name`.
- **Kind only** – All resources of that kind are matched. Example: `resource: "SecurityGroup"` matches every `SecurityGroup` resource.
- **Full pattern** – Use `Kind/name-pattern` with `*` in the name part. Examples:
  - `resource: "SecurityGroup/group-*"` – All `SecurityGroup` resources whose name starts with `group-`
  - `resource: "*/*"` – All resources of all kinds
  - `resource: "SecurityGroup/mygroup"` – Exact name (no wildcards); matches only that resource

**Behavior:** For assertion types that use `resource` (Exists, NotExists, FieldType, FieldExists, FieldNotExists, FieldValue), the assertion is evaluated against **each** matching resource. For **Count**, when `resource` is set, the count is the number of resources that match the pattern; when `resource` is omitted, the count is the total number of rendered resources (see [Count](#count)).

### Count

Validates the number of rendered resources. With an optional `resource` pattern, validates the number of resources that match that pattern.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"Count"`
- `value` - Expected count (number)

**Optional Fields:**
- `resource` - If set, only resources matching this pattern are counted. If omitted, all rendered resources are counted (same as before).

**Examples:**
```yaml
assertions:
  xprin:
  # Count all rendered resources 
  - name: "renders-three-resources"
    type: "Count"
    value: 3

  # Count only resources matching a pattern
  - name: "three-security-groups"
    type: "Count"
    resource: "SecurityGroup/group-*"
    value: 3

  # Count all resources of a kind
  - name: "exactly-two-deployments"
    type: "Count"
    resource: "Deployment"
    value: 2
```

**Use Case:** Ensure the total number of rendered resources, or the number of resources matching a pattern, is as expected. Count with `resource` can replace multiple Exists/NotExists assertions when you only care about how many match.

---

### Exists

Validates that at least one resource matching the given pattern exists in the rendered output. Supports exact names, kind only, and [patterns](#resource-field-patterns-globbing).

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"Exists"`
- `resource` - Resource identifier: exact `Kind/name`, kind only, or pattern (e.g., `"Deployment/my-app"`, `"SecurityGroup/group-*"`)

**Example:**
```yaml
assertions:
  xprin:
  - name: "deployment-exists"
    type: "Exists"
    resource: "Deployment/my-app"
  - name: "service-exists"
    type: "Exists"
    resource: "Service/my-app"
  - name: "at-least-one-security-group"
    type: "Exists"
    resource: "SecurityGroup/group-*"
```

**Use Case:** Verify that specific resources (or resources matching a pattern) are created by the composition.

---

### NotExists

Validates that no resource matching the given pattern exists in the rendered output. Supports exact names, kind-only, and [patterns](#resource-field-patterns-globbing).

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"NotExists"`
- `resource` - Resource identifier: exact `Kind/name`, `Kind` (all of that kind), or pattern (e.g., `"Deployment/old-app"`, `"Pod"`, `"Deployment/legacy-*"`)

**Example:**
```yaml
assertions:
  xprin:
  - name: "no-old-deployment"
    type: "NotExists"
    resource: "Deployment/old-app"
  - name: "no-pods"
    type: "NotExists"
    resource: "Pod"
  - name: "no-legacy-security-groups"
    type: "NotExists"
    resource: "SecurityGroup/legacy-*"
```

**Use Case:** Ensure deprecated resources are not created, or verify that certain resource types or name patterns are excluded.

---

### FieldType

Validates the type of a field in each resource matching the given pattern. Applied to every matched resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldType"`
- `resource` - Resource identifier: exact `Kind/name`, kind only, or [pattern](#resource-field-patterns-globbing)
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
  xprin:
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

Validates that a field exists at a given path in each resource matching the given pattern. Applied to every matched resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldExists"`
- `resource` - Resource identifier: exact `Kind/name`, kind only, or [pattern](#resource-field-patterns-globbing)
- `field` - Field path using dot notation (e.g., `"spec.replicas"`, `"metadata.labels.app"`)

**Example:**
```yaml
assertions:
  xprin:
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

Validates that a field does not exist at a given path in each resource matching the given pattern. Applied to every matched resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldNotExists"`
- `resource` - Resource identifier: exact `Kind/name`, kind only, or [pattern](#resource-field-patterns-globbing)
- `field` - Field path using dot notation (e.g., `"spec.deprecated"`)

**Example:**
```yaml
assertions:
  xprin:
  - name: "no-deprecated-field"
    type: "FieldNotExists"
    resource: "Deployment/my-app"
    field: "spec.deprecated"
```

**Use Case:** Ensure deprecated or unwanted fields are not present in rendered resources.

---

### FieldValue

Validates the value of a field in each resource matching the given pattern using comparison operators. Applied to every matched resource.

**Required Fields:**
- `name` - Assertion name
- `type` - Must be `"FieldValue"`
- `resource` - Resource identifier: exact `Kind/name`, kind only, or [pattern](#resource-field-patterns-globbing)
- `field` - Field path using dot notation (e.g., `"spec.replicas"`)
- `operator` - Comparison operator: `"=="` (equals) or `"is"` (string comparison)
- `value` - Expected value (type must match field type)

**Supported Operators:**
- `==` - Equality comparison (works for numbers, strings, booleans)
- `is` - Equality comparison (same as `==`, provided for readability)

**Example:**
```yaml
assertions:
  xprin:
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
    xprin:
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

    # Pattern: assert on all SecurityGroups whose name starts with "group-"
    - name: "security-groups-have-vpc"
      type: "FieldExists"
      resource: "SecurityGroup/group-*"
      field: "spec.forProvider.vpcId"
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
    xprin:
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
    - name: "service-with-unpredictable-name-exists"
      type: "Exists"
      resource: "Service/*-app"

    # Resource non-existence
    - name: "no-old-deployment"
      type: "NotExists"
      resource: "Deployment/old-app"
    - name: "no-pods"
      type: "NotExists"
      resource: "Pod"
    - name: "no-deployments-exist"
      type: "NotExists"
      resource: "Deployment"

    # Field existence
    - name: "has-replicas-field"
      type: "FieldExists"
      resource: "Deployment/my-app"
      field: "spec.replicas"
    - name: "all-deployments-have-team-label"
      type: "FieldExists"
      resource: "Deployment"
      field: "metadata.labels.team"
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
    - name: "all-clusters-with-name-suffix-have-env-prod-label"
      type: "FieldValue"
      resource: "Cluster/*-db"
      field: "metadata.labels.env"
      operator: "is"
      value: "prod"
```

## Common vs Test-Level Assertions

Assertions can be defined in both the `common` section and at the test case level.

### Merging Behavior

Merge is **per group** (`assertions.xprin`, `assertions.diff`, `assertions.dyff`, `assertions.errors`):

- **If the test case has no assertions for a group**: Common’s assertions for that group are used.
- **If the test case has any assertions for a group**: The test case’s list for that group is used (common’s list for that group is not appended).

### Example

```yaml
common:
  assertions:
    xprin:
    - name: "common-count"
      type: "Count"
      value: 3
    - name: "common-exists"
      type: "Exists"
      resource: "Deployment/my-app"
    - name: "two-services-exist"
      type: "Count"
      value: 2
      resource: "Service"

tests:
- name: "Test 1"
  # No assertions defined, so common assertions are used
  inputs:
    xr: xr1.yaml
    composition: comp.yaml
    functions: /path/to/functions

- name: "Test 2"
  # Test case defines xprin only; common's xprin is replaced; common's diff/dyff (if any) would still be used
  inputs:
    xr: xr2.yaml
    composition: comp.yaml
    functions: /path/to/functions
  assertions:
    xprin:
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

**A render that is expected to fail:** with `assertions.errors` for the render (see [Error assertions](#error-assertions-errors)), a failed render ends the test case, and only these assertions run, then the post-test hooks.

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
- Declarative validation (count, existence, field checks) — **xprin**
- Golden-file comparison (full render or single resource vs expected YAML) — **diff** or **dyff**
- Type and value validation — **xprin**
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

