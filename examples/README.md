# xprin Examples

This directory contains step-by-step examples to help you learn how to use `xprin` incrementally. Each example builds on the previous one, introducing new features gradually.

> **📖 Reference**: For command syntax, testsuite structure, and template variables, see the [Getting Started Documentation](../docs/getting-started.md) and [Documentation Index](../docs/README.md).

## Table of Contents

- [Basic Examples](#basic-examples)
- [Multiple Test Cases](#multiple-test-cases)
- [Patching XRs](#patching-xrs)
- [Hooks](#hooks)
- [Chained Tests / Artifacts](#chained-tests--artifacts)
- [Assertions](#assertions)

See [How to Run Tests](../docs/getting-started.md#how-to-run-tests) in the getting started documentation for details on running tests. The basic pattern is:

```bash
xprin test examples/mytests/...
```

---

## Basic Examples

Basic examples for getting started with `xprin`. These examples demonstrate fundamental concepts with single test cases.

### Example 1: Simple Test using XR

**File**: [`mytests/1_simple_tests/example1_using-xr_xprin.yaml`](mytests/1_simple_tests/example1_using-xr_xprin.yaml)

This example demonstrates the most basic test case: rendering a Composition with an XR (Composite Resource) input. The output is identical to `go test`:

```bash
xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
```

**Outputs:**

<details>
<summary>Non-verbose output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	2.150s
```

</details>

<details>
<summary>Verbose output (-v/--verbose)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.75s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.757s
```

</details>

<details>
<summary>Debug output (--debug)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
DEBUG: Created testsuite artifacts directory: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testsuite-artifacts-864435356
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using XR)'
DEBUG: Created temporary directory for test case: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956
DEBUG: - Inputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/inputs
DEBUG: - Outputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/outputs
DEBUG: Test specification:
DEBUG:   Inputs:
DEBUG:   - XR: ../../aws/xr.yaml
DEBUG:   - Composition: ../../aws/composition.yaml
DEBUG:   - Functions: ../../aws/functions.yaml
DEBUG: Test specification with expanded input paths:
DEBUG:   Inputs:
DEBUG:   - XR: /Users/myuser/repos/xprin/examples/aws/xr.yaml
DEBUG:   - Composition: /Users/myuser/repos/xprin/examples/aws/composition.yaml
DEBUG:   - Functions: /Users/myuser/repos/xprin/examples/aws/functions.yaml
DEBUG: Copied xr to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/inputs/xr/xr.yaml
DEBUG: Copied composition to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/inputs/composition/composition.yaml
DEBUG: Copied functions to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/inputs/functions/functions.yaml
DEBUG: Using provided XR file: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/inputs/xr/xr.yaml
DEBUG: Running render command: crossplane render --include-full-xr ...
DEBUG: Wrote rendered output to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3575798956/outputs/rendered.yaml
DEBUG: Skipped validate command "crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using XR)' completed with status: PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.736s
```

</details>

<details>
<summary>Verbose and Debug output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
DEBUG: Created testsuite artifacts directory: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testsuite-artifacts-1203146703
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using XR)'
DEBUG: Created temporary directory for test case: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052
DEBUG: - Inputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/inputs
DEBUG: - Outputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/outputs
DEBUG: Test specification:
DEBUG:   Inputs:
DEBUG:   - XR: ../../aws/xr.yaml
DEBUG:   - Composition: ../../aws/composition.yaml
DEBUG:   - Functions: ../../aws/functions.yaml
DEBUG: Test specification with expanded input paths:
DEBUG:   Inputs:
DEBUG:   - XR: /Users/myuser/repos/xprin/examples/aws/xr.yaml
DEBUG:   - Composition: /Users/myuser/repos/xprin/examples/aws/composition.yaml
DEBUG:   - Functions: /Users/myuser/repos/xprin/examples/aws/functions.yaml
DEBUG: Copied xr to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/inputs/xr/xr.yaml
DEBUG: Copied composition to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/inputs/composition/composition.yaml
DEBUG: Copied functions to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/inputs/functions/functions.yaml
DEBUG: Using provided XR file: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/inputs/xr/xr.yaml
DEBUG: Running render command: crossplane render --include-full-xr ...
DEBUG: Wrote rendered output to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-3495228052/outputs/rendered.yaml
DEBUG: Skipped validate command "crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using XR)' completed with status: PASS
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.90s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.906s
```

</details>

<details>
<summary>Verbose with render output (-v --show-render)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --show-render
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.76s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.769s
```

</details>

<details>
<summary>Verbose with validate output (-v --show-validate)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --show-validate
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.98s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.985s
```

</details>

Note: No validation results are shown because, as mentioned in the debug output, validation was skipped (no CRDs were specified).

### Example 2: Simple Test using Claim

**File**: [`mytests/1_simple_tests/example2_using-claim_xprin.yaml`](mytests/1_simple_tests/example2_using-claim_xprin.yaml)

This example shows how to use a Claim instead of an XR. `xprin` automatically converts the Claim to an XR before rendering, which is mentioned in the debug output.

```bash
xprin test examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml
```

**Outputs:**

<details>
<summary>Debug output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml
DEBUG: Created testsuite artifacts directory: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testsuite-artifacts-3784384366
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using Claim)'
DEBUG: Created temporary directory for test case: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591
DEBUG: - Inputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/inputs
DEBUG: - Outputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/outputs
DEBUG: Test specification:
DEBUG:   Inputs:
DEBUG:   - Claim: ../../aws/claim.yaml
DEBUG:   - Composition: ../../aws/composition.yaml
DEBUG:   - Functions: ../../aws/functions.yaml
DEBUG: Test specification with expanded input paths:
DEBUG:   Inputs:
DEBUG:   - Claim: /Users/myuser/repos/xprin/examples/aws/claim.yaml
DEBUG:   - Composition: /Users/myuser/repos/xprin/examples/aws/composition.yaml
DEBUG:   - Functions: /Users/myuser/repos/xprin/examples/aws/functions.yaml
DEBUG: Copied claim to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/inputs/claim/claim.yaml
DEBUG: Copied composition to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/inputs/composition/composition.yaml
DEBUG: Copied functions to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/inputs/functions/functions.yaml
DEBUG: Converting Claim to XR
DEBUG: Wrote converted XR to temporary file: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/inputs/xr.yaml
DEBUG: Running render command: crossplane render ...
DEBUG: Wrote rendered output to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-2561622591/outputs/rendered.yaml
DEBUG: Skipped validate command "crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using Claim)' completed with status: PASS
ok      examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml 0.904s
```

</details>

<details>
<summary>Verbose output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop (using Claim)
--- PASS: Initial reconciliation loop (using Claim) (0.76s)
    Render:
        ├── XAWSInfrastructure/platform-aws-wg9vx
        └── SecurityGroup/platform-aws-wg9vx-sg
PASS
ok	examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml	0.768s
```

</details>

### Example 3: Test with schema validation

**File**: [`mytests/1_simple_tests/example3_validate_xprin.yaml`](mytests/1_simple_tests/example3_validate_xprin.yaml)

This example adds CRD validation to ensure the rendered manifests are valid according to their schemas.

```bash
xprin test examples/mytests/1_simple_tests/example3_validate_xprin.yaml
```

**Outputs:**

<details>
<summary>Debug output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example3_validate_xprin.yaml --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example3_validate_xprin.yaml
DEBUG: Created testsuite artifacts directory: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testsuite-artifacts-825609063
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (runs both render and validate)'
DEBUG: Created temporary directory for test case: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768
DEBUG: - Inputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs
DEBUG: - Outputs: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/outputs
DEBUG: Test specification:
DEBUG:   Inputs:
DEBUG:   - XR: ../../aws/xr.yaml
DEBUG:   - Composition: ../../aws/composition.yaml
DEBUG:   - Functions: ../../aws/functions.yaml
DEBUG:   - CRDs:
DEBUG:     - ../../aws/xrd.yaml
DEBUG:     - ../../aws/crossplane.yaml
DEBUG: Test specification with expanded input paths:
DEBUG:   Inputs:
DEBUG:   - XR: /Users/myuser/repos/xprin/examples/aws/xr.yaml
DEBUG:   - Composition: /Users/myuser/repos/xprin/examples/aws/composition.yaml
DEBUG:   - Functions: /Users/myuser/repos/xprin/examples/aws/functions.yaml
DEBUG:   - CRDs:
DEBUG:     - /Users/myuser/repos/xprin/examples/aws/xrd.yaml
DEBUG:     - /Users/myuser/repos/xprin/examples/aws/crossplane.yaml
DEBUG: Copied xr to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/xr/xr.yaml
DEBUG: Copied composition to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/composition/composition.yaml
DEBUG: Copied functions to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/functions/functions.yaml
DEBUG: Copied crds to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/crds/xrd.yaml
DEBUG: Copied crds to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/crds/crossplane.yaml
DEBUG: Using provided XR file: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/inputs/xr/xr.yaml
DEBUG: Running render command: crossplane render --include-full-xr ...
DEBUG: Wrote rendered output to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/outputs/rendered.yaml
DEBUG: Running validate command: crossplane resource validate --error-on-missing-schemas ...
DEBUG: Wrote validation output to: /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-371846768/outputs/validate.txt
DEBUG: Test case 'Initial reconciliation loop (runs both render and validate)' completed with status: PASS
ok	examples/mytests/1_simple_tests/example3_validate_xprin.yaml	1.572s
```

</details>

<details>
<summary>Verbose output</summary>

```
=== RUN   Initial reconciliation loop (runs both render and validate)
--- PASS: Initial reconciliation loop (runs both render and validate) (1.33s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] demo.aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
PASS
ok	examples/mytests/1_simple_tests/example3_validate_xprin.yaml	1.338s
```

</details>

---

## Multiple Test Cases

Examples demonstrating how to test multiple reconciliation loops and use common inputs across test cases.

### Example 1: Multiple Test Cases, emulating multiple Reconciliation Loops

**File**: [`mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml`](mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml)

This example demonstrates how to test multiple reconciliation loops. The second test case includes observed resources, which better emulates the reconciliation process. The Composition creates an extra resource due to that:

```bash
xprin test examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.32s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] demo.aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.35s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] demo.aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
PASS
ok	examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml	2.576s
```

</details>

### Example 2: Multiple Test Cases using Common Inputs

**File**: [`mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml`](mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml)

This example demonstrates the same multiple reconciliation loops pattern as Example 1, but uses the `common` section to avoid repeating shared inputs across all test cases. When multiple tests share common inputs (like composition, functions, and CRDs), you can define them once in the `common.inputs` section.

This functionality is originally provided by the `crossplane render --xrd` command which is available in Crossplane v2, but `xprin` is using `xprin-helpers patchxr --xrd` in the background, which is the same functionality but available to any Crossplane version.

```bash
xprin test examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.29s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] demo.aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.43s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] demo.aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
PASS
ok	examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml	2.674s
```

Note: This produces identical output to Example 1, demonstrating that `common.inputs` is purely a convenience feature for reducing repetition. Note that if an input is defined in both `common.inputs` and at the test case level, the test case value takes precedence.

</details>

---

## Patching XRs

Examples demonstrating how to use patches to apply XRD defaults to XRs before validation.

### Example 1: Patch XR with XRD Defaults

**File**: [`mytests/3_patch_xr/example1_patch-xr-xrd-defaults_xprin.yaml`](mytests/3_patch_xr/example1_patch-xr-xrd-defaults_xprin.yaml)

This example demonstrates how to use `patches.xrd` to apply default values from an XRD to an XR. When an XR/Claim doesn't have a required field specified, schema validation will fail. However, if `patches.xrd` is defined, it populates the field with the default value from the XRD before validation.

```bash
➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-xrd-defaults_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-xrd-defaults_xprin.yaml -v --show-render --show-validate
=== RUN   Validation fails without XRD defaults
--- FAIL: Validation fails without XRD defaults (1.57s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [x] schema validation error aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws : spec.team: Required value
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 1 success cases, 1 failure cases
        crossplane: error: cannot validate resources: could not validate all resources
=== RUN   Validation passes with XRD defaults applied
--- PASS: Validation passes with XRD defaults applied (1.32s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
FAIL
FAIL	examples/mytests/3_patch_xr/example1_patch-xr-xrd-defaults_xprin.yaml	2.894s
```

</details>

Note: Patches can also be specified in the `common.patches` section. If patches are defined at both the common and test case level, the test case level patches take precedence.

---

## Hooks

Examples demonstrating how to use pre-test and post-test hooks to run shell commands before and after test execution.

### Example 1: Pre-test and Post-test Hooks

**File**: [`mytests/4_hooks/example1_hooks_xprin.yaml`](mytests/4_hooks/example1_hooks_xprin.yaml)

This example demonstrates how to use pre-test and post-test hooks. Hooks allow you to run shell commands before and after test execution. Hooks can have optional names for better output readability, you can mix named and unnamed hooks, and you can use multiline commands with the `|` YAML syntax.

```bash
xprin test examples/mytests/4_hooks/example1_hooks_xprin.yaml -v --show-hooks
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/4_hooks/example1_hooks_xprin.yaml -v --show-hooks
=== RUN   Test with pre-test and post-test hooks
--- PASS: Test with pre-test and post-test hooks (1.29s)
    Pre-test Hooks:
        [✓] Prepare test environment
            Setting up test environment...
        [✓] Multiline hook
            first line
            second line
        [✓] echo "Pre-test hook without name"
            Pre-test hook without name
    Post-test Hooks:
        [✓] echo "Post-test hook without name"
            Post-test hook without name
        [✓] Cleanup
            Cleaning up test environment...
PASS
ok	examples/mytests/4_hooks/example1_hooks_xprin.yaml	1.289s
```

</details>

### Example 2: Pre-test Hooks

**File**: [`mytests/4_hooks/example2_pre-test-hooks_xprin.yaml`](mytests/4_hooks/example2_pre-test-hooks_xprin.yaml)

This example demonstrates how to use template variables in pre-test hooks. Template variables allow you to reference inputs dynamically in hook commands. All inputs are available as `{{ .Inputs.XR }}`, `{{ .Inputs.Composition }}`, `{{ .Inputs.Functions }}`, etc. in pre-test hooks. These are the copied destinations though to a temporary dir/file, so that we can manipulate them in pre-test hooks.

In this example, we demonstrate pre-test hooks with template variables. The hooks show how to access input file paths using `{{ .Inputs.XR }}` and `{{ .Inputs.Composition }}`, and that they resolve in the temporary copied files.

Afterwards, we use another pre-test hook to modify the XR's `status.rds` field using `yq` and the `{{ .Inputs.XR }}` template variable. When the composition renders, it checks for `status.rds` and conditionally creates an EC2 Instance resource. The first test shows the default behavior (2 resources), while the second test shows the modified behavior (3 resources) after the hook runs. Also, we see that in this test the original XR file is not changed, as we manipulate only the temporary copy.

```bash
xprin test examples/mytests/4_hooks/example2_pre-test-hooks_xprin.yaml -v --show-render --show-validate --show-hooks
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/4_hooks/example2_pre-test-hooks_xprin.yaml -v --show-render --show-validate --show-hooks
=== RUN   Without modifying XR status
--- PASS: Without modifying XR status (1.34s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   With XR status modified in pre-test hook
--- PASS: With XR status modified in pre-test hook (1.38s)
    Pre-test Hooks:
        [✓] Show XR path
            /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-.../inputs/xr/xr.yaml
        [✓] Show Composition path
            /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-.../inputs/composition/composition.yaml
        [✓] Set RDS status in XR
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Instance/platform-aws-ec2
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=Instance, platform-aws-ec2 validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
PASS
ok	examples/mytests/4_hooks/example2_pre-test-hooks_xprin.yaml	2.774s
```

</details>

### Example 3: Post-test Hooks

**File**: [`mytests/4_hooks/example3_post-test-hooks_xprin.yaml`](mytests/4_hooks/example3_post-test-hooks_xprin.yaml)

This example demonstrates how to use template variables in post-test hooks. Post-test hooks have access to both Inputs and Outputs template variables, allowing you to inspect and compare the test results after rendering and validation.

In this example, we use post-test hooks to:
- Inspect the XR status from the rendered output using `{{ .Outputs.XR }}`
- Display render count using `{{ .Outputs.RenderCount }}`
- Show the render output file path using `{{ .Outputs.Render }}`
- Show the assertions output path using `{{ .Outputs.Assertions }}`
- Access specific rendered resources using `{{ index .Outputs.Rendered "SecurityGroup/platform-aws-sg" }}`
- Compare input and output XRs using `dyff` with both `{{ .Inputs.XR }}` and `{{ .Outputs.XR }}`

```bash
xprin test examples/mytests/4_hooks/example3_post-test-hooks_xprin.yaml -v --show-hooks
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/4_hooks/example3_post-test-hooks_xprin.yaml -v --show-hooks
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.46s)
    Post-test Hooks:
        [✓] Inspect XR status
            conditions:
              - lastTransitionTime: "2024-01-01T00:00:00Z"
                message: 'Unready resources: sg'
                reason: Creating
                status: "False"
                type: Ready
        [✓] Show render count
            2
        [✓] Show render output path
            /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-204705582/outputs/rendered.yaml
        [✓] Show specific rendered resource
            /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-204705582/outputs/rendered-securitygroup-platform-aws-sg.yaml
        [✓] Compare input and output XRs
                 _        __  __
               _| |_   _ / _|/ _|  between /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-204705582/inputs/xr/xr.yaml
             / _' | | | | |_| |_       and /var/folders/st/_skftlwn3bb8z_vk249n6qy80000gn/T/xprin-testcase-204705582/outputs/xr.yaml
            | (_| | |_| |  _|  _|
             \__,_|\__, |_| |_|   returned one difference
                    |___/
            
            (root level)
            + one map entry added:
              status:
                conditions:
                - type: Ready
                  lastTransitionTime: "2024-01-01T00:00:00Z"
                  message: "Unready resources: sg"
                  reason: Creating
                  status: False
            
PASS
ok	examples/mytests/4_hooks/example3_post-test-hooks_xprin.yaml	1.464s
```

</details>

---

## Chained Tests / Artifacts

Test cases can depend on each other. A test case lists the IDs of the test cases it depends on in `needs`: it runs after them, and only if they passed (if one failed, or was skipped, it is skipped, and the reason is printed under it).

When a test case has an `id` field, its outputs are also stored and become available to the test cases that list it in `needs`, via the `.Tests` template variable. This allows you to chain tests together, using outputs from one test as inputs to another.

The examples build up: first a dependency alone, then what happens when a test case fails, then using the outputs, then chaining compositions.

### Example 1: Dependencies with `needs`

**File**: [`mytests/5_chained_tests/example1_needs_xprin.yaml`](mytests/5_chained_tests/example1_needs_xprin.yaml)

The simplest use of `needs`: a test case depends on another one, without using any of its outputs. This is useful when test cases share state, or when it only makes sense to run one once the other works. Here the second test case needs the first, and the third needs the second.

**Key Points:**
- A test case that is needed has an `id` (`aws_first`, `aws_second`)
- A test case lists the ones it depends on in `needs`: it runs after them, and only if they passed
- If one of them fails, the test case is not run: it is shown as `SKIP`, with the reason printed under it (see [Dependencies Between Test Cases](../docs/how-it-works.md#dependencies-between-test-cases-needs) and the next example)
- `needs` can list any test case of the file, also one listed **after** it. In this file the "Third" test case is listed first, but it needs `aws_second`, so it runs last: test cases run in the order of their dependencies, and the ones that don't depend on each other keep the order of the file. Look at the order of the output, and use `xprin graph` to see the dependencies without running anything
- Test cases that need each other (a cycle), or themselves, make the file invalid

```bash
xprin test examples/mytests/5_chained_tests/example1_needs_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```bash
➜ xprin test examples/mytests/5_chained_tests/example1_needs_xprin.yaml -v --show-render --show-validate

=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.33s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.34s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
=== RUN   Third reconciliation loop
--- PASS: Third reconciliation loop (1.35s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
PASS
ok	examples/mytests/5_chained_tests/example1_needs_xprin.yaml	4.021s
```

</details>

---

### Example 2: When a Test Case Fails, the Ones that Need It Are Skipped

**File**: [`mytests/5_chained_tests/example2_needs-failure_xprin.yaml`](mytests/5_chained_tests/example2_needs-failure_xprin.yaml)

This is what to expect when a test case that is higher in the dependency chain fails. The first test case has an assertion that is wrong on purpose, so it fails. The second one needs it, and the third one needs the second one.

**Key Points:**
- A test case that needs one that **failed** or was **skipped** is **not run**: it is shown as `SKIP`, and the reason is printed under it, even without `-v`, so you can see at a glance why it didn't run
- The skip propagates: a test case that needs one that was skipped is skipped too
- The reasons list one needed test case per line, and say if it failed or was skipped
- Only the test case that really failed is reported as `FAIL`: the skipped ones don't add noise, or misleading errors about missing outputs. Test cases that don't need the failed one run as usual
- The run fails (exit code 1) because of the failed test case

```bash
xprin test examples/mytests/5_chained_tests/example2_needs-failure_xprin.yaml -v --show-render --show-validate --show-assertions
```

<details>
<summary><strong>Output</strong></summary>

```bash
➜ xprin test examples/mytests/5_chained_tests/example2_needs-failure_xprin.yaml -v --show-render --show-validate --show-assertions

=== RUN   Initial reconciliation loop
--- FAIL: Initial reconciliation loop (1.33s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
    Assertions:
        [x] Number of resources (wrong on purpose, to make the test fail) - expected 0 resources, got 2
        Total: 1 assertions, 0 successful, 1 failed, 0 errors
=== RUN   Second reconciliation loop
--- SKIP: Second reconciliation loop (0.00s)
    [!] needs failed test case "aws_first"
=== RUN   Third reconciliation loop
--- SKIP: Third reconciliation loop (0.00s)
    [!] needs skipped test case "aws_second"
FAIL
FAIL	examples/mytests/5_chained_tests/example2_needs-failure_xprin.yaml	1.35s
```

</details>

---

### Example 3: Chained Test Outputs

**File**: [`mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml`](mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml)

This example demonstrates how to use outputs from a previous test as input to a subsequent test. The first test runs an initial reconciliation loop and stores its result with id `aws_first`. The second test uses `{{ .Tests.aws_first.Outputs.XR }}` to reference the XR output from the first test, enabling you to test multiple reconciliation loops in sequence.

**Key Points:**
- The first test has `id: aws_first`, which makes its outputs available as artifacts to later tests
- The second test uses `{{ .Tests.aws_first.Outputs.XR }}` to reference the XR output from the first test
- The second test lists `aws_first` in `needs`: a test case can only use (`.Tests`) the outputs of the test cases it lists
- The outputs are available in the hooks too (the second test has a `pre-test` hook that checks that the XR of the first one exists)
- This enables testing multiple reconciliation loops in sequence, where each loop uses the output from the previous one

```bash
xprin test examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```bash
➜ xprin test examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml -v --show-render --show-validate

=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.33s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.34s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
PASS
ok	examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml	2.668s
```

</details>

---

### Example 4: Cross-Composition Chaining

**File**: [`mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml`](mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml)

This example demonstrates cross-composition chaining, where one composition renders XRs that become the input to other compositions. The base composition creates both AWS and GCP XRs; the GCP and AWS layer tests each use the corresponding rendered XR as input.

**Key Points:**
- The base composition renders `XAWSInfrastructure` and `XGCPInfrastructure` XRs as part of its output
- The GCP test uses `{{ index .Tests.base_final.Outputs.Rendered "XGCPInfrastructure/platform-base-gcp" }}` to extract the GCP XR from the first test's output
- The AWS test uses `{{ index .Tests.base_final.Outputs.Rendered "XAWSInfrastructure/platform-base-aws" }}` to extract the AWS XR from the first test's output
- Both list `base_final` in `needs`
- This enables testing compositions that depend on outputs from other compositions

<details>
<summary><strong>Output</strong></summary>

```bash
➜ xprin test examples/mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml -v --show-render --show-validate

=== RUN   Base layer final loop
--- PASS: Base layer final loop (0.86s)
    Render:
        ├── XBaseInfrastructure/platform-base
        ├── XAWSInfrastructure/platform-base-aws
        ├── XGCPInfrastructure/platform-base-gcp
        └── Object/platform-base-base-namespace
    Validate:
        [✓] base.example.com/v1, Kind=XBaseInfrastructure, platform-base validated successfully
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-base-aws validated successfully
        [✓] gcp.example.com/v1, Kind=XGCPInfrastructure, platform-base-gcp validated successfully
        [✓] kubernetes.crossplane.io/v1alpha2, Kind=Object, platform-base-base-namespace validated successfully
        Total 4 resources: 0 missing schemas, 4 success cases, 0 failure cases
=== RUN   GCP layer first loop
--- PASS: GCP layer first loop (0.95s)
    Render:
        ├── XGCPInfrastructure/platform-base-gcp
        └── Firewall/platform-base-gcp-firewall
    Validate:
        [✓] gcp.example.com/v1, Kind=XGCPInfrastructure, platform-base-gcp validated successfully
        [✓] compute.gcp.upbound.io/v1beta1, Kind=Firewall, platform-base-gcp-firewall validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   AWS layer first loop
--- PASS: AWS layer first loop (1.28s)
    Render:
        ├── XAWSInfrastructure/platform-base-aws
        └── SecurityGroup/platform-base-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-base-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-base-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
PASS
ok	examples/mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml	3.09s
```

</details>

---

### Example 5: Combining all the above

**File**: [`mytests/5_chained_tests/example5_xprin.yaml`](mytests/5_chained_tests/example5_xprin.yaml)

This example combines all the above cases, I'll let you discover the output :smile: Feel free also to change/break it to see how it works on failures.

To see how its test cases depend on each other (its dependency graph) without running anything, use `xprin graph`:

```bash
xprin graph examples/mytests/5_chained_tests/example5_xprin.yaml
```

```
➜ xprin graph examples/mytests/5_chained_tests/example5_xprin.yaml
examples/mytests/5_chained_tests/example5_xprin.yaml
└── Base layer final loop [base_final]
    ├── GCP layer first loop [gcp_first]
    │   └── GCP layer second loop [gcp_second]
    │       └── GCP layer third loop
    └── AWS layer first loop [aws_first]
        └── AWS layer second loop [aws_second]
            └── AWS layer third loop
```

The base layer comes first. The GCP and AWS chains each need it, but not each other.

---

### What happens when `needs` goes wrong

The files under [`mytests/0_e2e/`](mytests/0_e2e/) and [`mytests/5_chained_tests/`](mytests/5_chained_tests/) also act as the acceptance tests of `needs`:

- [`failures_xprin.yaml`](mytests/0_e2e/failures_xprin.yaml) (the `Needs:` test cases at the end): more cases of skipped test cases (see Example 2). A test case that uses the outputs of another one without listing it fails with an error that says it has to be listed. A typo in the ID and a typo in the output fail too, with the error of the template.
- [`broken_needs_xprin.yaml`](mytests/0_e2e/broken_needs_xprin.yaml): `needs` that make the file invalid: an ID that doesn't exist, test cases that need each other (a cycle), and a test case that needs itself. All the problems are reported together.
- [`legacy_success_xprin.yaml`](mytests/5_chained_tests/legacy_success_xprin.yaml) and [`legacy_failure_xprin.yaml`](mytests/5_chained_tests/legacy_failure_xprin.yaml): the old way (deprecated), without `needs`: the dependency is detected from `.Tests`. It works and xprin prints a warning, but a typo in the ID, or a test case listed below, fail. These files are removed in xprin 0.7.0.

---

## Assertions

Examples demonstrating how to use assertions to declaratively validate rendered resources. Assertions provide a way to validate the structure, content, and count of rendered manifests without writing custom scripts.

### Example 1: Comprehensive Assertions

**File**: [`mytests/6_assertions/example1_assertions_xprin.yaml`](mytests/6_assertions/example1_assertions_xprin.yaml)

This example demonstrates all assertion types available in `xprin`:

- **Count**: Validate the total number of rendered resources
- **Exists**: Check if a specific resource exists (format: `Kind/name`)
- **NotExists**: Verify that a resource doesn't exist (supports both `Kind/name` and `Kind` formats)
- **FieldExists**: Check if a field exists in a resource
- **FieldNotExists**: Verify that a field doesn't exist in a resource
- **FieldType**: Validate the type of a field value (supports: `string`, `number`, `boolean`, `array`, `object`, `null`)
- **FieldValue**: Compare a field's value using operators (`==` or `is`)

Assertions run after validation (if CRDs are provided) or after rendering, and before post-test hooks. All assertions are evaluated even if some fail, and failed assertions are reported in the test output when using `--show-assertions` with `--verbose`.

```bash
xprin test examples/mytests/6_assertions/example1_assertions_xprin.yaml -v --show-assertions
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/6_assertions/example1_assertions_xprin.yaml -v --show-assertions
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.45s)
    Assertions:
        [✓] Number of all rendered resources (including XR) - found 3 resources (as expected)
        [✓] Exactly 1 resource of a certain Kind/Name should exist - found 1 resources (as expected): SecurityGroup/platform-aws-sg
        [✓] Exactly 0 resources of a certain Kind/Name (basically - resource should not exist) - found 0 resources (as expected)
        [✓] At least 1 SecurityGroup should exist - resources found: SecurityGroup/platform-aws-sg
        [✓] RDS cluster named platform-aws-rds should exist - resources found: Cluster/platform-aws-rds
        [✓] EC2 with a certain name should not exist - resource EC2/platform-aws-ec2 not found (as expected)
        [✓] No EC2 instances should exist - resource EC2 not found (as expected)
        [✓] SecurityGroup should have a name - field metadata.name exists on resource SecurityGroup/platform-aws-sg
        [✓] RDS should not have deprecated field - field spec.deprecatedField does not exist (as expected) on resource Cluster/platform-aws-rds
        [✓] SecurityGroup description should be string - field spec.forProvider.description has expected type string on resource SecurityGroup/platform-aws-sg
        [✓] RDS backupRetentionPeriod should be number - field spec.forProvider.backupRetentionPeriod has expected type number on resource Cluster/platform-aws-rds
        [✓] RDS vpcSecurityGroupIds should be array - field spec.forProvider.vpcSecurityGroupIds has expected type array on resource Cluster/platform-aws-rds
        [✓] SecurityGroup metadata labels should be object - field metadata.labels has expected type object on resource SecurityGroup/platform-aws-sg
        [✓] RDS masterPasswordSecretRef should be object - field spec.forProvider.masterPasswordSecretRef has expected type object on resource Cluster/platform-aws-rds
        [✓] RDS storageEncrypted should be boolean - field spec.forProvider.storageEncrypted has expected type boolean on resource Cluster/platform-aws-rds
        [✓] XR optionalNote should be null - field spec.optionalNote has expected type null on resource XAWSInfrastructure/platform-aws
        [✓] RDS should be Aurora PostgreSQL - field spec.forProvider.engine is aurora-postgresql on resource Cluster/platform-aws-rds
        [✓] RDS backupRetentionPeriod should equal 1 - field spec.forProvider.backupRetentionPeriod == 1 on resource Cluster/platform-aws-rds
        Total: 18 assertions, 18 successful, 0 failed, 0 errors
PASS
ok	examples/mytests/6_assertions/example1_assertions_xprin.yaml	1.450s
```

</details>

**Key Points:**
- Assertions provide declarative validation without writing custom scripts
- All assertion types are demonstrated, including all FieldType values (`string`, `number`, `boolean`, `array`, `object`, `null`)
- Assertions complement post-test hooks: use assertions for declarative validation, hooks for complex operations or external tool integration
- Failed assertions are clearly reported in the output, making it easy to identify validation issues

### Example 2: Pattern-matching assertions

**File**: [`mytests/6_assertions/example2_assertions_pattern_xprin.yaml`](mytests/6_assertions/example2_assertions_pattern_xprin.yaml)

This example demonstrates wildcard pattern matching in resource identifiers (e.g. `Instance/*-ec2`, `Cluster/platform-*`, `*/*`). Patterns can be used with `Count`, `Exists`, `NotExists`, `FieldExists`, `FieldNotExists`, `FieldType`, and `FieldValue` assertions to match multiple resources at once or to target resources whose names are not fully predictable.

```bash
xprin test examples/mytests/6_assertions/example2_assertions_pattern_xprin.yaml -v --show-assertions
```

### Example 3: Diff and Dyff (golden-file) assertions

**File**: [`mytests/6_assertions/example3_golden_file_xprin.yaml`](mytests/6_assertions/example3_golden_file_xprin.yaml)

- **Diff** assertions compare the full render (or a single resource file) to a golden file using a byte-for-byte comparison.
- **Dyff** assertions do a structural YAML comparison and produce a human-readable diff on failure.

This example has a single test that **passes** when the render output matches the golden files. It uses `golden_full_render.yaml` (full render) and `golden_single_resource.yaml` (one resource: `Cluster/platform-aws-rds`). Each assertion has `name` and `expected` (path to the golden file, relative to the test suite file). The optional `resource` field (e.g. `Cluster/platform-aws-rds`) compares that resource’s file instead of the full render.

**First-time setup — generate the golden files:**  
The file includes a commented-out preliminary test “Generate golden files”. Uncomment that test, run the suite once, then comment it back out. That run will create `golden_full_render.yaml` and `golden_single_resource.yaml` in the same directory.

**To see failure output:**  
Edit one of the golden files (e.g. change a value) and run the test again; the assertion will fail and show the diff or dyff output.

```bash
xprin test examples/mytests/6_assertions/example3_golden_file_xprin.yaml -v --show-assertions
```
