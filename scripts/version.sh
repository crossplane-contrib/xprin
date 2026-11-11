#!/usr/bin/env bash
# version.sh - prints the version to embed in the xprin binaries.
#
# If XPRIN_VERSION is set (e.g. by the release workflow) it is printed verbatim.
# Otherwise the version is derived from the git repository.
# - On detached HEAD or non-release branches: v<MAJOR>.<MINOR+1>.0-pre.<commit timestamp>.g<short hash>[-dirty]
# - On a release branch (not detached):       v<MAJOR>.<MINOR>.<PATCH+1>-pre.<commit timestamp>.g<short hash>[-dirty]

set -euo pipefail

if [ -n "${XPRIN_VERSION:-}" ]; then
  printf '%s' "$XPRIN_VERSION"
  exit 0
fi

current_branch=$(git branch --show-current)

if [[ "$current_branch" =~ ^release-([0-9]+)\.([0-9]+)$ ]]; then
  # On a release branch: next patch of that series
  series="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
  last=$(git tag --list "v${series}.*" | grep -E "^v${series//./\\.}\.[0-9]+$" | sort -V | tail -n 1 || true)
  if [ -n "$last" ]; then
    base="v${series}.$((${last##*.} + 1))"
  else
    base="v${series}.0"
  fi
else
  # On a non-release branch: next minor of the latest release series
  latest=$(git for-each-ref --format='%(refname:lstrip=2)' 'refs/heads/release-*' | grep -E '^release-[0-9]\.[0-9]+$' | sed 's/^release-//' | sort -V | tail -n 1 || true)

  if [ -n "$latest" ]; then
    base="v${latest%%.*}.$((${latest#*.} + 1)).0"
  else
    base="v0.0.0"
  fi
fi

# Ignore all tags so that git describe prints just <short hash>[-dirty].
commit=$(git describe --always --dirty --exclude='*')

printf '%s-pre.%s.g%s' "$base" "$(git show -s --format=%ct HEAD)" "$commit"
