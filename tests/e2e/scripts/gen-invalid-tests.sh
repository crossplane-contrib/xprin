#!/bin/bash
# gen-invalid-tests.sh - Generates schema-invalid e2e testsuite files so they are not committed.
# Files are created with a generated_ prefix so runners can remove them with rm ${E2E_TESTS_DIR}/generated_*.yaml.
# Expects E2E_TESTS_DIR to be set, or defaults to PROJECT_ROOT/examples/mytests/0_e2e using script location.

set -euo pipefail

if [ -z "${E2E_TESTS_DIR:-}" ]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
    E2E_TESTS_DIR="${PROJECT_ROOT}/examples/mytests/0_e2e"
fi

printf '%s\n' '- hello' > "${E2E_TESTS_DIR}/generated_invalid_xprin.yaml"
printf 'tests:\n- name: "Missing required inputs (multiline error)"\n' > "${E2E_TESTS_DIR}/generated_missing_required_inputs_xprin.yaml"
