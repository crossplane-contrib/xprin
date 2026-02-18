# Installation

## Prerequisites

- **Crossplane 1.15+**: Required for the `crossplane beta validate` command
- **Docker daemon**: Required for running Composition Functions (alternatives like Podman are also supported)
- **Go 1.24+**: Required for building from source

## Install xprin

### Using the install script

Download and run the official install script (Unix/macOS). It fetches a pre-built binary from [GitHub Releases](https://github.com/crossplane-contrib/xprin/releases).

**Recommended: compressed tarball with SHA256 verification**

Download the `.tar.gz` bundle and verify both the archive and the extracted binary using the release `.sha256` files. The script removes the tarball and hash files and keeps only the binary.

```bash
# Install latest version
COMPRESSED=true VERIFY_SHA=true curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh

# Install a specific version (e.g. v0.1.0)
COMPRESSED=true VERIFY_SHA=true VERSION=v0.1.0 curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh
```

**Alternatives**

- **Tarball only** (no verification): `COMPRESSED=true`
- **Binary with SHA256 verification**: `VERIFY_SHA=true` (downloads the binary and its `.sha256`, verifies, then keeps only the binary)
- **Binary only** (no verification): default; single binary download

```bash
# Tarball only
COMPRESSED=true curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh

# Binary with verification
VERIFY_SHA=true curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh

# Binary only (default)
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh
```

Then move the binary to your PATH:

```bash
sudo mv xprin /usr/local/bin/
xprin version
```

To install **xprin-helpers** instead, set `PACKAGE=xprin-helpers`:

```bash
PACKAGE=xprin-helpers curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh
sudo mv xprin-helpers /usr/local/bin/
```

You can override `OS` and `ARCH` if needed (e.g. for cross-installs).

### Using Homebrew

```bash
brew install tampakrap/tap/xprin
```

### Using Go

```bash
# Install from source (latest from main)
go install github.com/crossplane-contrib/xprin/cmd/xprin@latest

# Install a specific release version
go install github.com/crossplane-contrib/xprin/cmd/xprin@v0.1.0

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

## Optional: Editor / IDE support

To get autocompletion and validation for test suite YAML files (`xprin.yaml`, `*_xprin.yaml`) in VS Code, Cursor, or JetBrains, see [IDE integration](ide-integration.md).

---

**Next Steps:**
- If you need custom subcommands (e.g., for older Crossplane versions) or want to use repositories as template variables, see [Configuration](configuration.md)
- Otherwise, continue to [Getting Started](getting-started.md) to run your first test
