#!/bin/bash
# regen-expected.sh - Regenerate e2e expected output files (run inside container)
#
# Usage: regen-expected.sh [v1|v2|cleanup|all]
#   No arg or "all": run v1 pass, v2 pass, then remove redundant .v1.output (needs CROSSPLANE_V1 and CROSSPLANE_V2).
#   v1: run only v1 pass (needs CROSSPLANE_V1). No cleanup.
#   v2: run only v2 pass (needs CROSSPLANE_V2). No cleanup.
#   cleanup: only remove .v1.output files identical to .output (no crossplane needed).
#
# Used by: earthly +regen-e2e-expected

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EXPECTED_DIR="${SCRIPT_DIR}/../expected"
TESTCASES_FILE="${SCRIPT_DIR}/testcases.sh"
NORMALIZE_SCRIPT="${SCRIPT_DIR}/normalize.sh"
XPRIN_BIN="${XPRIN_BIN:-${PROJECT_ROOT}/xprin}"

# Parse mode: v1 | v2 | cleanup | all (default)
MODE="${1:-all}"
case "${MODE}" in
    v1|v2|cleanup|all) ;;
    *)
        echo "Usage: $0 [v1|v2|cleanup|all]"
        echo "  No arg or 'all': run v1 and v2 passes, then cleanup."
        echo "  v1: run only v1 pass (no cleanup)."
        echo "  v2: run only v2 pass (no cleanup)."
        echo "  cleanup: only remove redundant .v1.output files."
        exit 1
        ;;
esac

cd "${PROJECT_ROOT}"

if [ ! -f "${TESTCASES_FILE}" ]; then
    echo "Test case list not found: ${TESTCASES_FILE}"
    exit 1
fi

if [ "${MODE}" != "cleanup" ] && [ ! -x "${XPRIN_BIN}" ]; then
    echo "xprin binary not found or not executable: ${XPRIN_BIN}"
    exit 1
fi

if [ ! -f "${NORMALIZE_SCRIPT}" ]; then
    echo "Normalize script not found: ${NORMALIZE_SCRIPT}"
    exit 1
fi

# shellcheck source=/dev/null
source "${TESTCASES_FILE}"

TEST_CASES=($(compgen -v | grep '^testcase_' | grep -v '_exit$' | LC_ALL=C sort))

if [ "${#TEST_CASES[@]}" -eq 0 ]; then
    echo "No test cases defined in ${TESTCASES_FILE}"
    exit 1
fi

# Require crossplane binaries only for the modes that use them
if [ "${MODE}" = "v1" ] || [ "${MODE}" = "all" ]; then
    if [ -z "${CROSSPLANE_V1:-}" ]; then
        echo "CROSSPLANE_V1 (path to crossplane binary) is required for mode: ${MODE}"
        exit 1
    fi
    if [ ! -x "${CROSSPLANE_V1}" ]; then
        echo "CROSSPLANE_V1 not executable: ${CROSSPLANE_V1}"
        exit 1
    fi
fi
if [ "${MODE}" = "v2" ] || [ "${MODE}" = "all" ]; then
    if [ -z "${CROSSPLANE_V2:-}" ]; then
        echo "CROSSPLANE_V2 (path to crossplane binary) is required for mode: ${MODE}"
        exit 1
    fi
    if [ ! -x "${CROSSPLANE_V2}" ]; then
        echo "CROSSPLANE_V2 not executable: ${CROSSPLANE_V2}"
        exit 1
    fi
fi

# Run one pass: PATH is set so crossplane is the binary for this pass; suffix is .v1.output or .output.
run_pass() {
    local suffix="$1"
    echo "Regenerating expected outputs (suffix=${suffix}) into ${EXPECTED_DIR}"
    echo ""
    for test_var in "${TEST_CASES[@]}"; do
        test_id="${test_var#testcase_}"
        test_args="${!test_var}"

        if [ -z "${test_id}" ] || [ -z "${test_args}" ]; then
            echo "Invalid test case entry: ${test_var}"
            exit 1
        fi

        echo "  testcase_${test_id}..."
        read -ra cmd_args <<< "${test_args}"

        ACTUAL_OUTPUT="$(mktemp)"
        set +e
        "${XPRIN_BIN}" test "${cmd_args[@]}" > "${ACTUAL_OUTPUT}" 2>&1
        set -e

        "${NORMALIZE_SCRIPT}" "${ACTUAL_OUTPUT}" > "${EXPECTED_DIR}/testcase_${test_id}${suffix}"
        rm -f "${ACTUAL_OUTPUT}"
    done
    echo ""
}

run_cleanup() {
    echo "--- Removing redundant .v1.output (identical to .output) ---"
    removed=0
    for v1file in "${EXPECTED_DIR}"/testcase_*.v1.output; do
        [ -f "${v1file}" ] || continue
        base="${v1file%.v1.output}"
        outfile="${base}.output"
        if [ -f "${outfile}" ] && cmp -s "${v1file}" "${outfile}"; then
            rm -f "${v1file}"
            echo "  removed $(basename "${v1file}") (identical to $(basename "${outfile}"))"
            removed=$((removed + 1))
        fi
    done
    [ "${removed}" -eq 0 ] && echo "  none"
    echo ""
}

if [ "${MODE}" = "v1" ] || [ "${MODE}" = "all" ]; then
    export PATH="$(dirname "${CROSSPLANE_V1}"):${PATH}"
    which "${CROSSPLANE_V1}"
    ${CROSSPLANE_V1} version --client
    run_pass ".v1.output"
fi

if [ "${MODE}" = "v2" ] || [ "${MODE}" = "all" ]; then
    export PATH="$(dirname "${CROSSPLANE_V2}"):${PATH}"
    which "${CROSSPLANE_V2}"
    ${CROSSPLANE_V2} version --client
    run_pass ".output"
fi

# Cleanup only when we have or had both sets (all) or when explicitly requested (cleanup)
if [ "${MODE}" = "all" ] || [ "${MODE}" = "cleanup" ]; then
    run_cleanup
fi

echo "Done. Wrote expected file(s) to ${EXPECTED_DIR}."
