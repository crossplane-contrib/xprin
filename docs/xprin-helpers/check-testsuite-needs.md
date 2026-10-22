# check-testsuite-needs

Checks that the test cases of testsuite files list in `needs` the test cases whose outputs they use.

## Why This Tool?

A test case that uses the outputs of another one (`{{ .Tests.<id>.Outputs... }}`) has to list it in [`needs`](../how-it-works.md#dependencies-between-test-cases-needs). Until xprin 0.5 there was no `needs`: xprin found those dependencies by reading the templates. That is deprecated, and it goes away in 0.7.0. This tool checks the test cases of your testsuite files, and shows what is missing, so that you can add it.

## Installation

See [Installation](../xprin-helpers.md#installation).

## Usage

```bash
xprin-helpers check-testsuite-needs <targets>... [-v] [--quiet]
```

The targets are the same as for `xprin test`: files, directories, and `dir/...` for recursive directories. Files must be named `xprin.yaml` or `*_xprin.yaml`.

| Option | Description |
|--------|-------------|
| `-v, --verbose` | Show every test case, also the ones that have nothing to say |
| `-q, --quiet` | Suppress `[no testsuite files]` and `[no test cases found]` messages |

The exit code is 1 when anything is missing or has to be fixed by a person, so it can check in a pipeline that the files are migrated.

## What It Does

For **every test case of every file**, also in the files that already use `needs`, it looks for the test cases whose outputs it uses through `.Tests`, in the same places as `xprin test` renders them: its fields, the `common` configuration it inherits, and its hooks. It shows the ones that are missing in its `needs`.

- A `needs` that isn't used through `.Tests` is never a problem: it is a legitimate way to order test cases, for example when they share state.
- A test case can be listed before or after the ones it needs.

## The Output

The output is the one of `xprin test`: a result for each test case, and a line for each file.

| Result | When | Shown without `-v` |
|--------|------|--------------------|
| `PASS` | it uses the outputs of other test cases, and it lists all of them | no |
| `SKIP` | it uses no other test case | no |
| `FAIL` | something is missing in its `needs`, or has to be fixed by a person | yes, under `Missing Needs:` with a `[x]` for each, or with `[!]` |
| `SKIP`, with a reason | it uses `.Tests` in a way that can't be traced, and it lists needs: they can't be checked, and a person took care of them | yes, with `[!]` and the reason |

| Mark | Meaning |
|------|---------|
| `[x]` | A need that it uses and does not list, one for each, under `Missing Needs:`: add it to its `needs` |
| `[!]` | What a person has to fix, see [What Requires a Person](#what-requires-a-person), or the reason of a skip |

```
➜ xprin-helpers check-testsuite-needs tests/...
--- FAIL: Second reconciliation loop (0.00s)
    Missing Needs:
        [x] aws_first
FAIL
FAIL	tests/aws_xprin.yaml	0.002s
ok	tests/gcp_xprin.yaml	0.002s
FAIL
Total:
    [!] 1 testsuite file misses needs: see Missing Needs above
```

Like `xprin test` and `go test`, a run in which any file fails ends with a `FAIL` line, and the `Total:` that follows says what is left to do: the files that miss needs, and the files that require a person (see [What Requires a Person](#what-requires-a-person)).

With `-v` all the test cases are shown, also the ones with nothing to say:

```
➜ xprin-helpers check-testsuite-needs -v tests/gcp_xprin.yaml
=== RUN   First reconciliation loop
--- SKIP: First reconciliation loop (0.00s)
=== RUN   Second reconciliation loop
--- PASS: Second reconciliation loop (0.00s)
PASS
ok	tests/gcp_xprin.yaml	0.002s
```

## What Requires a Person

It doesn't guess. It fails these test cases with a line marked `[!]`, and exits with 1:

| Case | Why |
|------|-----|
| `uses .Tests in a way that can't be traced` | For example `range .Tests`, or an ID computed in the template: it can't know which test cases it uses. When the test case lists some `needs`, it is skipped with that reason instead: a person took care of them, and they can't be checked |
| `uses .Tests.<id>, but no test case has that ID` | Most likely a typo |
| `uses its own outputs` | A test case can't use outputs that don't exist yet |
| `listing the needs it misses would make the test cases need each other` | Adding the missing `needs` would create a cycle |

A file that is not valid, for example with a `needs` that points to an ID that doesn't exist, is reported as an error.

## Examples

```bash
# Check a testsuite file
xprin-helpers check-testsuite-needs tests/aws_xprin.yaml

# Check all the testsuite files under tests/, for example in a pipeline
xprin-helpers check-testsuite-needs tests/...
```

## Getting Help

```bash
xprin-helpers check-testsuite-needs --help
```
