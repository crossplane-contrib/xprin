#!/bin/bash
# regen-expected.sh - Regenerate e2e expected output files (run inside container)
#
# Behavior is driven by the following environment variables:
#
#   GENERATE=true
#     Find crossplane in PATH -> get version -> XP_MAJOR (same as run.sh) -> output suffix
#     (.v1.output or .output) -> run one generate pass.
#
#   CLEANUP=true
#     Run cleanup (remove redundant .v1.output identical to .output). Can be standalone or
#     after both passes when CROSSPLANE_V1+V2 are set.
#
#   CROSSPLANE_V1 and CROSSPLANE_V2 (paths)
#     Run generate for V1 (.v1.output), then for V2 (.output). Run cleanup only if CLEANUP=true.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EXPECTED_DIR="${SCRIPT_DIR}/../expected"
TESTCASES_FILE="${SCRIPT_DIR}/testcases.sh"
NORMALIZE_SCRIPT="${SCRIPT_DIR}/normalize.sh"
GEN_INVALID_TESTS_SCRIPT="${SCRIPT_DIR}/gen-invalid-tests.sh"
E2E_TESTS_DIR="${PROJECT_ROOT}/examples/mytests/0_e2e"
XPRIN_BIN="${XPRIN_BIN:-${PROJECT_ROOT}/xprin}"
DEFAULT_TIERS="${DEFAULT_TIERS:-v2}"
CROSSPLANE_VERSION="${CROSSPLANE_VERSION:-}"

XPRIN_ARGS=()
if [[ -n "${CROSSPLANE_VERSION}" ]]; then
    [[ "${CROSSPLANE_VERSION}" == v* ]] || CROSSPLANE_VERSION="v${CROSSPLANE_VERSION}"
    XPRIN_ARGS=("--crossplane-version=${CROSSPLANE_VERSION}")
fi

cd "${PROJECT_ROOT}"

# Clean up generated_*.yaml on exit (created by gen-invalid-tests.sh)
trap 'rm -f "${E2E_TESTS_DIR}"/generated_*.yaml' EXIT

# Detect Crossplane CLI tier (v1 / v2legacy / v2) from binary - same logic as run.sh.
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

# Map CLI tier to expected output suffix.
# v1 → .v1.output, v2legacy → .v2legacy.output, v2 → .output (default)
expected_suffix_for_tier() {
    case "$1" in
        v1)       echo ".v1.output" ;;
        v2legacy) echo ".v2legacy.output" ;;
        *)        echo ".output" ;;
    esac
}

run_pass() {
    local suffix="$1"
    local tier="$2"
    echo "Regenerating expected outputs (suffix=${suffix}) into ${EXPECTED_DIR}"
    echo ""
    for test_var in "${TEST_CASES[@]}"; do
        test_id="${test_var#testcase_}"
        test_args="${!test_var}"

        if [ -z "${test_id}" ] || [ -z "${test_args}" ]; then
            echo "Invalid test case entry: ${test_var}"
            exit 1
        fi

        # Run testcase only if the current tier is in testcase_NNN_tiers (fallback: DEFAULT_TIERS).
        tiers_var="${test_var}_tiers"
        if compgen -v | grep -q "^${tiers_var}$"; then
            allowed_tiers="${!tiers_var}"
        else
            allowed_tiers="${DEFAULT_TIERS}"
        fi
        if ! echo " ${allowed_tiers} " | grep -q " ${tier} "; then
            echo "  testcase_${test_id}... SKIP (tier ${tier} not in: ${allowed_tiers})"
            continue
        fi

        echo "  testcase_${test_id}..."
        read -ra cmd_args <<< "${test_args}"

        ACTUAL_OUTPUT="$(mktemp)"
        set +e
        "${XPRIN_BIN}" test "${XPRIN_ARGS[@]}" "${cmd_args[@]}" > "${ACTUAL_OUTPUT}" 2>&1
        set -e

        "${NORMALIZE_SCRIPT}" "${ACTUAL_OUTPUT}" > "${EXPECTED_DIR}/testcase_${test_id}${suffix}"
        rm -f "${ACTUAL_OUTPUT}"
    done
    echo ""
}

