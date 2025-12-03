# Test Suite Specification

Test suite files define the test cases for xprin. They must be named `xprin.yaml` or `*_xprin.yaml`.

## Basic Structure

```yaml
common:
  inputs:
    xr: /path/to/common-xr.yaml  # or claim: /path/to/common-claim.yaml
    composition: /path/to/composition.yaml
    functions: /path/to/functions
    crds:
      - ../../path/to/crd_dir
      - {{ .Repositories.myrepo }}/path/to/crossplane.yaml
    context-files:
      key1: /path/to/context1.yaml
      key2: /path/to/context2.yaml
    context-values:
      key1: "value1"
      key2: "value2"
    observed-resources: /path/to/observed-resources.yaml
    extra-resources: /path/to/extra-resources.yaml
    function-credentials: /path/to/function-credentials.yaml
  patches:
    xrd: /path/to/xrd.yaml
    connection-secret: true
    connection-secret-name: "my-secret"
    connection-secret-namespace: "my-namespace"
  hooks:
    pre-test:
      - name: "setup environment"
        run: "echo 'Setting up test environment'"
      - name: "another pre-test hook"
        run: "echo 'Another pre-test action'"
    post-test:
      - name: "cleanup"
        run: "echo 'Cleaning up test artifacts'"
      - name: "validate common outputs"
        run: "echo 'Validating {{ .Outputs.XR }}'"
  assertions:
    - name: "common-resource-count"
      type: "Count"
      value: 3

tests:
- name: "My Test Case"
  id: "test-case-1"
  inputs:
    xr: xr.yaml
    composition: comp.yaml
    functions: /path/to/functions
    crds:
      - ../../path/to/crd_dir
    context-files:
      key1: /path/to/context1.yaml
    context-values:
      key1: "value1"
    observed-resources: /path/to/observed-resources.yaml
    extra-resources: /path/to/extra-resources.yaml
    function-credentials: /path/to/function-credentials.yaml
  patches:
    xrd: /path/to/xrd.yaml
    connection-secret: true
    connection-secret-name: "my-secret"
    connection-secret-namespace: "my-namespace"
  hooks:
    pre-test:
      - name: "pre-test setup"
        run: "echo 'Pre-test setup for {{ .Inputs.XR }}'"
    post-test:
      - name: "validate outputs"
        run: "echo 'Validating {{ .Outputs.XR }}'"
      - name: "check render count"
        run: "echo 'Rendered {{ .Outputs.RenderCount }} resources'"
  assertions:
    - name: "resource-count"
      type: "Count"
      value: 3
    - name: "deployment-exists"
      type: "Exists"
      resource: "Deployment/my-app"
    - name: "replicas-value"
      type: "FieldValue"
      resource: "Deployment/my-app"
      field: "spec.replicas"
      operator: "=="
      value: 3
- name: "Test: Basic Setup with Claim"
  id: "test-case-2"
  inputs:
    claim: claim.yaml
    composition: comp.yaml
    functions: /path/to/functions
  hooks:
    pre-test:
      - name: "use previous test output"
        run: "echo 'Previous test XR: {{ .Tests.test-case-1.Outputs.XR }}'"
    post-test:
      - name: "compare outputs"
        run: "diff -u {{ .Tests.test-case-1.Outputs.Render }} {{ .Outputs.Render }} || true"
```

## Field Reference

### Root Level

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `common` | ❌ | map | Shared settings for all tests |
| `tests` | ✅ | list | List of test cases |

### Common Section

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `inputs` | ❌ | map | Common inputs for all test cases |
| `patches` | ❌ | map | Common patches for all test cases |
| `hooks` | ❌ | map | Common hooks for all test cases |
| `assertions` | ❌ | list | Common assertions for all test cases |

### Test Case

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | ✅ | string | Test case name (alphanumeric, underscores, hyphens) |
| `id` | ❌ | string | Optional unique test case ID (enables cross-test references and artifact storage) |
| `inputs` | ✅ | map | Inputs for the test case |
| `patches` | ❌ | map | XR patching configuration |
| `hooks` | ❌ | map | Hooks for the test case |
| `assertions` | ❌ | list | Assertions to validate rendered resources |

