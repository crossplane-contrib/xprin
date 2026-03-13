# E2E test scripts – how they work

## How tests are chosen (no “fetching” of 0_e2e)

- **`run.sh`** does not discover or “fetch” the `0_e2e` folder.
- It **sources** `testcases.sh`, which defines variables like:
  - `testcase_005="examples/mytests/0_e2e/failures_xprin.yaml ..."`
  - `testcase_006="..."`
  - `testcase_007="..."`
- Line 59 in `run.sh` collects **all shell variables** whose names start with `testcase_` and don’t end with `_exit`:
  - `TEST_CASES=($(compgen -v | grep '^testcase_' | grep -v '_exit$' | sort))`
- So the list of tests is exactly the **uncommented** `testcase_XXX=...` lines in `testcases.sh`. Right now only **005, 006, 007** are active; the rest are commented out.
- The `0_e2e` folder is used only because those three test cases **hardcode** paths under `examples/mytests/0_e2e/` in `testcases.sh`. There is no automatic discovery of that directory.

## What each test run does

For each `testcase_XXX`:

1. Run: `xprin test <arguments from testcase_XXX>` (e.g. the YAML files and flags).
2. Save stdout+stderr to a temp file, then run **`normalize.sh`** on it (strip timing, paths, etc.).
3. Compare the normalized output to **`tests/e2e/expected/testcase_XXX.output`** (or `testcase_XXX.v1.output` / `testcase_XXX.v2.output` if present for your Crossplane major version).
4. If the expected file is missing → FAIL (line 114–118 in `run.sh`).
5. If `diff` of expected vs normalized output is non‑empty → FAIL (line 118–128); that’s what sets `TEST_FAILED=1` around line 128–129.
6. If exit code doesn’t match `testcase_XXX_exit` (e.g. 005/006/007 expect non‑zero) → FAIL.

So **line 130** (`TEST_FAILED=1`) is set when either the **output diff** or the **exit code check** fails.

## About `example2_golden_file_xprin.yaml` (6_assertions)

- **`run.sh` does not run** `examples/mytests/6_assertions/example2_golden_file_xprin.yaml`. That path is not in `testcases.sh`.
- It only appears in **`examples/README.md`** as a **manual** example:
  - `xprin test examples/mytests/6_assertions/example2_golden_file_xprin.yaml -v --show-assertions`
- If you run that command yourself, its success/failure is separate from the e2e script. The e2e script only runs what’s defined in `testcases.sh` (005, 006, 007).

## Why e2e tests might “fail nonstop”

1. **Output mismatch (most common)**  
   The normalized output of `xprin test ...` no longer matches `tests/e2e/expected/testcase_XXX.output`. Typical causes:
   - Code or test YAML changed so that messages or ordering changed.
   - **Path normalization**: `normalize.sh` only rewrites paths that look like `/Users/.../repos/.../xprin`. If your project path is different (e.g. `.../Desktop/personal/xprin`), absolute paths in the output might not get normalized and the diff will fail. Fix: add a similar `sed` rule in `normalize.sh` for your path, or regenerate expected files from your machine (see below).

2. **Missing expected file**  
   You’ll see: `FAIL: Expected file not found: .../testcase_XXX.output`. Fix: create it (or uncomment the test only after generating it).

3. **Wrong exit code**  
   Test expects non‑zero (e.g. 005/006/007) but `xprin` exits 0, or the opposite. Fix: align behavior or update `testcase_XXX_exit` in `testcases.sh`.

4. **Crossplane version**  
   Expected files can be `testcase_XXX.v1.output` or `testcase_XXX.v2.output`. If your Crossplane major version doesn’t match what was used to generate those, output can differ. Fix: use the matching expected file or regenerate.

## Updating expected output (regen)

If you intentionally changed behavior or fixed normalization and want to accept the current output as the new baseline:

```bash
# From project root
bash tests/e2e/scripts/regen-expected.sh
```

This reruns each active test case and overwrites `tests/e2e/expected/testcase_XXX.output` (and versioned variants) with the current normalized output. Run it only when the current output is what you want to lock in.

## Quick checklist when e2e fails

- Run: `make test-e2e` (or `bash tests/e2e/scripts/run.sh` from project root).
- Note which `testcase_XXX` failed and whether the message is “output mismatch” or “Expected file not found” or “expected exit code …”.
- For output mismatch: look at the printed `diff`; if it’s only path or timing differences, either extend `normalize.sh` or run `regen-expected.sh` if the new output is correct.
- For `example2_golden_file_xprin.yaml`: run that with `xprin test ...` manually; fixing that is independent of `run.sh` unless you add it to `testcases.sh`.
