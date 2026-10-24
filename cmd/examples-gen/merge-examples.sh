#!/bin/bash
# Merge examples/README.md outputs of the Crossplane CLI v2 and v1 (run inside the container of the
# regen-examples target, see docs/contributing/examples-gen.md)
#
# In the working directory:
#   README.md     the file before the regeneration
#   README.v2.md  the file with the untagged blocks regenerated
#   README.v1.md  the file with the blocks tagged v1 regenerated
#
# Each of them changed other blocks of README.md, so a three-way merge puts the changes together. The result goes to
# README.md, and the script fails when the changes overlap.

set -euo pipefail

merged="$(mktemp)"

if ! git merge-file -p README.v2.md README.md README.v1.md > "${merged}"; then
    echo "merge-examples.sh: README.v2.md and README.v1.md changed the same lines of README.md"
    exit 1
fi

mv "${merged}" README.md
