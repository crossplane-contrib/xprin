#!/bin/bash
# normalize.sh - Normalizes xprin test output for comparison
#
# Usage: normalize.sh <file>

set -euo pipefail

root="$(pwd)"
# Strip project root prefix so paths become relative (expected output uses relative paths).
# When root is "/", must only strip one leading slash; otherwise "s|/|...|g" would replace every "/".
if [ "$root" = "/" ]; then
    root_sed='s|^/||'
    written_sed='s|    written: /|    written: |'
else
    root_escaped="${root//\//\\/}"
    root_sed="s|^${root_escaped}/||"
    written_sed="s|    written: ${root_escaped}/|    written: |"
fi

sed_args=(
    -E
    -e '/schemas does not exist, downloading:/d'
    -e 's/[0-9]+\.[0-9]+s/X.XXXs/g'
    -e 's|/var/folders/[^/]+/[^/]+/[^/]+/xprin-[^/]+|/tmp/xprin-XXXXX|g'
    -e 's|/tmp/[^/]+/xprin-[^/]+|/tmp/xprin-XXXXX|g'
    -e 's|(/tmp/xprin-testcase-)[0-9]+|\1XXXX|g'
    -e 's|(/tmp/xprin-testsuite-artifacts-)[0-9]+|\1XXXX|g'
    -e 's|(xprin-artifacts-)[0-9]{14}|\1YYYYMMDDHHMMSS|g'
    -e 's|/Users/[^/]+/repos/[^/]+/[^/]+|/Users/user/repos/xprin|g'
    -e "$root_sed"
    -e 's/[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/YYYY\/MM\/DD HH:MM:SS/g'
    -e 's/(--crossplane-version=v)[0-9]+\.[0-9]+\.[0-9]+/\1X.Y.Z/g'
    -e 's/(--crossplane-docker-network=xprin-net-)[0-9a-f]+/\1XXXX/g'
    -e 's|(--crossplane-image=xpkg\.crossplane\.io/crossplane/crossplane:v)[0-9]+\.[0-9]+\.[0-9]+|\1X.Y.Z|g'
    -e "${written_sed}"
    -e 's/\r$//'
    -e 's/[[:space:]]+$//'
)

if [ "$#" -eq 0 ]; then
    sed "${sed_args[@]}" /dev/stdin
else
    sed "${sed_args[@]}" "$@"
fi
