# Getting Started

> **📖 Examples**: For step-by-step examples with real outputs, see the [Examples README](../examples/README.md).

## Table of Contents

- [Command Examples](#command-examples)
  - [How to Run Tests](#how-to-run-tests)
  - [Common Command Options](#common-command-options)
  - [Configuration Management](#configuration-management)
- [Testsuite examples](#testsuite-examples)
  - [Simple Test Suite](#simple-test-suite)
  - [Common Inputs](#common-inputs)
  - [Patching](#patching)
  - [Hooks](#hooks)
    - [Pre-test hooks](#pre-test-hooks)
    - [Post-test hooks](#post-test-hooks)
  - [Assertions](#assertions)
  - [Test Chaining](#test-chaining)
- [Output Examples](#output-examples)
  - [Successful Test Run](#successful-test-run)
  - [Verbose Output](#verbose-output)
  - [Failed Test Run](#failed-test-run)
  - [Show Rendered Resources](#show-rendered-resources)
  - [Show Validation Results](#show-validation-results)
- [Integration Examples](#integration-examples)
  - [With CI/CD Pipeline](#with-cicd-pipeline)

---

## Command Examples

### How to Run Tests

`xprin` supports running tests in several ways. You can run a single file, multiple files, all files in a directory, recursively, or any combination of these:

```bash
# Single file
xprin test tests/basic_xprin.yaml

# Multiple files
xprin test tests/test1_xprin.yaml tests/test2_xprin.yaml

# Directory (non-recursive) - finds all *_xprin.yaml files in the directory
xprin test tests/

# Recursive - finds all *_xprin.yaml files in directory and subdirectories
xprin test tests/...

# Combination - mix any of the above in a single command
xprin test tests/test1_xprin.yaml tests/... tests/test2_xprin.yaml
```

### Common Command Options

```bash
# Verbose output
xprin test tests/basic_xprin.yaml -v

# Show rendered resources
xprin test tests/basic_xprin.yaml -v --show-render

# Show validation results
xprin test tests/basic_xprin.yaml -v --show-validate

# Show hooks execution
xprin test tests/advanced_xprin.yaml -v --show-hooks

# Show assertion results
xprin test tests/advanced_xprin.yaml -v --show-assertions

# Debug mode (shows detailed execution information)
xprin test tests/basic_xprin.yaml --debug
```

### Configuration Management

```bash
# Show current configuration
xprin config

# Check configuration and dependencies
xprin check

# Or use the config command (equivalent)
xprin config --check

# Use custom config file
xprin -c /path/to/config.yaml test tests/
```

---

## Testsuite examples

### Simple Test Suite

A simple testsuite consists of an array of test cases, each one with its own inputs.

```yaml
# tests/simple_xprin.yaml
tests:
- name: "Database Setup"
  inputs:
    xr: database-xr.yaml
    composition: database-composition.yaml
    functions: /path/to/functions
    crds:
    - /path/to/crds
- name: "Web Application"
  inputs:
    claim: webapp-claim.yaml
    composition: webapp-composition.yaml
    functions: /path/to/functions
```

It supports both XR and Claim as inputs. The Claim will be converted to XR using the `xprin-helpers convert-claim-to-xr` tool.

In this example:
- the "Database Setup" test will run both `render` and `validate`
- the "Web Application" test will run only `render`. The `validate` command will be skipped, because no CRDs are defined.

### Common Inputs

It might be possible that the test cases in a testsuite file might have common inputs. To avoid duplication, we can define them in the `common` block:

```yaml
# tests/common_inputs_xprin.yaml
common:
  inputs:
    functions: /path/to/functions
    crds:
    - /path/to/crds

tests:
- name: "Database Setup"
  inputs:
    xr: database-xr.yaml
    composition: database-composition.yaml
- name: "Web Application"
  inputs:
    claim: webapp-claim.yaml
    composition: webapp-composition.yaml
```

If an input is defined both in the common and the testcase level, the testcase prevails.

### Patching

We can do simple patching via the `xprin-helpers patch-xr` tool. 

```yaml
# tests/patch_inputs_xprin.yaml
common:
  inputs:
    composition: database-composition.yaml
    functions: /path/to/functions
    crds:
    - /path/to/crds

tests:
- name: "EU Database Setup"
  patches:
    xrd: /path/to/crds/xrd.yaml
  inputs:
    xr: eu-database-xr.yaml
- name: "US Database Setup"
  patches:
    connection-secret: true
  inputs:
    claim: us-database-claim.yaml
```

Similarly to inputs, the patches can be defined either in the common or on the testcase level. In case they are defined in both, the testcase level prevails.

### Hooks

Hooks can run arbitrary shell commands before and after each test.

#### Pre-test hooks
Use pre-test hooks for:
- test environment setup (creating temp dirs, copying fixtures, exporting env)
- pre-test validations (e.g. ensure `.spec.k8s_version` matches `1.3[0-9]`)
- advanced patching of any input (see [input template variables](testsuite-specification.md#template-variables))

Example:
```yaml
# tests/pre_post_hooks_xprin.yaml
common:
  hooks:
    pre-test:
    - name: "setup environment"
      run: ./scripts/setup_test_env.sh
    - name: "validate k8s version"
      run: ./scripts/validate_k8s_version.sh {{ .Inputs.XR }}
    - name: "patch XR size to small if unset"
      run: ./scripts/patch_xr_size.sh {{ .Inputs.XR }}
```

For more complex operations, you can use inline commands. Here's an example with a one-liner:
```yaml
hooks:
  pre-test:
  - name: "validate and patch in one step"
    run: v=$(yq -r '.spec.k8s_version' {{ .Inputs.XR }}) && echo "$v" | grep -Eq '^1\.3[0-9]$' || { echo "Invalid version: $v"; exit 1; } && yq -i '.spec.parameters.size //= "small"' {{ .Inputs.XR }}
```

Notes:
- `{{ .Inputs.XR }}` and other input variables are available in pre-test hooks.
- See [Template Variables](testsuite-specification.md#template-variables) for the full list.
- Scripts can be stored in your repository and referenced by path, making hooks cleaner and reusable.

#### Post-test hooks
Use post-test hooks for:
- cleaning up the test environment
- printing outputs (see [output template variables](testsuite-specification.md#template-variables))
- comparing inputs and outputs (e.g. `diff -u {{ .Inputs.XR }} {{ .Outputs.XR }}`)
- assertions via scripts or external tools (e.g. Kyverno Chainsaw, UpTest) using the render output

Example:
```yaml
tests:
- name: "XR sanity"
  inputs:
    xr: xr.yaml
    composition: comp.yaml
  hooks:
    post-test:
    # 1) Print paths to outputs
    - name: "print outputs"
      run: ./scripts/print_outputs.sh {{ .Outputs.XR }} {{ .Outputs.Render }}

    # 2) Compare original XR vs rendered XR
    - name: "diff inputs vs outputs"
      run: ./scripts/diff_xr.sh {{ .Inputs.XR }} {{ .Outputs.XR }}

    # 3) Cleanup
    - name: "cleanup"
      run: ./scripts/cleanup_test_env.sh
```

### Assertions

Assertions provide declarative validation of rendered resources without writing custom scripts. They are grouped by **engine**:
- **xprin:** In-process assertions: count, existence, field type/value checks on rendered resources.
- **diff:** Golden-file comparison using a **unified diff**
- **dyff:** Golden-file comparison using **dyff** (structural YAML diff, document-aware).

**Quick Example:**
```yaml
tests:
- name: "AWS Infra"
  inputs:
    xr: xr.yaml
    composition: composition.yaml
    functions: /path/to/functions
  assertions:
    xprin:
    - name: "Number of resources"
      type: "Count"
      value: 3
    - name: "SecurityGroup should exist"
      type: "Exists"
      resource: "SecurityGroup/platform-aws-sg"
    - name: "RDS backupRetentionPeriod should equal 1"
      type: "FieldValue"
      resource: "Cluster/platform-aws-rds"
      field: "spec.forProvider.backupRetentionPeriod"
      operator: "=="
      value: 1
    diff:
    - name: "Full render matches golden (using diff)"
      expected: golden_full_render.yaml
    - name: "Cluster resource matches golden (using diff)"
      expected: golden_single_resource.yaml
      resource: "Cluster/platform-aws-rds"
    dyff:
    - name: "Full render matches golden (using dyff)"
      expected: golden_full_render.yaml
```

For complete documentation on all assertion types, examples, and usage, see [Assertions](assertions.md).

**When to use assertions vs hooks:**
- **Use assertions** for declarative validation: xprin (count, existence, field checks), diff/dyff (golden-file comparison).
- **Use hooks** for complex operations, external tool integration, or custom validation logic.

### Test Chaining

Tests can be chained together by assigning an `id` to a test case. Tests with IDs have their outputs stored in a shared artifacts directory and can be referenced by later tests using the `.Tests.{test-id}` template variable.

This is useful for:
- **Sequential workflows**: Using outputs from one test as inputs for the next
- **Incremental validation**: Building up complexity across multiple test cases
- **Cross-test comparisons**: Comparing outputs between different test scenarios

Example:
```yaml
tests:
- name: "Initial Database Setup"
  id: "db-setup"
  inputs:
    xr: database-xr.yaml
    composition: database-composition.yaml
    functions: /path/to/functions
  hooks:
    post-test:
    - name: "save database outputs"
      run: ./scripts/save_test_outputs.sh {{ .Outputs.XR }} {{ .Outputs.RenderCount }}

- name: "Application Deployment"
  inputs:
    xr: app-xr.yaml
    composition: app-composition.yaml
    functions: /path/to/functions
  hooks:
    pre-test:
    - name: "verify database setup"
      run: ./scripts/verify_database_setup.sh {{ .Tests.db-setup.Outputs.Render }} {{ .Tests.db-setup.Outputs.XR }}

    post-test:
    - name: "compare outputs"
      run: ./scripts/compare_test_outputs.sh {{ .Tests.db-setup.Outputs.RenderCount }} {{ .Outputs.RenderCount }} {{ .Tests.db-setup.Outputs.Render }}
```

For detailed information about how test chaining works, see [How It Works](how-it-works.md#test-chaining-and-artifacts).

---

## Output Examples

### Successful Test Run

```bash
$ xprin test tests/basic_xprin.yaml
ok      tests/basic_xprin.yaml     4.896s
```

### Verbose Output

```bash
$ xprin test tests/basic_xprin.yaml -v
=== RUN   Database_Setup
--- PASS: Database_Setup (2.33s)
=== RUN   Web_Application
--- PASS: Web_Application (2.21s)
PASS
ok      tests/basic_xprin.yaml     4.546s
```

### Failed Test Run

Failures are shown with section headers (**Render:**, **Validate:**, **Assertions:**, **Pre-test Hooks:**, **Post-test Hooks:**) and symbols **[✓]** pass, **[x]** fail, **[!]** error. Example:

```bash
$ xprin test tests/broken_xprin.yaml
--- FAIL: gcp (0.00s)
    [!] failed to expand or verify paths:
    [!] composition file not found: stat /path/to/comp.yaml: no such file or directory
--- FAIL: gcp_claim_aws_composition (0.22s)
    Render:
        [!] crossplane: error: composition's compositeTypeRef.kind (XAWSAccount) does not match XR's kind (XGCPAccount)
--- FAIL: EKS_clustername (2.15s)
    Validate:
        [!] could not find CRD/XRD for: kubernetes.crossplane.io/v1alpha2, Kind=Object
        Total 2 resources: 1 missing schemas, 1 success cases, 0 failure cases
        crossplane: error: cannot validate resources: could not validate all resources, schema(s) missing
FAIL
FAIL    tests/broken_xprin.yaml  4.863s
FAIL
```

### Show Rendered Resources

```bash
$ xprin test -v --show-render tests/hello_xprin.yaml
=== RUN   onecluster
    Render:
        ├── XCluster/mycluster-72nd5
        └── Object/mycluster
--- PASS: onecluster (2.15s)
PASS
ok      tests/hello_xprin.yaml     4.422s
```

### Show Validation Results

```bash
$ xprin test -v --show-validate tests/hello_xprin.yaml
=== RUN   onecluster
    Validate:
        [✓] mycluster.myorg.com/v1alpha1, Kind=XCluster, mycluster-qngkb validated successfully
        [✓] kubernetes.crossplane.io/v1alpha2, Kind=Object, mycluster validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
--- PASS: onecluster (3.03s)
PASS
ok      tests/hello_xprin.yaml     5.193s
```

## Integration Examples

### With CI/CD Pipeline

```yaml
# .github/workflows/test.yml
name: Test Compositions
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      - name: Install xprin
        run: |
          go install github.com/crossplane-contrib/xprin/cmd/xprin@latest
          go install github.com/crossplane-contrib/xprin/cmd/xprin-helpers@latest
      - name: Check xprin dependencies
        run: xprin check
      - name: Run tests
        run: xprin test tests/
```

---

**Next Steps:**
- Learn by doing with step-by-step [Examples](../examples/README.md) that build complexity progressively
- Or jump to the [Test Suite Specification](testsuite-specification.md) for a complete reference of all available options