run_cleanup() {
    echo "--- Removing redundant tier-specific outputs identical to .output ---"
    removed=0
    for suffix in v1 v2legacy; do
        for tierfile in "${EXPECTED_DIR}"/testcase_*.${suffix}.output; do
            [ -f "${tierfile}" ] || continue
            base="${tierfile%.${suffix}.output}"
            outfile="${base}.output"
            if [ -f "${outfile}" ] && cmp -s "${tierfile}" "${outfile}"; then
                rm -f "${tierfile}"
                echo "  removed $(basename "${tierfile}") (identical to $(basename "${outfile}"))"
                removed=$((removed + 1))
            fi
        done
    done
    [ "${removed}" -eq 0 ] && echo "  none"
    echo ""

    echo "--- Merging identical .v1.output and .v2legacy.output into .v1_v2legacy.output ---"
    merged=0
    for v1file in "${EXPECTED_DIR}"/testcase_*.v1.output; do
        [ -f "${v1file}" ] || continue
        base="${v1file%.v1.output}"
        v2legacyfile="${base}.v2legacy.output"
        if [ -f "${v2legacyfile}" ] && cmp -s "${v1file}" "${v2legacyfile}"; then
            cp "${v1file}" "${base}.v1_v2legacy.output"
            rm -f "${v1file}" "${v2legacyfile}"
            echo "  merged $(basename "${v1file}") + $(basename "${v2legacyfile}") → $(basename "${base}.v1_v2legacy.output")"
            merged=$((merged + 1))
        fi
    done
    [ "${merged}" -eq 0 ] && echo "  none"
    echo ""
}

if [ "${GENERATE:-}" = "true" ] || [ -n "${CROSSPLANE_V1:-}" ] || [ -n "${CROSSPLANE_V2:-}" ]; then
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

    TEST_CASES=($(compgen -v | grep '^testcase_' | grep -v '_exit' | grep -v '_tiers' | LC_ALL=C sort))
    if [ "${#TEST_CASES[@]}" -eq 0 ]; then
        echo "No test cases defined in ${TESTCASES_FILE}"
        exit 1
    fi

    export E2E_TESTS_DIR

    # Generate schema-invalid e2e testsuite files so they are not committed.
    "${GEN_INVALID_TESTS_SCRIPT}"

    if [ -n "${GENERATE:-}" ] && ([ -n "${CROSSPLANE_V1:-}" ] || [ -n "${CROSSPLANE_V2:-}" ]); then
        echo "Error: GENERATE and CROSSPLANE_V1/CROSSPLANE_V2 cannot be set at the same time"
        exit 1
    fi

    # --- Both: CROSSPLANE_V1 and CROSSPLANE_V2 (paths) ---
    if [ -n "${CROSSPLANE_V1:-}" ] && [ -n "${CROSSPLANE_V2:-}" ]; then
        if [ ! -x "${CROSSPLANE_V1}" ]; then
            echo "CROSSPLANE_V1 not executable: ${CROSSPLANE_V1}"
            exit 1
        fi
        if [ ! -x "${CROSSPLANE_V2}" ]; then
            echo "CROSSPLANE_V2 not executable: ${CROSSPLANE_V2}"
            exit 1
        fi
        export PATH="$(dirname "${CROSSPLANE_V1}"):${PATH}"
        which "${CROSSPLANE_V1}"
        ${CROSSPLANE_V1} version --client
        V1_TIER="$(xp_tier_from_binary "${CROSSPLANE_V1}")"
        run_pass "$(expected_suffix_for_tier "${V1_TIER}")" "${V1_TIER}"
        export PATH="$(dirname "${CROSSPLANE_V2}"):${PATH}"
        which "${CROSSPLANE_V2}"
        ${CROSSPLANE_V2} version --client
        V2_TIER="$(xp_tier_from_binary "${CROSSPLANE_V2}")"
        run_pass "$(expected_suffix_for_tier "${V2_TIER}")" "${V2_TIER}"
        echo "Done. Wrote expected file(s) to ${EXPECTED_DIR}."
    fi

    # --- Single: GENERATE=true, crossplane from PATH ---
    if [ "${GENERATE:-}" = "true" ]; then
        CROSSPLANE_BIN="$(command -v crossplane || true)"
        if [ -z "${CROSSPLANE_BIN}" ] || [ ! -x "${CROSSPLANE_BIN}" ]; then
            echo "crossplane not found or not executable on PATH"
            exit 1
        fi
        CUR_TIER="$(xp_tier_from_binary "${CROSSPLANE_BIN}")"
        SUFFIX="$(expected_suffix_for_tier "${CUR_TIER}")"
        export PATH="$(dirname "${CROSSPLANE_BIN}"):${PATH}"
        which crossplane
        crossplane version --client
        run_pass "${SUFFIX}" "${CUR_TIER}"
        echo "Done. Wrote expected file(s) to ${EXPECTED_DIR}."
    fi
fi

if [ "${CLEANUP:-}" = "true" ]; then
    run_cleanup
fi
