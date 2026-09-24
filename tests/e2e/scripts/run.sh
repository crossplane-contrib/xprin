#!/bin/bash
# run.sh - Single test runner for e2e tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROJECT_ROOT="${PROJECT_ROOT%/}"
EXPECTED_DIR="${SCRIPT_DIR}/../expected"
TESTCASES_FILE="${SCRIPT_DIR}/testcases.sh"
NORMALIZE_SCRIPT="${SCRIPT_DIR}/normalize.sh"
GEN_INVALID_TESTS_SCRIPT="${SCRIPT_DIR}/gen-invalid-tests.sh"
E2E_TESTS_DIR="${PROJECT_ROOT}/examples/mytests/0_e2e"
XPRIN_BIN="${XPRIN_BIN:-${PROJECT_ROOT}/xprin}"
CROSSPLANE_VERSION="${CROSSPLANE_VERSION:-}"
DEFAULT_TIERS="${DEFAULT_TIERS:-v2}"
STATUS=0

cd "${PROJECT_ROOT}"

# Detect Crossplane CLI tier (v1 / v2legacy / v2) from binary - same logic as regen-expected.sh.
# v1       = crossplane/crossplane v1.x
# v2legacy = crossplane/cli v2.0–v2.2 (uses beta validate, no resource validate)
# v2       = crossplane/cli v2.3+ (uses resource validate)
xp_tier_from_binary() {
    local bin="$1"
    local ver
    ver="$("${bin}" version --client 2>/dev/null | cut -d':' -f2 | xargs || true)"
    if [[ "${ver}" == v1.* ]]; then
        echo "v1"
    elif [[ "${ver}" == v2.* ]]; then
        local minor
        minor=$(echo "${ver#v2.}" | cut -d'.' -f1)
        if [[ "${minor}" -lt 3 ]] 2>/dev/null; then
            echo "v2legacy"
        else
            echo "v2"
        fi
    else
        echo "v2"
    fi
}

if [ ! -f "${TESTCASES_FILE}" ]; then
    echo "Test case list not found: ${TESTCASES_FILE}"
    exit 1
fi
if [ ! -x "${XPRIN_BIN}" ]; then
    echo "xprin binary not found or not executable: ${XPRIN_BIN}"
    exit 1
fi
if [ ! -f "${NORMALIZE_SCRIPT}" ]; then
    echo "Normalize script not found: ${NORMALIZE_SCRIPT}"
    exit 1
fi

# shellcheck source=/dev/null
source "${TESTCASES_FILE}"

XP_TIER=$(xp_tier_from_binary crossplane)

# Guardrail: when EXPECTED_XP_TIER is set (optional; e.g. by Earthly tier targets), ensure the Crossplane in PATH matches.
if [ -n "${EXPECTED_XP_TIER:-}" ]; then
    if [ "${XP_TIER}" != "${EXPECTED_XP_TIER}" ]; then
        echo "E2E guardrail: expected CLI tier ${EXPECTED_XP_TIER}, but crossplane binary reports tier ${XP_TIER}"
        echo "  Crossplane version: $(crossplane version --client 2>/dev/null || true)"
        exit 1
    fi
fi

# Build global xprin args. CROSSPLANE_VERSION is set by e2e-run (v2 tier only).
XPRIN_ARGS=()
if [[ -n "${CROSSPLANE_VERSION}" ]]; then
    # Normalize to always carry a leading 'v'
    [[ "${CROSSPLANE_VERSION}" == v* ]] || CROSSPLANE_VERSION="v${CROSSPLANE_VERSION}"
    XPRIN_ARGS=("--crossplane-version=${CROSSPLANE_VERSION}")
fi

TEST_CASES=($(compgen -v | grep '^testcase_' | grep -v '_exit' | grep -v '_tiers' | LC_ALL=C sort))
if [ "${#TEST_CASES[@]}" -eq 0 ]; then
    echo "No test cases defined in ${TESTCASES_FILE}"
    exit 1
fi

