# Test Suite Specification

A test suite file defines test cases for xprin. Its name must be `xprin.yaml` or end with `_xprin.yaml` (see [Discovering Testsuite Files](how-it-works.md#discovering-testsuite-files)). For a complete file, see [Getting Started](getting-started.md) and the [examples](../examples/README.md).

This page describes the format, key by key. The tables are generated from the Go types in `internal/api`; for what xprin does with each key, see [How It Works](how-it-works.md).

## Rules

- **`common` and test cases are merged.** `inputs`, `patches`, `hooks` and `assertions` can be set in `common`, in a test case, or in both, see [Common vs Test-Level Configuration](how-it-works.md#common-vs-test-level-configuration). A property that is 🧩 merged or 🔀 one-of in the tables (see [Requirement](#requirement)) is required in the merged result, so it can come from either.
- **`xr` or `claim`, not both.** One of them is required, in the test case or in `common`.
- **`needs` lists the `id`s of other test cases.** A test case can't list itself, the `needs` can't form a cycle, and each one has to be the `id` of a test case in the same file, listed before or after it. A test case that needs another one runs after it, and is skipped if it failed or was skipped. See [Dependencies Between Test Cases](how-it-works.md#dependencies-between-test-cases-needs), and `xprin graph` to see the resulting graph.
- **Paths** are absolute, relative to the test suite file, or built with template variables (`{{ .Repositories.myrepo }}/...`), see [Path Resolution](how-it-works.md#path-resolution) and [Template Variable Expansion](how-it-works.md#template-variable-expansion).
- **Hooks** are shell commands that run before (`pre-test`) and after (`post-test`) the test case, see [Hooks Execution](how-it-works.md#hooks-execution).
- **Assertions** are grouped by engine (`xprin`, `diff`, `dyff`), see [Assertions](assertions.md).

## Index

Every property of the file, by its full path:

<!-- DOCS_START_INDEX -->

- [`common`](#common)
- [`common.inputs`](#inputs)
- [`common.inputs.claim`](#inputs.claim)
- [`common.inputs.xr`](#inputs.xr)
- [`common.inputs.composition`](#inputs.composition)
- [`common.inputs.functions`](#inputs.functions)
- [`common.inputs.crds[*]`](#inputs.crds)
- [`common.inputs.context-files`](#inputs.context-files)
- [`common.inputs.context-values`](#inputs.context-values)
- [`common.inputs.observed-resources`](#inputs.observed-resources)
- [`common.inputs.extra-resources`](#inputs.extra-resources)
- [`common.inputs.function-credentials`](#inputs.function-credentials)
- [`common.patches`](#patches)
- [`common.patches.xrd`](#patches.xrd)
- [`common.patches.connection-secret`](#patches.connection-secret)
- [`common.patches.connection-secret-name`](#patches.connection-secret-name)
- [`common.patches.connection-secret-namespace`](#patches.connection-secret-namespace)
- [`common.hooks`](#hooks)
- [`common.hooks.pre-test[*]`](#hookspre-test-and-hookspost-test)
- [`common.hooks.pre-test[*].name`](#hookspre-test-and-hookspost-test.name)
- [`common.hooks.pre-test[*].run`](#hookspre-test-and-hookspost-test.run)
- [`common.hooks.post-test[*]`](#hookspre-test-and-hookspost-test)
- [`common.hooks.post-test[*].name`](#hookspre-test-and-hookspost-test.name)
- [`common.hooks.post-test[*].run`](#hookspre-test-and-hookspost-test.run)
- [`common.assertions`](#assertions)
- [`common.assertions.xprin[*]`](#assertionsxprin)
- [`common.assertions.xprin[*].name`](#assertionsxprin.name)
- [`common.assertions.xprin[*].type`](#assertionsxprin.type)
- [`common.assertions.xprin[*].resource`](#assertionsxprin.resource)
- [`common.assertions.xprin[*].field`](#assertionsxprin.field)
- [`common.assertions.xprin[*].operator`](#assertionsxprin.operator)
- [`common.assertions.xprin[*].value`](#assertionsxprin.value)
- [`common.assertions.diff[*]`](#assertionsdiff-and-assertionsdyff)
- [`common.assertions.diff[*].name`](#assertionsdiff-and-assertionsdyff.name)
- [`common.assertions.diff[*].expected`](#assertionsdiff-and-assertionsdyff.expected)
- [`common.assertions.diff[*].resource`](#assertionsdiff-and-assertionsdyff.resource)
- [`common.assertions.dyff[*]`](#assertionsdiff-and-assertionsdyff)
- [`common.assertions.dyff[*].name`](#assertionsdiff-and-assertionsdyff.name)
- [`common.assertions.dyff[*].expected`](#assertionsdiff-and-assertionsdyff.expected)
- [`common.assertions.dyff[*].resource`](#assertionsdiff-and-assertionsdyff.resource)
- [`tests[*]`](#tests)
- [`tests[*].name`](#tests.name)
- [`tests[*].id`](#tests.id)
- [`tests[*].needs[*]`](#tests.needs)
- [`tests[*].inputs`](#inputs)
- [`tests[*].inputs.claim`](#inputs.claim)
- [`tests[*].inputs.xr`](#inputs.xr)
- [`tests[*].inputs.composition`](#inputs.composition)
- [`tests[*].inputs.functions`](#inputs.functions)
- [`tests[*].inputs.crds[*]`](#inputs.crds)
- [`tests[*].inputs.context-files`](#inputs.context-files)
- [`tests[*].inputs.context-values`](#inputs.context-values)
- [`tests[*].inputs.observed-resources`](#inputs.observed-resources)
- [`tests[*].inputs.extra-resources`](#inputs.extra-resources)
- [`tests[*].inputs.function-credentials`](#inputs.function-credentials)
- [`tests[*].patches`](#patches)
- [`tests[*].patches.xrd`](#patches.xrd)
- [`tests[*].patches.connection-secret`](#patches.connection-secret)
- [`tests[*].patches.connection-secret-name`](#patches.connection-secret-name)
- [`tests[*].patches.connection-secret-namespace`](#patches.connection-secret-namespace)
- [`tests[*].hooks`](#hooks)
- [`tests[*].hooks.pre-test[*]`](#hookspre-test-and-hookspost-test)
- [`tests[*].hooks.pre-test[*].name`](#hookspre-test-and-hookspost-test.name)
- [`tests[*].hooks.pre-test[*].run`](#hookspre-test-and-hookspost-test.run)
- [`tests[*].hooks.post-test[*]`](#hookspre-test-and-hookspost-test)
- [`tests[*].hooks.post-test[*].name`](#hookspre-test-and-hookspost-test.name)
- [`tests[*].hooks.post-test[*].run`](#hookspre-test-and-hookspost-test.run)
- [`tests[*].assertions`](#assertions)
- [`tests[*].assertions.xprin[*]`](#assertionsxprin)
- [`tests[*].assertions.xprin[*].name`](#assertionsxprin.name)
- [`tests[*].assertions.xprin[*].type`](#assertionsxprin.type)
- [`tests[*].assertions.xprin[*].resource`](#assertionsxprin.resource)
- [`tests[*].assertions.xprin[*].field`](#assertionsxprin.field)
- [`tests[*].assertions.xprin[*].operator`](#assertionsxprin.operator)
- [`tests[*].assertions.xprin[*].value`](#assertionsxprin.value)
- [`tests[*].assertions.diff[*]`](#assertionsdiff-and-assertionsdyff)
- [`tests[*].assertions.diff[*].name`](#assertionsdiff-and-assertionsdyff.name)
- [`tests[*].assertions.diff[*].expected`](#assertionsdiff-and-assertionsdyff.expected)
- [`tests[*].assertions.diff[*].resource`](#assertionsdiff-and-assertionsdyff.resource)
- [`tests[*].assertions.dyff[*]`](#assertionsdiff-and-assertionsdyff)
- [`tests[*].assertions.dyff[*].name`](#assertionsdiff-and-assertionsdyff.name)
- [`tests[*].assertions.dyff[*].expected`](#assertionsdiff-and-assertionsdyff.expected)
- [`tests[*].assertions.dyff[*].resource`](#assertionsdiff-and-assertionsdyff.resource)

<!-- DOCS_END_INDEX -->

## Requirement

Every table of the reference has a **Requirement** column, with the emojis of what the property needs. A property may have more than one.

<!-- DOCS_START_LEGEND -->

| | Name | Meaning |
| :-: | :--- | :------ |
| ✅ | required | The property has to be here. |
| 🧩 | merged | The property has to be in `common` or in the test case, in the merged result: it can't be in neither. If it is in both, the one of the test case prevails. |
| 🔀 | one-of | Exactly one of the property and the one it is exclusive with, named in its description, has to be set. |
| 🔹 | optional | The property can be missing. |

<!-- DOCS_END_LEGEND -->

## Reference

The top level of the file:

<!-- DOCS_START_ROOT -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="root.common"></a>`common` | [`common`](#common) | 🔹 | Common config for all tests |
| <a id="root.tests"></a>`tests` | list of [`tests[*]`](#tests) | ✅ | List of test cases |

<!-- DOCS_END_ROOT -->

### `common`

Shared by every test case.

<!-- DOCS_START_COMMON -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="common.inputs"></a>`inputs` | [`inputs`](#inputs) | 🧩 | Common inputs (composition, Claim/XR, etc.) for all testcases |
| <a id="common.patches"></a>`patches` | [`patches`](#patches) | 🔹 | Common XR patching configuration for all testcases |
| <a id="common.hooks"></a>`hooks` | [`hooks`](#hooks) | 🔹 | Common hooks for all testcases |
| <a id="common.assertions"></a>`assertions` | [`assertions`](#assertions) | 🔹 | Common assertions to validate rendered resources for all testcases |

<!-- DOCS_END_COMMON -->

### `tests[*]`

A test case: an element of `tests`.

<!-- DOCS_START_TESTS -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="tests.name"></a>`name` | `string` | ✅ | Descriptive name for the testcase |
| <a id="tests.id"></a>`id` | `string` | 🔹 | Unique identifier for the testcase |
| <a id="tests.needs"></a>`needs` | list of `string` (at least 1) | 🔹 | IDs of the testcases that this one depends on |
| <a id="tests.inputs"></a>`inputs` | [`inputs`](#inputs) | 🧩 | Inputs of a testcase |
| <a id="tests.patches"></a>`patches` | [`patches`](#patches) | 🔹 | XR patching configuration |
| <a id="tests.hooks"></a>`hooks` | [`hooks`](#hooks) | 🔹 | Execution hooks |
| <a id="tests.assertions"></a>`assertions` | [`assertions`](#assertions) | 🔹 | Assertions to validate rendered resources |

<!-- DOCS_END_TESTS -->

### `inputs`

Used in `common.inputs` and `tests[*].inputs`.

<!-- DOCS_START_INPUTS -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="inputs.claim"></a>`claim` | `string` | 🧩🔀 | Path to Claim file. Exclusive with: `xr` |
| <a id="inputs.xr"></a>`xr` | `string` | 🧩🔀 | Path to XR file. Exclusive with: `claim` |
| <a id="inputs.composition"></a>`composition` | `string` | 🧩 | Path to composition file |
| <a id="inputs.functions"></a>`functions` | `string` | 🧩 | Path to functions file or directory |
| <a id="inputs.crds"></a>`crds` | list of `string` | 🔹 | Paths to CRD files |
| <a id="inputs.context-files"></a>`context-files` | map of `string` | 🔹 | Map of context keys to file paths |
| <a id="inputs.context-values"></a>`context-values` | map of `string` | 🔹 | Map of context keys to inline values |
| <a id="inputs.observed-resources"></a>`observed-resources` | `string` | 🔹 | Path to observed resources file |
| <a id="inputs.extra-resources"></a>`extra-resources` | `string` | 🔹 | Path to extra resources file |
| <a id="inputs.function-credentials"></a>`function-credentials` | `string` | 🔹 | Path to function credentials file |

<!-- DOCS_END_INPUTS -->

### `patches`

Used in `common.patches` and `tests[*].patches`.

<!-- DOCS_START_PATCHES -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="patches.xrd"></a>`xrd` | `string` | 🔹 | Path to the XR's or Claim's XRD |
| <a id="patches.connection-secret"></a>`connection-secret` | `boolean` | 🔹 | When true, create a connection secret for the XR |
| <a id="patches.connection-secret-name"></a>`connection-secret-name` | `string` | 🔹 | Name of the connection secret |
| <a id="patches.connection-secret-namespace"></a>`connection-secret-namespace` | `string` | 🔹 | Namespace of the connection secret |

<!-- DOCS_END_PATCHES -->

### `hooks`

Used in `common.hooks` and `tests[*].hooks`. Hooks run before (`pre-test`) or after (`post-test`) the test case, each one a list of hooks.

<!-- DOCS_START_HOOKS -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="hooks.pre-test"></a>`pre-test` | list of [`hook`](#hookspre-test-and-hookspost-test) | 🔹 | Hooks that are executed before the testcase |
| <a id="hooks.post-test"></a>`post-test` | list of [`hook`](#hookspre-test-and-hookspost-test) | 🔹 | Hooks that are executed after the testcase |

<!-- DOCS_END_HOOKS -->

### `hooks.pre-test[*]` and `hooks.post-test[*]`

A hook, an element of `pre-test` or `post-test`.

<!-- DOCS_START_HOOK -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="hookspre-test-and-hookspost-test.name"></a>`name` | `string` | 🔹 | Descriptive name for the hook |
| <a id="hookspre-test-and-hookspost-test.run"></a>`run` | `string` | ✅ | Command to run |

<!-- DOCS_END_HOOK -->

See [Hooks Execution](how-it-works.md#hooks-execution).

### `assertions`

Used in `common.assertions` and `tests[*].assertions`.

<!-- DOCS_START_ASSERTIONS -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="assertions.xprin"></a>`xprin` | list of [`assertions.xprin[*]`](#assertionsxprin) | 🔹 | xprin assertions (in-process) |
| <a id="assertions.diff"></a>`diff` | list of [`golden-file assertion`](#assertionsdiff-and-assertionsdyff) | 🔹 | diff assertions (go-native compare to golden file) |
| <a id="assertions.dyff"></a>`dyff` | list of [`golden-file assertion`](#assertionsdiff-and-assertionsdyff) | 🔹 | dyff assertions (dyff between expected and actual) |

<!-- DOCS_END_ASSERTIONS -->

### `assertions.xprin[*]`

<!-- DOCS_START_ASSERTIONS_XPRIN -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="assertionsxprin.name"></a>`name` | `string` | ✅ | Descriptive name for the assertion |
| <a id="assertionsxprin.type"></a>`type` | `string` | ✅ | Type of assertion. Allowed values: `Count`, `Exists`, `NotExists`, `FieldType`, `FieldExists`, `FieldNotExists`, `FieldValue` |
| <a id="assertionsxprin.resource"></a>`resource` | `string` | 🔹 | Resource identifier for resource-based assertions (format: Kind/Name e.g. "Cluster/platform-aws-rds") |
| <a id="assertionsxprin.field"></a>`field` | `string` | 🔹 | Field path for field-based assertions (e.g., "metadata.name") |
| <a id="assertionsxprin.operator"></a>`operator` | `string` | 🔹 | Operator for field value assertions (== or is). Allowed values: `==`, `is` |
| <a id="assertionsxprin.value"></a>`value` | any | 🔹 | Expected value for the assertion |

<!-- DOCS_END_ASSERTIONS_XPRIN -->

Which of `resource`, `field`, `operator` and `value` an assertion needs depends on its `type`, see [Assertion types (xprin)](assertions.md#assertion-types-xprin).

### `assertions.diff[*]` and `assertions.dyff[*]`

<!-- DOCS_START_ASSERTIONS_GOLDEN -->

| Property | Type | Requirement | Description |
| :------- | :--- | :---------: | :---------- |
| <a id="assertionsdiff-and-assertionsdyff.name"></a>`name` | `string` | ✅ | Descriptive name for the assertion |
| <a id="assertionsdiff-and-assertionsdyff.expected"></a>`expected` | `string` | ✅ | Path to golden (expected) file |
| <a id="assertionsdiff-and-assertionsdyff.resource"></a>`resource` | `string` | 🔹 | Resource identifier for resource-based assertions (format: Kind/Name e.g. "Cluster/platform-aws-rds") |

<!-- DOCS_END_ASSERTIONS_GOLDEN -->

See [Golden-file assertions (diff and dyff)](assertions.md#golden-file-assertions-diff-and-dyff).

---

**Next Steps:**
- Learn how to assert rendered resources with [Assertions](assertions.md)
