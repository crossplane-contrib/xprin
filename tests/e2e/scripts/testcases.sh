#!/bin/bash
# testcases.sh - Test case definitions for acceptance tests
#
# This file defines all acceptance test cases. Each test case is a simple string variable
# containing space-separated arguments (test files and flags).
# The test ID is extracted from the variable name (e.g., testcase_001 -> "001")
# Available configuration options for each test case:
# - testcase_<ID>_exit: set an expected non-zero exit code.
# - testcase_<ID>_exit_<tier>: set a per-tier expected non-zero exit code (v1/v2legacy/v2).
# - testcase_<ID>_tiers="v1 v2legacy v2": run a testcase for specific tiers (space-separated). If it is not set, the testcase runs only for DEFAULT_TIERS (defined in run.sh).
# - testcase_<ID>_update_goldens=true: run `xprin update-goldens` (file targets only) before xprin test.

# Multiple Successful Files (Non-Verbose)
testcase_001="examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml"

# Multiple Successful Files (Verbose)
testcase_002="examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml examples/mytests/2_multiple_testcases/example2_multiple-reconciliation-loops-using-common_xprin.yaml -v"

# Combination of successful and failed testsuite files (Non-Verbose)
testcase_003="examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml examples/mytests/0_e2e/single_failure_xprin.yaml examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml"
testcase_003_exit=1

# Combination of successful and failed testsuite files (Verbose)
testcase_004="examples/mytests/1_simple_tests/example1_using-xr_xprin.yaml examples/mytests/0_e2e/single_failure_xprin.yaml examples/mytests/1_simple_tests/example2_using-claim_xprin.yaml -v"
testcase_004_exit=1

# Multiple Failures - Combined File (Non-Verbose)
testcase_005="examples/mytests/0_e2e/failures_xprin.yaml examples/mytests/0_e2e/generated_missing_required_inputs_xprin.yaml"
testcase_005_exit=1
testcase_005_update_goldens=true

# Multiple Failures - Combined File (Verbose)
testcase_006="examples/mytests/0_e2e/failures_xprin.yaml examples/mytests/0_e2e/generated_missing_required_inputs_xprin.yaml -v"
testcase_006_exit=1
testcase_006_update_goldens=true

# Multiple Failures - Combined File (Verbose, show flags)
testcase_007="examples/mytests/0_e2e/failures_xprin.yaml examples/mytests/0_e2e/generated_missing_required_inputs_xprin.yaml -v --show-render --show-validate --show-hooks --show-assertions"
testcase_007_exit=1
testcase_007_update_goldens=true

# Successful with hooks/validate/assertions (Verbose, show flags) + update-goldens verification
testcase_008="examples/mytests/0_e2e/success_xprin.yaml examples/mytests/6_assertions/ -v --show-render --show-validate --show-hooks --show-assertions"
testcase_008_update_goldens=true

# Test with Chained Outputs
testcase_009="examples/mytests/5_chained_tests/example1_chained-test-outputs_xprin.yaml -v --show-render --show-validate"

# Cross-Composition Chaining
testcase_010="examples/mytests/5_chained_tests/example2_cross-composition-chaining_xprin.yaml -v --show-render --show-validate"

# Invalid testsuite file
testcase_011="examples/mytests/0_e2e/generated_invalid_xprin.yaml -v --show-render --show-validate --show-hooks --show-assertions"
testcase_011_exit=1

# Validations with combinations of incomplete / complete XR and with / without patches.xrd (XRD v1)
testcase_012="examples/mytests/0_e2e/validations_xrd_v1_xprin.yaml --debug"
testcase_012_tiers="v1 v2legacy v2"
testcase_012_exit_v1=1
testcase_012_exit_v2=1

# --artifacts-dir: complete test (hooks/render/validate/assertions) + silent shell checks on artifact structure
testcase_013="examples/mytests/0_e2e/used_in_readme_xprin.yaml --artifacts-dir=/tmp/xprin-e2e-artifacts-test"
testcase_013_update_goldens=true