PASSED=0
FAILED=0
SKIPPED=0
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
    # Per-tier exit code takes precedence over the global _exit fallback.
    tier_exit_var="${test_var}_exit_${XP_TIER}"
    if compgen -v | grep -q "^${tier_exit_var}$"; then
        expected_exit="${!tier_exit_var}"
    else
        exit_var="${test_var}_exit"
        expected_exit="${!exit_var:-0}"
    fi

    if [ -z "${test_id}" ] || [ -z "${test_args}" ]; then
        echo "Invalid test case entry: ${test_var}"
        exit 1
    fi

    # Run testcase only if XP_TIER is in testcase_NNN_tiers (fallback: DEFAULT_TIERS).
    tiers_var="${test_var}_tiers"
    if compgen -v | grep -q "^${tiers_var}$"; then
        allowed_tiers="${!tiers_var}"
    else
        allowed_tiers="${DEFAULT_TIERS}"
    fi
    if ! echo " ${allowed_tiers} " | grep -q " ${XP_TIER} "; then
        echo "SKIP: testcase_${test_id} (tier ${XP_TIER} not in: ${allowed_tiers})"
        echo ""
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    echo "Running testcase_${test_id}..."
    read -ra cmd_args <<< "${test_args}"
    echo "Command: xprin test ${XPRIN_ARGS[@]:+${XPRIN_ARGS[*]} }${cmd_args[*]}"

    TMPDIR="$(mktemp -d)"
    TMPDIRS+=("${TMPDIR}")

    # Lookup order: tier-specific (.v1 / .v2legacy) → shared (.v1_v2legacy for legacy tiers) → default (.output)
    EXPECTED_TIER_FILE="${EXPECTED_DIR}/testcase_${test_id}.${XP_TIER}.output"
    EXPECTED_SHARED_FILE="${EXPECTED_DIR}/testcase_${test_id}.v1_v2legacy.output"
    if [ -f "${EXPECTED_TIER_FILE}" ]; then
        EXPECTED_OUTPUT="${EXPECTED_TIER_FILE}"
    elif [[ "${XP_TIER}" == v1 || "${XP_TIER}" == v2legacy ]] && [ -f "${EXPECTED_SHARED_FILE}" ]; then
        EXPECTED_OUTPUT="${EXPECTED_SHARED_FILE}"
    else
        EXPECTED_OUTPUT="${EXPECTED_DIR}/testcase_${test_id}.output"
    fi
    ACTUAL_OUTPUT="${TMPDIR}/actual.output"
    NORMALIZED_OUTPUT="${TMPDIR}/normalized.output"

    set +e
    "${XPRIN_BIN}" test "${XPRIN_ARGS[@]}" "${cmd_args[@]}" > "${ACTUAL_OUTPUT}" 2>&1
    EXIT_CODE=$?
    set -e

    "${NORMALIZE_SCRIPT}" "${ACTUAL_OUTPUT}" > "${NORMALIZED_OUTPUT}"

    TEST_FAILED=0

    # When CROSSPLANE_VERSION is set and the testcase uses --debug, assert that the pinned
    # controller image appears in the raw debug output (validates --crossplane-image wiring).
    if [[ -n "${CROSSPLANE_VERSION}" ]] && printf '%s\n' "${cmd_args[@]}" | grep -qx -- '--debug'; then
        expected_image="xpkg.crossplane.io/crossplane/crossplane:${CROSSPLANE_VERSION}"
        if ! grep -q "${expected_image}" "${ACTUAL_OUTPUT}"; then
            echo "  FAIL: crossplane-image assertion: '${expected_image}' not found in --debug output"
            TEST_FAILED=1
        fi
    fi

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
echo "xprin binary:                  ${XPRIN_BIN}"
echo "xprin version:                 $("${XPRIN_BIN}" version)"
echo "Crossplane CLI version:        $(crossplane version --client | cut -d':' -f2 | xargs)"
echo "Crossplane CLI tier:           ${XP_TIER}"
echo "Crossplane Controller version: ${CROSSPLANE_VERSION:-"(not pinned)"}"
echo ""

# E2E results
echo "--- E2E results ---"
echo "Total:  $((PASSED + FAILED + SKIPPED))  Passed: ${PASSED}  Failed: ${FAILED}  Skipped: ${SKIPPED}"

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
