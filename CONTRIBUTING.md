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

Before touching the code, read the [How It Works](docs/how-it-works.md) and [Architecture](docs/architecture.md): it shows the flow of a run, and the packages and components behind each step.

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

The outputs of the examples in `examples/README.md` are generated too, see [examples-gen](docs/contributing/examples-gen.md) to regenerate them.

The JSON schema (`data/xprin-testsuite.json`) and the tables, the index and the legend of the emojis of the [test suite specification](docs/testsuite-specification.md) are generated from the types of `internal/api`: after changing them, run `earthly +generate` (`earthly +reviewable` does it too). The comment of a field is its description, and `schema-gen` adds a note at the end of it that says if the field is required, so don't write one, see [IDE integration](docs/ide-integration.md).

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
