# Installation

## Prerequisites

- **Crossplane 1.15+**: Required for the `crossplane beta validate` command
- **Docker daemon**: Required for running Composition Functions (alternatives like Podman are also supported)
- **Go 1.24+**: Required for building from source

## Install xprin

### Using Go

```bash
# Install from source
go install github.com/crossplane-contrib/xprin/cmd/xprin@latest

# Or build locally
git clone https://github.com/crossplane-contrib/xprin
cd xprin
go build -o xprin ./cmd/xprin
```

### Using Earthly

```bash
# Clone the repository
git clone https://github.com/crossplane-contrib/xprin
cd xprin

# Build locally
earthly +build
```

The built binaries are put under the `_output` directory.

### Using Homebrew

```bash
brew install tampakrap/tap/xprin
```

## Verify Installation

After installing xprin, verify that everything is set up correctly:

```bash
# Check xprin installation
xprin version

# Verify dependencies and configuration
xprin check
```

The `xprin check` command verifies that:
- Required dependencies (like `crossplane`) are available
- Configuration file (if present) is valid
- Repositories (if configured) are accessible

## xprin-helpers

**Note**: xprin-helpers are used as libraries by xprin and are automatically included when you install xprin. You don't need to install them separately.

If you want to use xprin-helpers as standalone tools or need to build them from source, see the [xprin-helpers documentation](xprin-helpers.md) for detailed installation instructions.

## Optional: Global Configuration

Create a configuration file at `~/.config/xprin.yaml` to specify dependencies and repositories:

```yaml
dependencies:
  crossplane: /usr/local/bin/crossplane

repositories:
  myclaims: /path/to/repos/myclaims
  mycompositions: /path/to/repos/mycompositions

subcommands:
  render: render --include-full-xr
  validate: beta validate --error-on-missing-schemas
```

Validate your configuration:

```bash
# Check dependencies and configuration
xprin check

# Or use the config command (equivalent)
xprin config --check
```

See [Configuration](configuration.md) for detailed configuration options.

---

**Next Steps:**
- If you need custom subcommands (e.g., for older Crossplane versions) or want to use repositories as template variables, see [Configuration](configuration.md)
- Otherwise, continue to [Getting Started](getting-started.md) to run your first test
