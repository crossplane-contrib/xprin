# xprin-helpers

Helper utilities for xprin that provide additional functionality for working with Crossplane resources.

## Overview

xprin-helpers consists of two main tools:

- **[convert-claim-to-xr](xprin-helpers/convert-claim-to-xr.md)**: Convert Crossplane Claims to XRs (Composite Resources)
- **[patch-xr](xprin-helpers/patch-xr.md)**: Apply patches to XRs for enhanced testing scenarios

## Quick Start

Install xprin-helpers (see [Installation](#installation) for all options), then run:

```bash
xprin-helpers convert-claim-to-xr claim.yaml
```

```bash
xprin-helpers patch-xr xr.yaml --xrd=xrd.yaml --add-connection-secret
```

## Tools

### convert-claim-to-xr

Converts Crossplane Claims to XRs so they can be used with `crossplane render`. This is necessary because the `crossplane render` command doesn't support Claims directly.

**Key features:**
- Automatic kind conversion (Claim → XClaim)
- Optional direct XR creation (no Claim references)
- Custom kind support
- Integration with `crossplane render`

[📖 Full Documentation](xprin-helpers/convert-claim-to-xr.md)

### patch-xr

Applies patches to XRs for enhanced testing scenarios, including XRD defaults and connection secret configuration.

**Key features:**
- Apply default values from XRD schemas
- Add connection secret functionality
- Custom connection secret names and namespaces
- Integration with other tools

[📖 Full Documentation](xprin-helpers/patch-xr.md)

## Integration with xprin

These tools are automatically used by xprin when needed:

- **Claim inputs**: Automatically converted using `convert-claim-to-xr`
- **XR patching**: Applied using `patch-xr` when patching flags are specified

## Installation

### Using the install script

The official install script (Unix/macOS) fetches a pre-built binary from [GitHub Releases](https://github.com/crossplane-contrib/xprin/releases). The script detects your OS and architecture and places the binary in the current directory; add it to your `PATH` or move it as needed.

**Recommended: compressed tarball with SHA256 verification**

The following command will install the latest stable version:

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true VERIFY_SHA=true PACKAGE=xprin-helpers sh
```

Install a specific version (e.g. v0.1.1):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true VERIFY_SHA=true PACKAGE=xprin-helpers VERSION=v0.1.1 sh
```

**Other options**

Tarball only (no checksum verification):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | COMPRESSED=true PACKAGE=xprin-helpers sh
```

Binary with SHA256 verification (single binary + `.sha256` file):

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | VERIFY_SHA=true PACKAGE=xprin-helpers sh
```

Binary only:

```bash
curl -sL https://raw.githubusercontent.com/crossplane-contrib/xprin/main/install.sh | PACKAGE=xprin-helpers sh
```

You can override `OS` and `ARCH` if needed (e.g. for cross-installs): add them before `sh` in the command above (e.g. `ARCH=arm64 ... sh`).

### Manual install

If you prefer not to run the install script, download the xprin-helpers binary or tarball for your platform from [GitHub Releases](https://github.com/crossplane-contrib/xprin/releases). Extract if needed, then move the binary to a directory in your `PATH` (e.g. `/usr/local/bin`).

### Using Homebrew

```bash
brew install tampakrap/tap/xprin-helpers
```

### Using Go

Install from source (latest from main):

```bash
go install github.com/crossplane-contrib/xprin/cmd/xprin-helpers@latest
```

Install a specific release version:

```bash
go install github.com/crossplane-contrib/xprin/cmd/xprin-helpers@v0.1.1
```

Or build locally:

```bash
git clone https://github.com/crossplane-contrib/xprin
cd xprin
go build -o xprin-helpers ./cmd/xprin-helpers
```

### Using Earthly

```bash
git clone https://github.com/crossplane-contrib/xprin
cd xprin
earthly +build
```

The built binaries are put under the `_output` directory.

## Verify Installation

After installing xprin-helpers, verify that everything is set up correctly:

```bash
xprin-helpers version
```

## Getting Help

Each tool provides detailed help information:

```bash
xprin-helpers convert-claim-to-xr --help
```

```bash
xprin-helpers patch-xr --help
```
