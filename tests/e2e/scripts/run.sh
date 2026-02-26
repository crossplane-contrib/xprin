#!/bin/bash
# run.sh - Single test runner for e2e tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EXPECTED_DIR="${SCRIPT_DIR}/../expected"
TESTCASES_FILE="${SCRIPT_DIR}/testcases.sh"
NORMALIZE_SCRIPT="${SCRIPT_DIR}/normalize.sh"
GEN_INVALID_TESTS_SCRIPT="${SCRIPT_DIR}/gen-invalid-tests.sh"
E2E_TESTS_DIR="${PROJECT_ROOT}/examples/mytests/0_e2e"
XPRIN_BIN="${XPRIN_BIN:-${PROJECT_ROOT}/xprin}"
STATUS=0

cd "${PROJECT_ROOT}"

# Detect Crossplane major version (1 or 2) from binary - same logic as regen-expected.sh
xp_major_from_binary() {
    local bin="$1"
    local ver
    ver="$("${bin}" version --client 2>/dev/null | cut -d':' -f2 | xargs || true)"
    if [[ "${ver}" == v1.* ]]; then
        echo 1
    elif [[ "${ver}" == v2.* ]]; then
        echo 2
    else
        echo 2
    fi
}

if [ ! -f "${TESTCASES_FILE}" ]; then
    echo "Test case list not found: ${TESTCASES_FILE}"
    exit 1
fi
if [ ! -x "${XPRIN_BIN}" ]; then
    echo "xprin binary not found or not executable: ${XPRIN_BIN}"
    echo "Run: make xprin-build"
    exit 1
fi
if [ ! -f "${NORMALIZE_SCRIPT}" ]; then
    echo "Normalize script not found: ${NORMALIZE_SCRIPT}"
    exit 1
fi

# shellcheck source=/dev/null
source "${TESTCASES_FILE}"

XP_MAJOR=$(xp_major_from_binary crossplane)

# Guardrail: when EXPECTED_XP_MAJOR is set (optional; e.g. by Earthly e2e-v1/e2e-v2), ensure the Crossplane in PATH matches.
if [ -n "${EXPECTED_XP_MAJOR:-}" ]; then
    if [ "${XP_MAJOR}" -ne "${EXPECTED_XP_MAJOR}" ]; then
        echo "E2E guardrail: expected Crossplane major version ${EXPECTED_XP_MAJOR}, but crossplane binary reports major ${XP_MAJOR}"
        echo "  Crossplane version: $(crossplane version --client 2>/dev/null || true)"
        exit 1
    fi
fi

TEST_CASES=($(compgen -v | grep '^testcase_' | grep -v '_exit$' | LC_ALL=C sort))
if [ "${#TEST_CASES[@]}" -eq 0 ]; then
    echo "No test cases defined in ${TESTCASES_FILE}"
    exit 1
fi

PASSED=0
FAILED=0
FAILED_TESTS=()
TMPDIRS=()

export E2E_TESTS_DIR

# Set trap before generating so we clean up on any exit (including script failure).
trap 'for d in "${TMPDIRS[@]}"; do rm -rf "${d}"; done; rm -f "${E2E_TESTS_DIR}"/generated_*.yaml' EXIT

# Generate schema-invalid e2e testsuite files so they are not committed.
"${GEN_INVALID_TESTS_SCRIPT}"

for test_var in "${TEST_CASES[@]}"; do
    test_id="${test_var#testcase_}"
    test_args="${!test_var}"
    exit_var="${test_var}_exit"
    expected_exit="${!exit_var:-0}"

    if [ -z "${test_id}" ] || [ -z "${test_args}" ]; then
        echo "Invalid test case entry: ${test_var}"
        exit 1
    fi

    echo "Running testcase_${test_id}..."
    read -ra cmd_args <<< "${test_args}"
    echo "Command: xprin test ${cmd_args[*]}"

    TMPDIR="$(mktemp -d)"
    TMPDIRS+=("${TMPDIR}")

    # Prefer version-specific expected file (.v1.output / .v2.output) if present; else default .output
    EXPECTED_VERSIONED="${EXPECTED_DIR}/testcase_${test_id}.v${XP_MAJOR}.output"
    if [ -f "${EXPECTED_VERSIONED}" ]; then
        EXPECTED_OUTPUT="${EXPECTED_VERSIONED}"
    else
        EXPECTED_OUTPUT="${EXPECTED_DIR}/testcase_${test_id}.output"
    fi
    ACTUAL_OUTPUT="${TMPDIR}/actual.output"
    NORMALIZED_OUTPUT="${TMPDIR}/normalized.output"

    set +e
    "${XPRIN_BIN}" test "${cmd_args[@]}" > "${ACTUAL_OUTPUT}" 2>&1
    EXIT_CODE=$?
    set -e

    "${NORMALIZE_SCRIPT}" "${ACTUAL_OUTPUT}" > "${NORMALIZED_OUTPUT}"

    TEST_FAILED=0

    if [ ! -f "${EXPECTED_OUTPUT}" ]; then
        echo "FAIL: Expected file not found: ${EXPECTED_OUTPUT}"
        echo "Please create the expected file manually."
        TEST_FAILED=1
    elif ! diff -u "${EXPECTED_OUTPUT}" "${NORMALIZED_OUTPUT}" > /dev/null; then
        echo "FAIL: output mismatch for testcase_${test_id}"
        # echo "Expected:"
        # cat "${EXPECTED_OUTPUT}"
        # echo ""
        # echo "Actual:"
        # cat "${NORMALIZED_OUTPUT}"
        # echo ""
        echo "Diff:"
        diff -u "${EXPECTED_OUTPUT}" "${NORMALIZED_OUTPUT}" || true
        TEST_FAILED=1
    fi

    if [ "${expected_exit}" -eq 0 ]; then
        if [ ${EXIT_CODE} -ne 0 ]; then
            echo "FAIL: expected exit code 0, got ${EXIT_CODE}"
            TEST_FAILED=1
        fi
    else
        if [ ${EXIT_CODE} -eq 0 ]; then
            echo "FAIL: expected non-zero exit code, got 0"
            TEST_FAILED=1
        fi
    fi

    if [ ${TEST_FAILED} -eq 1 ]; then
        FAILED=$((FAILED + 1))
        FAILED_TESTS+=("testcase_${test_id}")
    else
        echo "PASS: testcase_${test_id}"
        PASSED=$((PASSED + 1))
    fi

    rm -rf "${TMPDIR}"
    echo ""
done

# Environment (debug info)
echo ""
echo "--- Environment ---"
echo "xprin binary:  ${XPRIN_BIN}"
echo "xprin version: $("${XPRIN_BIN}" version)"
echo "Crossplane:    $(crossplane version --client | cut -d':' -f2 | xargs)"
echo ""

# E2E results
echo "--- E2E results ---"
echo "Total:  $((PASSED + FAILED))  Passed: ${PASSED}  Failed: ${FAILED}"

if [ ${FAILED} -gt 0 ]; then
    echo "Failed tests:"
    for test in "${FAILED_TESTS[@]}"; do
        echo "  - ${test}"
    done
    STATUS=1
fi

echo ""
if [ ${STATUS} -eq 0 ]; then
    echo "All tests passed."
fi

exit ${STATUS}
