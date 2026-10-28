# xprin Examples

This directory contains step-by-step examples to help you learn how to use `xprin` incrementally. Each example builds on the previous one, introducing new features gradually.

> **📖 Reference**: For command syntax, testsuite structure, and template variables, see the [Getting Started Documentation](../docs/getting-started.md) and [Documentation Index](../docs/README.md).

## Table of Contents

- [Check Your Setup](#check-your-setup)
- [Basic Examples](#basic-examples)
- [Multiple Test Cases](#multiple-test-cases)
- [Patching XRs](#patching-xrs)
- [Hooks](#hooks)
- [Chained Tests](#chained-tests)
- [Assertions](#assertions)
- [Show All Results](#show-all-results)

See [How to Run Tests](../docs/getting-started.md#how-to-run-tests) in the getting started documentation for details on running tests. The basic pattern is:

```bash
xprin test examples/mytests/...
```

---

## Check Your Setup

`xprin check` verifies the dependencies and the settings, and tells which Crossplane CLI it found. xprin adapts to it: the Crossplane CLI v1 and v2 differ in the commands (for example `render --xrd`, or `beta validate` against `resource validate`) and in what they do, and xprin runs the right ones (see [Crossplane CLI Versions](../docs/how-it-works.md#crossplane-cli-versions)). These examples run with the Crossplane CLI v2 (2.3 or newer), and show the output of v1 where it differs.

```bash
xprin check
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin check
No global configuration file provided. Using detected dependencies and default settings...

Dependencies:
- crossplane: /usr/local/bin/crossplane (from PATH)

Subcommands:
- render: render --include-full-xr
- validate: resource validate --error-on-missing-schemas

INFO: Crossplane CLI detected as v2+, current (>= v2.3.0)
INFO: crossplane version --client: v2.5.0

OK: dependencies and settings verified
```

</details>

<details>
<summary><strong>Output with the Crossplane CLI v1</strong></summary>

<!-- examples-gen: v1 -->
```
➜ xprin check
No global configuration file provided. Using detected dependencies and default settings...

Dependencies:
- crossplane: /usr/local/bin/crossplane (from PATH)

Subcommands:
- render: render --include-full-xr
- validate: beta validate --error-on-missing-schemas

INFO: Crossplane CLI detected as v1
INFO: crossplane version --client: v1.20.13

OK: dependencies and settings verified
```

</details>

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
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.507s
```

</details>

<details>
<summary>Verbose output (-v/--verbose)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.48s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.484s
```

</details>

<details>
<summary>Debug output (--debug)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
DEBUG: Created internal artifacts directory: /tmp/xprin-internal-artifacts-152756207
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using XR)'
DEBUG: Created temporary directory for test case: /tmp/xprin-testcase-389004930
DEBUG: - Inputs: /tmp/xprin-testcase-389004930/inputs
DEBUG: - Outputs: /tmp/xprin-testcase-389004930/outputs
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
DEBUG: Copied xr to: /tmp/xprin-testcase-389004930/inputs/xr/xr.yaml
DEBUG: Copied composition to: /tmp/xprin-testcase-389004930/inputs/composition/composition.yaml
DEBUG: Copied functions to: /tmp/xprin-testcase-389004930/inputs/functions/functions.yaml
DEBUG: Using provided XR file: /tmp/xprin-testcase-389004930/inputs/xr/xr.yaml
DEBUG: Running render command: /usr/local/bin/crossplane render --include-full-xr --crossplane-docker-network=xprin-net-15e6f88b /tmp/xprin-testcase-389004930/inputs/xr/xr.yaml /tmp/xprin-testcase-389004930/inputs/composition/composition.yaml /tmp/xprin-testcase-389004930/inputs/functions/functions.yaml
DEBUG: Wrote rendered output to: /tmp/xprin-testcase-389004930/outputs/rendered.yaml
DEBUG: Skipped validate command "/usr/local/bin/crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using XR)' completed with status: PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.506s
```

</details>

<details>
<summary>Verbose and Debug output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --debug
DEBUG: Processing testsuite file examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml
DEBUG: Created internal artifacts directory: /tmp/xprin-internal-artifacts-376558074
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using XR)'
DEBUG: Created temporary directory for test case: /tmp/xprin-testcase-829220625
DEBUG: - Inputs: /tmp/xprin-testcase-829220625/inputs
DEBUG: - Outputs: /tmp/xprin-testcase-829220625/outputs
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
DEBUG: Copied xr to: /tmp/xprin-testcase-829220625/inputs/xr/xr.yaml
DEBUG: Copied composition to: /tmp/xprin-testcase-829220625/inputs/composition/composition.yaml
DEBUG: Copied functions to: /tmp/xprin-testcase-829220625/inputs/functions/functions.yaml
DEBUG: Using provided XR file: /tmp/xprin-testcase-829220625/inputs/xr/xr.yaml
DEBUG: Running render command: /usr/local/bin/crossplane render --include-full-xr --crossplane-docker-network=xprin-net-20ce3a4c /tmp/xprin-testcase-829220625/inputs/xr/xr.yaml /tmp/xprin-testcase-829220625/inputs/composition/composition.yaml /tmp/xprin-testcase-829220625/inputs/functions/functions.yaml
DEBUG: Wrote rendered output to: /tmp/xprin-testcase-829220625/outputs/rendered.yaml
DEBUG: Skipped validate command "/usr/local/bin/crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using XR)' completed with status: PASS
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.51s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.514s
```

</details>

<details>
<summary>Verbose with render output (-v --show-render)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --show-render
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.49s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.489s
```

</details>

<details>
<summary>Verbose with validate output (-v --show-validate)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml -v --show-validate
=== RUN   Initial reconciliation loop (using XR)
--- PASS: Initial reconciliation loop (using XR) (0.47s)
PASS
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.468s
```

</details>

Note: No validation results are shown because, as mentioned in the debug output, validation was skipped (no CRDs were specified).

Tip: `--show-all` enables all the `--show-*` flags at once (`--show-render`, `--show-validate`, `--show-hooks` and `--show-assertions`), so `-v --show-all` gives the most detailed output.

To inspect a run afterwards, `--artifacts-dir` exports the inputs and the outputs of every test case (the external artifacts) to a directory of your choice, one per test case (see [Artifacts](../docs/how-it-works.md#artifacts)):

<details>
<summary>Export the artifacts (--artifacts-dir)</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml --artifacts-dir /tmp/xprin-debug
ok	examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml	0.498s
artifacts: /tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests/example1_using-xr_xprin

➜ find /tmp/xprin-debug -maxdepth 7
/tmp/xprin-debug
/tmp/xprin-debug/xprin-artifacts-20261023130617
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests/example1_using-xr_xprin
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests/example1_using-xr_xprin/01_Initial_reconciliation_loop_(using_XR)
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests/example1_using-xr_xprin/01_Initial_reconciliation_loop_(using_XR)/inputs
/tmp/xprin-debug/xprin-artifacts-20261023130617/examples/mytests/1_simple_tests/example1_using-xr_xprin/01_Initial_reconciliation_loop_(using_XR)/outputs
```

</details>

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
DEBUG: Created internal artifacts directory: /tmp/xprin-internal-artifacts-908425348
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (using Claim)'
DEBUG: Created temporary directory for test case: /tmp/xprin-testcase-479652883
DEBUG: - Inputs: /tmp/xprin-testcase-479652883/inputs
DEBUG: - Outputs: /tmp/xprin-testcase-479652883/outputs
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
DEBUG: Copied claim to: /tmp/xprin-testcase-479652883/inputs/claim/claim.yaml
DEBUG: Copied composition to: /tmp/xprin-testcase-479652883/inputs/composition/composition.yaml
DEBUG: Copied functions to: /tmp/xprin-testcase-479652883/inputs/functions/functions.yaml
DEBUG: Converting Claim to XR
DEBUG: Wrote converted XR to temporary file: /tmp/xprin-testcase-479652883/inputs/xr.yaml
DEBUG: Running render command: /usr/local/bin/crossplane render --include-full-xr --crossplane-docker-network=xprin-net-8d4da2a0 /tmp/xprin-testcase-479652883/inputs/xr.yaml /tmp/xprin-testcase-479652883/inputs/composition/composition.yaml /tmp/xprin-testcase-479652883/inputs/functions/functions.yaml
DEBUG: Wrote rendered output to: /tmp/xprin-testcase-479652883/outputs/rendered.yaml
DEBUG: Skipped validate command "/usr/local/bin/crossplane resource validate --error-on-missing-schemas" because no CRDs were specified
DEBUG: Test case 'Initial reconciliation loop (using Claim)' completed with status: PASS
ok	examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml	0.555s
```

</details>

<details>
<summary>Verbose output</summary>

```
➜ xprin test examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop (using Claim)
--- PASS: Initial reconciliation loop (using Claim) (0.50s)
    Render:
        ├── XAWSInfrastructure/platform-aws-fxnfl
        └── SecurityGroup/platform-aws-fxnfl-sg
PASS
ok	examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml	0.499s
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
DEBUG: Created internal artifacts directory: /tmp/xprin-internal-artifacts-728102659
DEBUG: Using testsuite file directory for relative path resolution: examples/mytests/1_simple_tests
DEBUG: Found 1 test case
DEBUG: Starting test case 'Initial reconciliation loop (runs both render and validate)'
DEBUG: Created temporary directory for test case: /tmp/xprin-testcase-184759430
DEBUG: - Inputs: /tmp/xprin-testcase-184759430/inputs
DEBUG: - Outputs: /tmp/xprin-testcase-184759430/outputs
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
DEBUG: Copied xr to: /tmp/xprin-testcase-184759430/inputs/xr/xr.yaml
DEBUG: Copied composition to: /tmp/xprin-testcase-184759430/inputs/composition/composition.yaml
DEBUG: Copied functions to: /tmp/xprin-testcase-184759430/inputs/functions/functions.yaml
DEBUG: Copied crds to: /tmp/xprin-testcase-184759430/inputs/crds/xrd.yaml
DEBUG: Copied crds to: /tmp/xprin-testcase-184759430/inputs/crds/crossplane.yaml
DEBUG: Using provided XR file: /tmp/xprin-testcase-184759430/inputs/xr/xr.yaml
DEBUG: Running render command: /usr/local/bin/crossplane render --include-full-xr --crossplane-docker-network=xprin-net-e1af4481 /tmp/xprin-testcase-184759430/inputs/xr/xr.yaml /tmp/xprin-testcase-184759430/inputs/composition/composition.yaml /tmp/xprin-testcase-184759430/inputs/functions/functions.yaml
DEBUG: Wrote rendered output to: /tmp/xprin-testcase-184759430/outputs/rendered.yaml
DEBUG: Running validate command: /usr/local/bin/crossplane resource validate --error-on-missing-schemas /tmp/xprin-testcase-184759430/inputs/crds /tmp/xprin-testcase-184759430/outputs/rendered.yaml
DEBUG: Wrote validation output to: /tmp/xprin-testcase-184759430/outputs/validate.txt
DEBUG: Test case 'Initial reconciliation loop (runs both render and validate)' completed with status: PASS
ok	examples/mytests/1_simple_tests/example3_validate_xprin.yaml	1.312s
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

### Example 4: Test with validation failures

**File**: [`mytests/1_simple_tests/example4_validate-failure_xprin.yaml`](mytests/1_simple_tests/example4_validate-failure_xprin.yaml)

Example 3 passes the validation. This example shows the two ways that it fails, in the same test case. The XR doesn't match its XRD: `spec.sshPublicKey` can only be `qa_ssh_key`, and the XR sets another value, which is a schema validation error. And the CRDs of the provider are not given, so the `SecurityGroup` that the composition renders has no schema: xprin validates with `--error-on-missing-schemas`, so a resource without a schema fails the validation instead of being skipped. The render works, the test case fails, and the `Validate:` section says which resources failed and why:

```bash
xprin test examples/mytests/1_simple_tests/example4_validate-failure_xprin.yaml
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/1_simple_tests/example4_validate-failure_xprin.yaml
--- FAIL: Validation fails: a value that the XRD doesn't allow, and the provider CRDs are missing (0.60s)
    Validate:
        [x] schema validation error aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws : spec.sshPublicKey: Unsupported value: "prod_ssh_key": supported values: "qa_ssh_key"
        [!] could not find CRD/XRD for: ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup
        Total 2 resources: 1 missing schemas, 0 success cases, 1 failure cases
        crossplane: error: could not validate all resources
FAIL
FAIL	examples/mytests/1_simple_tests/example4_validate-failure_xprin.yaml	0.598s
FAIL
2026/10/24 13:32:18 processing completed with errors
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
--- PASS: Initial reconciliation loop (1.15s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.89s)
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
ok	examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml	2.044s
```

</details>

The function containers are reused across the test cases, and the testsuite files, of a run: the first test case that uses a function image pays for pulling it (if it isn't on the machine yet) and starting its container, and the next ones reuse the running container. With `--no-container-reuse`, every render starts and removes a fresh container instead (see [Function Container Reuse](../docs/how-it-works.md#function-container-reuse)). Compare the times of the second test case:

<details>
<summary>Without <code>--no-container-reuse</code> (the default) and with it</summary>

```
➜ xprin test examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml -v
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.21s)
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.90s)
PASS
ok	examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml	2.105s

➜ xprin test examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml -v --no-container-reuse
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.39s)
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (1.42s)
PASS
ok	examples/mytests/2_multiple_testcases/example1_multiple-reconciliation-loops_xprin.yaml	2.813s
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
--- PASS: Initial reconciliation loop (1.14s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.90s)
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
ok	examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml	2.043s
```

Note: This produces identical output to Example 1, demonstrating that `common.inputs` is purely a convenience feature for reducing repetition. Note that if an input is defined in both `common.inputs` and at the test case level, the test case value takes precedence.

</details>

---

## Patching XRs

Examples demonstrating how to use `patches` with an XRD.

### Example 1: Patch XR with the XRD (legacy XRD)

**File**: [`mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml`](mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml)

The XR of this example doesn't set `spec.team`, which its XRD requires and defaults to `devops`, and the XRD is a legacy one (`apiextensions.crossplane.io/v1`, with a Claim). Both test cases render the same XR, and the second one gives the XRD with `patches.xrd`. `crossplane render` needs the XRD to apply the defaults and to know the scope of the XR, and the Crossplane CLI v1 and v2 do that differently, so does xprin:

- **v2 (2.3 or newer)**: xprin passes `patches.xrd` to `crossplane render` as `--xrd`. Without it, render takes the XRD as one of the Modern scope, and puts `resourceRefs` at `spec.crossplane.resourceRefs`, which the legacy XRD doesn't have: the first test case fails the validation (`spec.crossplane` is an unknown field).
- **v1**: `crossplane render` has no `--xrd`. xprin applies the defaults of the XRD to the XR itself before it renders it. Without it, the first test case fails the validation (`spec.team` is required, and the XR doesn't have it).

In both versions the first test case fails and the second one passes, with `--debug` to see the difference:

```bash
xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml -v --show-render --show-validate
xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml --debug 2>&1 | grep -oE '(Crossplane CLI v1 detected.*|Applying XRD defaults|--xrd=[^ ]+)'
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml -v --show-render --show-validate
=== RUN   Validation fails without patches.xrd (legacy XRD)
--- FAIL: Validation fails without patches.xrd (legacy XRD) (1.29s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [x] schema validation error aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws : spec.crossplane: Invalid value: "crossplane": unknown field: "crossplane"
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 1 success cases, 1 failure cases
        crossplane: error: could not validate all resources
=== RUN   Validation passes with patches.xrd (legacy XRD)
--- PASS: Validation passes with patches.xrd (legacy XRD) (1.04s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
FAIL
FAIL	examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml	2.326s
FAIL
2026/10/23 14:05:25 processing completed with errors

➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml --debug 2>&1 | grep -oE '(Crossplane CLI v1 detected.*|Applying XRD defaults|--xrd=[^ ]+)'
--xrd=/tmp/xprin-testcase-783691409/inputs/xrd/xrd_legacy.yaml
```

</details>

<details>
<summary><strong>Output with the Crossplane CLI v1</strong></summary>

<!-- examples-gen: v1 -->
```
➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml -v --show-render --show-validate
=== RUN   Validation fails without patches.xrd (legacy XRD)
--- FAIL: Validation fails without patches.xrd (legacy XRD) (1.92s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [x] schema validation error aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws : spec.team: Required value
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 1 success cases, 1 failure cases
        crossplane: error: cannot validate resources: could not validate all resources
=== RUN   Validation passes with patches.xrd (legacy XRD)
--- PASS: Validation passes with patches.xrd (legacy XRD) (0.69s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
FAIL
FAIL	examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml	2.608s
FAIL
2026/10/23 14:06:04 processing completed with errors

➜ xprin test examples/mytests/3_patch_xr/example1_patch-xr-legacy-xrd_xprin.yaml --debug 2>&1 | grep -oE '(Crossplane CLI v1 detected.*|Applying XRD defaults|--xrd=[^ ]+)'
Crossplane CLI v1 detected; patches.xrd will not be passed as --xrd to render
Applying XRD defaults
```

</details>

Note: Patches can also be specified in the `common.patches` section. If patches are defined at both the common and test case level, the test case level prevails.

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
--- PASS: Test with pre-test and post-test hooks (0.52s)
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
ok	examples/mytests/4_hooks/example1_hooks_xprin.yaml	0.524s
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
--- PASS: Without modifying XR status (1.18s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   With XR status modified in pre-test hook
--- PASS: With XR status modified in pre-test hook (0.88s)
    Pre-test Hooks:
        [✓] Show XR path
            /tmp/xprin-testcase-524360000/inputs/xr/xr.yaml
        [✓] Show Composition path
            /tmp/xprin-testcase-524360000/inputs/composition/composition.yaml
        [✓] Set RDS status in XR
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Instance/platform-aws-primary-ec2
        ├── Instance/platform-aws-secondary-ec2
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=Instance, platform-aws-primary-ec2 validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=Instance, platform-aws-secondary-ec2 validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 4 resources: 0 missing schemas, 4 success cases, 0 failure cases
PASS
ok	examples/mytests/4_hooks/example2_pre-test-hooks_xprin.yaml	2.059s
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
--- FAIL: Initial reconciliation loop (1.45s)
    Post-test Hooks:
        [✓] Inspect XR status
            conditions:
              - lastTransitionTime: "2024-01-01T00:00:00Z"
                reason: WatchCircuitClosed
                status: "True"
                type: Responsive
              - lastTransitionTime: "2024-01-01T00:00:00Z"
                reason: ReconcileSuccess
                status: "True"
                type: Synced
              - lastTransitionTime: "2024-01-01T00:00:00Z"
                message: 'Unready resources: sg'
                reason: Creating
                status: "False"
                type: Ready
        [✓] Show render count
            2
        [✓] Show render output path
            /tmp/xprin-testcase-961589126/outputs/rendered.yaml
        [✓] Show specific rendered resource
            /tmp/xprin-testcase-961589126/outputs/rendered-securitygroup-platform-aws-sg.yaml
        [x] Compare input and output XRs [exit code: 1]
                 _        __  __
               _| |_   _ / _|/ _|  between /tmp/xprin-testcase-961589126/inputs/xr/xr.yaml
             / _' | | | | |_| |_       and /tmp/xprin-testcase-961589126/outputs/xr.yaml
            | (_| | |_| |  _|  _|
             \__,_|\__, |_| |_|   returned two differences
                    |___/
            
            (root level)
            + one map entry added:
              status:
                conditions:
                - type: Responsive
                  lastTransitionTime: "2024-01-01T00:00:00Z"
                  reason: WatchCircuitClosed
                  status: True
                - type: Synced
                  lastTransitionTime: "2024-01-01T00:00:00Z"
                  reason: ReconcileSuccess
                  status: True
                - type: Ready
                  lastTransitionTime: "2024-01-01T00:00:00Z"
                  message: "Unready resources: sg"
                  reason: Creating
                  status: False
            
            spec
              + one map entry added:
                crossplane:
                  resourceRefs:
                  - apiVersion: ec2.aws.upbound.io/v1beta1
                    kind: SecurityGroup
                    name: platform-aws-sg
            
FAIL
FAIL	examples/mytests/4_hooks/example3_post-test-hooks_xprin.yaml	1.446s
FAIL
2026/10/23 20:20:02 processing completed with errors
```

</details>

---

## Chained Tests

Test cases can depend on each other. A test case lists the IDs of the test cases it depends on in `needs`: it runs after them, and only if they passed (if one failed, or was skipped, it is skipped, and the reason is printed under it).

When a test case has an `id` field, its outputs are also stored and become available to the test cases that list it in `needs`, via the `.Tests` template variable. This allows you to chain tests together, using outputs from one test as inputs to another.

**Quote the template in YAML:** a value that *starts* with `{{` must be quoted, e.g. `xr: '{{ .Tests.aws_first.Outputs.XR }}'`. Unquoted, it is not valid YAML (YAML reads it as a flow mapping), so editors and the JSON Schema report the file as broken. Use single quotes, as the templates often contain double quotes. See [Quoting Templates in YAML](../docs/how-it-works.md#quoting-templates-in-yaml).

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

```
➜ xprin test examples/mytests/5_chained_tests/example1_needs_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.18s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.90s)
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
--- PASS: Third reconciliation loop (0.90s)
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
ok	examples/mytests/5_chained_tests/example1_needs_xprin.yaml	2.979s
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

```
➜ xprin test examples/mytests/5_chained_tests/example2_needs-failure_xprin.yaml -v --show-render --show-validate --show-assertions
=== RUN   Initial reconciliation loop
--- FAIL: Initial reconciliation loop (1.20s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
    Assertions:
        [x] Number of resources (wrong on purpose, to make the test fail): expected 0 resources, got 2
        Total: 1 assertions, 0 successful, 1 failed, 0 errors
=== RUN   Second reconciliation loop
--- SKIP: Second reconciliation loop (0.00s)
    [!] needs failed test case "aws_first"
=== RUN   Third reconciliation loop
--- SKIP: Third reconciliation loop (0.00s)
    [!] needs skipped test case "aws_second"
FAIL
FAIL	examples/mytests/5_chained_tests/example2_needs-failure_xprin.yaml	1.201s
FAIL
2026/10/23 13:06:55 processing completed with errors
```

</details>

---

### Example 3: Chained Test Outputs

**File**: [`mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml`](mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml)

This example demonstrates how to use outputs from a previous test as input to a subsequent test. The first test runs an initial reconciliation loop and stores its result with id `aws_first`. The second test uses `{{ .Tests.aws_first.Outputs.XR }}` to reference the XR output from the first test, enabling you to test multiple reconciliation loops in sequence.

**Key Points:**
- The first test has `id: aws_first`, which makes its outputs available to later tests
- The second test uses `{{ .Tests.aws_first.Outputs.XR }}` to reference the XR output from the first test
- The second test lists `aws_first` in `needs`: a test case can only use (`.Tests`) the outputs of the test cases it lists
- The outputs are available in the hooks too (the second test has a `pre-test` hook that checks that the XR of the first one exists)
- This enables testing multiple reconciliation loops in sequence, where each loop uses the output from the previous one

```bash
xprin test examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml -v --show-render --show-validate
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml -v --show-render --show-validate
=== RUN   Initial reconciliation loop
--- PASS: Initial reconciliation loop (1.18s)
    Render:
        ├── XAWSInfrastructure/platform-aws
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.89s)
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
ok	examples/mytests/5_chained_tests/example3_chained-test-outputs_xprin.yaml	2.065s
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

```
➜ xprin test examples/mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml -v --show-render --show-validate
=== RUN   Base layer final loop
--- PASS: Base layer final loop (2.12s)
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
--- PASS: GCP layer first loop (4.23s)
    Render:
        ├── XGCPInfrastructure/platform-base-gcp
        └── Firewall/platform-base-gcp-firewall
    Validate:
        [✓] gcp.example.com/v1, Kind=XGCPInfrastructure, platform-base-gcp validated successfully
        [✓] compute.gcp.upbound.io/v1beta1, Kind=Firewall, platform-base-gcp-firewall validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
=== RUN   AWS layer first loop
--- PASS: AWS layer first loop (0.92s)
    Render:
        ├── XAWSInfrastructure/platform-base-aws
        └── SecurityGroup/platform-base-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-base-aws validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-base-aws-sg validated successfully
        Total 2 resources: 0 missing schemas, 2 success cases, 0 failure cases
PASS
ok	examples/mytests/5_chained_tests/example4_cross-composition-chaining_xprin.yaml	2.577s
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

### Example 6: Mistakes in the use of `needs`

**File**: [`mytests/5_chained_tests/example6_needs-mistakes_xprin.yaml`](mytests/5_chained_tests/example6_needs-mistakes_xprin.yaml)

A test case can only use the outputs of the test cases that it lists in `needs`. Using the outputs of one that it doesn't list fails with an error that says it has to be listed. A typo in the ID, or in the name of the output, fails with the error of the template:

```bash
xprin test examples/mytests/5_chained_tests/example6_needs-mistakes_xprin.yaml
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/5_chained_tests/example6_needs-mistakes_xprin.yaml
--- FAIL: Uses the outputs of a test case without listing it (fails: it has to be listed in needs) (0.00s)
    [!] failed to process template variables: failed to render template: failed to execute template: one or more test cases are used but are not listed in needs
--- FAIL: A typo in the ID at the template variable (0.00s)
    [!] failed to process template variables: failed to render template: failed to execute template: template: testcase:6:14: executing "testcase" at <.Tests.frist.Outputs.XR>: map has no entry for key "frist"
--- FAIL: A typo in the name of the output (0.00s)
    [!] failed to process template variables: failed to render template: failed to execute template: template: testcase:6:14: executing "testcase" at <.Tests.first.Outputs.XRR>: can't evaluate field XRR in type engine.Outputs
FAIL
FAIL	examples/mytests/5_chained_tests/example6_needs-mistakes_xprin.yaml	0.550s
FAIL
2026/10/23 13:38:19 processing completed with errors
```

</details>

### Example 7: `needs` that make the file invalid

**File**: [`mytests/5_chained_tests/example7_needs-invalid_xprin.yaml`](mytests/5_chained_tests/example7_needs-invalid_xprin.yaml)

Some `needs` make the testsuite file invalid: an ID that doesn't exist, test cases that need each other (a cycle), and a test case that needs itself. No test case runs, and all the problems are reported together:

```bash
xprin test examples/mytests/5_chained_tests/example7_needs-invalid_xprin.yaml
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin test examples/mytests/5_chained_tests/example7_needs-invalid_xprin.yaml
# examples/mytests/5_chained_tests/example7_needs-invalid_xprin.yaml
invalid testsuite file:
- test case "Second loop, with a typo in needs" needs "frist", but no test case has that ID
- needs cycle: third -> fourth -> third
- needs cycle: fifth -> fifth
FAIL	examples/mytests/5_chained_tests/example7_needs-invalid_xprin.yaml	[invalid testsuite file]
FAIL
2026/10/23 13:38:19 processing completed with errors
```

</details>

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
--- PASS: Second reconciliation loop (0.49s)
    Assertions:
        [✓] Number of all rendered resources (including XR): found 3 resources (as expected)
        [✓] Exactly 1 resource of a certain Kind/Name should exist: found 1 resources (as expected): SecurityGroup/platform-aws-sg
        [✓] Exactly 0 resources of a certain Kind/Name (basically - resource should not exist): found 0 resources (as expected)
        [✓] At least 1 SecurityGroup should exist: resources found: SecurityGroup/platform-aws-sg
        [✓] RDS cluster named platform-aws-rds should exist: resources found: Cluster/platform-aws-rds
        [✓] EC2 with a certain name should not exist: resource EC2/platform-aws-ec2 not found (as expected)
        [✓] No EC2 instances should exist: resource EC2 not found (as expected)
        [✓] SecurityGroup should have a name: field metadata.name exists on resource SecurityGroup/platform-aws-sg
        [✓] RDS should not have deprecated field: field spec.deprecatedField does not exist (as expected) on resource Cluster/platform-aws-rds
        [✓] SecurityGroup description should be string: field spec.forProvider.description has expected type string on resource SecurityGroup/platform-aws-sg
        [✓] RDS backupRetentionPeriod should be number: field spec.forProvider.backupRetentionPeriod has expected type number on resource Cluster/platform-aws-rds
        [✓] RDS vpcSecurityGroupIds should be array: field spec.forProvider.vpcSecurityGroupIds has expected type array on resource Cluster/platform-aws-rds
        [✓] SecurityGroup metadata labels should be object: field metadata.labels has expected type object on resource SecurityGroup/platform-aws-sg
        [✓] RDS masterPasswordSecretRef should be object: field spec.forProvider.masterPasswordSecretRef has expected type object on resource Cluster/platform-aws-rds
        [✓] RDS storageEncrypted should be boolean: field spec.forProvider.storageEncrypted has expected type boolean on resource Cluster/platform-aws-rds
        [✓] XR optionalNote should be null: field spec.optionalNote has expected type null on resource XAWSInfrastructure/platform-aws
        [✓] RDS should be Aurora PostgreSQL: field spec.forProvider.engine is aurora-postgresql on resource Cluster/platform-aws-rds
        [✓] RDS backupRetentionPeriod should equal 1: field spec.forProvider.backupRetentionPeriod == 1 on resource Cluster/platform-aws-rds
        Total: 18 assertions, 18 successful, 0 failed, 0 errors
PASS
ok	examples/mytests/6_assertions/example1_assertions_xprin.yaml	0.490s
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

### Example 3: Generating golden files with update-goldens

**File**: [`mytests/6_assertions/example3_golden_file_xprin.yaml`](mytests/6_assertions/example3_golden_file_xprin.yaml)

Golden files don't have to be written by hand. `xprin update-goldens` runs the render of each test case that has `assertions.diff` or `assertions.dyff` entries, and writes the result to their `expected:` paths. This is the workflow:

1. Write the `diff` or `dyff` assertions, with the `expected:` paths of the golden files that don't exist yet (here, we delete the two golden files that the next example uses).
2. Run `xprin update-goldens`. It lists the files it wrote under `Golden Files:`.
3. Check the generated file, and commit it: it is the baseline now.
4. Run `xprin test`: the assertions pass against the golden files that were just generated (the next example).

```bash
rm examples/mytests/6_assertions/golden_full_render.yaml examples/mytests/6_assertions/golden_single_resource.yaml
xprin update-goldens examples/mytests/6_assertions/example3_golden_file_xprin.yaml
head -15 examples/mytests/6_assertions/golden_single_resource.yaml
```

<details>
<summary><strong>Output</strong></summary>

```
➜ rm examples/mytests/6_assertions/golden_full_render.yaml examples/mytests/6_assertions/golden_single_resource.yaml

➜ xprin update-goldens examples/mytests/6_assertions/example3_golden_file_xprin.yaml
--- PASS: Successful assertions against golden files (0.52s)
    Golden Files:
        [✓] Full render matches golden (using diff): /Users/myuser/repos/xprin/examples/mytests/6_assertions/golden_full_render.yaml
        [✓] Cluster resource matches golden (using diff): /Users/myuser/repos/xprin/examples/mytests/6_assertions/golden_single_resource.yaml
        [✓] Full render matches golden (using dyff): /Users/myuser/repos/xprin/examples/mytests/6_assertions/golden_full_render.yaml
        [✓] Cluster resource matches golden (using dyff): /Users/myuser/repos/xprin/examples/mytests/6_assertions/golden_single_resource.yaml
ok	examples/mytests/6_assertions/example3_golden_file_xprin.yaml	0.516s

➜ head -15 examples/mytests/6_assertions/golden_single_resource.yaml
apiVersion: rds.aws.upbound.io/v1beta1
kind: Cluster
metadata:
  annotations:
    crossplane.io/composition-resource-name: rds
  labels:
    crossplane.io/composite: platform-aws
  name: platform-aws-rds
  ownerReferences:
  - apiVersion: aws.example.com/v1
    blockOwnerDeletion: true
    controller: true
    kind: XAWSInfrastructure
    name: platform-aws
    uid: 2de955c7-6718-5df2-94b0-e2a41302d6ba
```

</details>

Run `xprin update-goldens` again whenever the render output changes on purpose, and review the diff of the golden files in your version control.

### Example 4: Diff and Dyff (golden-file) assertions

**File**: [`mytests/6_assertions/example3_golden_file_xprin.yaml`](mytests/6_assertions/example3_golden_file_xprin.yaml)

- **Diff** assertions compare the full render (or a single resource file) to a golden file using a byte-for-byte comparison.
- **Dyff** assertions do a structural YAML comparison and produce a human-readable diff on failure.

This example has a single test that **passes** when the render output matches the golden files. It uses `golden_full_render.yaml` (full render) and `golden_single_resource.yaml` (one resource: `Cluster/platform-aws-rds`). Each assertion has `name` and `expected` (path to the golden file, relative to the test suite file). The optional `resource` field (e.g. `Cluster/platform-aws-rds`) compares that resource’s file instead of the full render.

**Golden files:** they are the ones that [Example 3](#example-3-generating-golden-files-with-update-goldens) generates, and they are in the repository too.

**To see failure output:**  
Edit one of the golden files (e.g. change a value) and run the test again; the assertion will fail and show the diff or dyff output.

```bash
xprin test examples/mytests/6_assertions/example3_golden_file_xprin.yaml -v --show-assertions
```

---

## Show All Results

The examples above enable the sections of the verbose output one by one: `--show-render`, `--show-validate`, `--show-hooks` and `--show-assertions`. `--show-all` enables all of them at once, and like them it requires `-v`. Adding individual `--show-*` flags to it has no extra effect.

This example uses a test that has pre-test hooks, CRDs for the validation, and assertions, so that every section is present. It compares the render with a golden file, so we generate it first (see [Example 3](#example-3-generating-golden-files-with-update-goldens) of the assertions).

```bash
xprin update-goldens examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml
xprin test examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml -v --show-all
```

<details>
<summary><strong>Output</strong></summary>

```
➜ xprin update-goldens examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml
--- PASS: Successful test with hooks, validation, and assertions (0.58s)
    Golden Files:
        [✓] Full render matches golden (using diff): /Users/myuser/repos/xprin/examples/mytests/7_show_all/golden_full_render.yaml
ok	examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml	0.579s

➜ xprin test examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml -v --show-all
=== RUN   Successful test with hooks, validation, and assertions
--- PASS: Successful test with hooks, validation, and assertions (1.33s)
    Pre-test Hooks:
        [✓] this is a pre-test hook
            hello
    Render:
        ├── XAWSInfrastructure/platform-aws
        ├── Cluster/platform-aws-rds
        └── SecurityGroup/platform-aws-sg
    Validate:
        [✓] aws.example.com/v1, Kind=XAWSInfrastructure, platform-aws validated successfully
        [✓] rds.aws.upbound.io/v1beta1, Kind=Cluster, platform-aws-rds validated successfully
        [✓] ec2.aws.upbound.io/v1beta1, Kind=SecurityGroup, platform-aws-sg validated successfully
        Total 3 resources: 0 missing schemas, 3 success cases, 0 failure cases
    Assertions:
        [✓] Number of resources - found 3 resources (as expected)
        [✓] SecurityGroup should exist - resources found: SecurityGroup/platform-aws-sg
        [✓] Full render matches golden (using diff) - files match
        Total: 3 assertions, 3 successful, 0 failed, 0 errors
    Post-test Hooks:
        [✓] this is a post-test hook
            bye
PASS
ok	examples/mytests/7_show_all/example1_used_in_readme_xprin.yaml	1.334s
```

</details>