### Inputs

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `xr` | ✅* | string | Composite Resource file |
| `claim` | ✅* | string | Claim file (mutually exclusive with `xr`) |
| `composition` | ✅ | string | Composition file |
| `functions` | ✅ | string | Path to Crossplane functions |
| `crds` | ❌ | []string | Paths to CRDs for validation |
| `context-files` | ❌ | map[string]string | Context files for render |
| `context-values` | ❌ | map[string]string | Context values for render |
| `observed-resources` | ❌ | string | Path to observed resources file |
| `extra-resources` | ❌ | string | Path to extra resources file |
| `function-credentials` | ❌ | string | Path to function credentials file |

*Either `xr` or `claim` is required, but not both. They can be specified either in the `common` section or in individual test cases. If specified in both, the test case value takes precedence.

### Patches

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `xrd` | ❌ | string | Path to the Claim's XRD file |
| `connection-secret` | ❌ | bool | Enable connection secret testing |
| `connection-secret-name` | ❌ | string | Custom name for connection secret |
| `connection-secret-namespace` | ❌ | string | Custom namespace for connection secret |

### Hooks

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `pre-test` | ❌ | list | Pre-test hooks (execute before test) |
| `post-test` | ❌ | list | Post-test hooks (execute after test) |

### Hook Item

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | ❌ | string | Hook name (used in error messages) |
| `run` | ✅ | string | Shell command to execute |

### Assertions

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `name` | ✅ | string | Assertion name (descriptive identifier) |
| `type` | ✅ | string | Assertion type (see [Assertions Documentation](assertions.md#assertion-types)) |
| `resource` | ✅* | string | Resource identifier (format: `Kind/name` or `Kind` depending on assertion type) |
| `field` | ✅* | string | Field path for field-based assertions (e.g., `metadata.name`, `spec.replicas`) |
| `operator` | ✅* | string | Operator for field value assertions (e.g., `==`, `is`) |
| `value` | ✅* | any | Expected value for count, type, or field value assertions |

*Required fields depend on assertion type. For complete documentation, see [Assertions](assertions.md).

## Path Resolution

Input path fields support:
- **Absolute paths**: `/absolute/path/to/file.yaml`
- **Relative paths**: `relative/path/to/file.yaml` (relative to test suite file)
- **Template variables**: `{{ .Repositories.myrepo }}/path/to/file.yaml`

For detailed information, see [How It Works](how-it-works.md#path-resolution).

## Template Variables

### Repository Variables
- `{{ .Repositories.name }}` - Repository paths from configuration

### Input Variables
Available in hooks and other test case fields:
- `{{ .Inputs.XR }}` - XR file path
- `{{ .Inputs.Claim }}` - Claim file path
- `{{ .Inputs.Composition }}` - Composition file path
- `{{ .Inputs.Functions }}` - Functions directory path
- All other input fields via `{{ .Inputs.FieldName }}`

### Output Variables
Available in post-test hooks only:
- `{{ .Outputs.XR }}` - XR file path
- `{{ .Outputs.Render }}` - Full rendered output path
- `{{ .Outputs.Validate }}` - Validation output path
- `{{ .Outputs.RenderCount }}` - Number of rendered resources
- `{{ index .Outputs.Rendered "Kind/Name" }}` - Individual resource paths

### Cross-test References
Available when test has `id` field:
- `{{ .Tests.{test-id}.Outputs.XR }}` - XR from referenced test
- `{{ .Tests.{test-id}.Outputs.Render }}` - Render output from referenced test
- `{{ .Tests.{test-id}.Outputs.Validate }}` - Validate output from referenced test
- `{{ .Tests.{test-id}.Outputs.RenderCount }}` - Render count from referenced test
- `{{ index .Tests.{test-id}.Outputs.Rendered "Kind/Name" }}` - Individual resource from referenced test

For detailed information, see [How It Works](how-it-works.md#template-variable-expansion) and [How It Works](how-it-works.md#test-chaining-and-artifacts).

## Test Discovery

Test suite files are discovered when they match:
- `xprin.yaml`
- `*_xprin.yaml` pattern

Targets can be:
- Single file: `xprin test file_xprin.yaml`
- Directory: `xprin test mytests/`
- Recursive: `xprin test mytests/...`
- Combination: `xprin test file_xprin.yaml mytests/...`

---

**Next Steps:**
- Learn how to assert rendered resources with [Assertions](assertions.md)
