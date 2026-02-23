# Installation

## Prerequisites

- **Crossplane 1.15+**: Required for the `crossplane beta validate` command
- **Docker daemon**: Required for running Composition Functions (alternatives like Podman are also supported)
- **Go 1.24+**: Optional, only if building from source

## Install xprin

### Using the install script

The official install script (Unix/macOS) fetches a pre-built binary from [GitHub Releases](https://github.com/crossplane-contrib/xprin/releases). The script detects your OS and architecture and places the `xprin` binary in the current directory; add it to your `PATH` or move it as needed.

**Recommended: compressed tarball with SHA256 verification**

The following command will install the latest stable version:

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true VERIFY_SHA=true sh
```

Install a specific version (e.g. v0.1.1):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true VERIFY_SHA=true VERSION=v0.1.1 sh
```

**Other options**

Tarball only (no checksum verification):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true sh
```

Binary with SHA256 verification (single binary + `.sha256` file):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | VERIFY_SHA=true sh
```

Binary only:

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | sh
```

To install **xprin-helpers** instead of xprin:

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | PACKAGE=xprin-helpers sh
```

You can override `OS` and `ARCH` if needed (e.g. for cross-installs): add them before `sh` in the command above (e.g. `ARCH=arm64 ... sh`).

### Manual install

If you prefer not to run the install script, download a binary or tarball for your platform from [GitHub Releases](https://github.com/crossplane-contrib/xprin/releases). Extract if needed, then move the binary to a directory in your `PATH` (e.g. `/usr/local/bin`).

### Using Homebrew

```bash
brew install tampakrap/tap/xprin
```

### Using Go

Install from source (latest from main):

```bash
go install github.com/crossplane-contrib/xprin/cmd/xprin@latest
```

Install a specific release version:

```bash
go install github.com/crossplane-contrib/xprin/cmd/xprin@v0.1.1
```

Or build locally:

```bash
git clone https://github.com/crossplane-contrib/xprin
cd xprin
go build -o xprin ./cmd/xprin
```

### Using Earthly

```bash
git clone https://github.com/crossplane-contrib/xprin
cd xprin
earthly +build
```

The built binaries are put under the `_output` directory.

## Verify Installation

After installing xprin, verify that everything is set up correctly:

```bash
xprin version
```

```bash
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
xprin check
```

```bash
xprin config --check
```

See [Configuration](configuration.md) for detailed configuration options.

## Optional: Editor / IDE support

To get autocompletion and validation for test suite YAML files (`xprin.yaml`, `*_xprin.yaml`) in VS Code, Cursor, or JetBrains, see [IDE integration](ide-integration.md).

---

**Next Steps:**
- If you need custom subcommands (e.g., for older Crossplane versions) or want to use repositories as template variables, see [Configuration](configuration.md)
- If you need autocompletion and validation of test suite YAML in your favorite IDE, see [IDE integration](ide-integration.md)
- Otherwise, continue to [Getting Started](getting-started.md) to run your first test
