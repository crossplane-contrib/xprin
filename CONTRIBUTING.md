# Contributing to xprin

## Getting Started

**Prerequisites:**
- Go 1.24+
- Docker
- [Earthly](https://earthly.dev/get-earthly)

**Development Setup:**
1. Fork and clone the repo
2. Build: `earthly +build`
3. Run xprin locally: `go run ./cmd/xprin test <testsuite-file>`

## Testing

```shell
# Code generation verification, linting, and unit tests
earthly +reviewable

# Full e2e suite against both Crossplane v1 and v2
# -P (--allow-privileged) is required for Docker-in-Docker
earthly -P +e2e

# E2e tests against a specific Crossplane version
earthly -P +e2e-run --CROSSPLANE_VERSION=$XP_VER
```

If you change example files that affect e2e test output, regenerate the expected files and commit them:

```shell
earthly -P +e2e-regen-expected
```

## PR Process

1. Rebase on `upstream/main`
2. Run `earthly +reviewable` and `earthly -P +e2e`
3. Open a PR with a clear description, issue references, and testing notes
4. Address review feedback promptly

**Commit format:**
```
<type>: <short description>

<longer description if needed>

Fixes #issue-number

Signed-off-by: Your Name <your@email.com>
```
Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

## Getting Help

- GitHub Issues: https://github.com/crossplane-contrib/xprin/issues
