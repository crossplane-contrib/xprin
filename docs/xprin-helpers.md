# xprin-helpers

Helper utilities for xprin that provide additional functionality for working with Crossplane resources.

## Overview

xprin-helpers consists of two main tools:

- **[convert-claim-to-xr](xprin-helpers/convert-claim-to-xr.md)**: Convert Crossplane Claims to XRs (Composite Resources)
- **[patch-xr](xprin-helpers/patch-xr.md)**: Apply patches to XRs for enhanced testing scenarios

## Quick Start

```bash
# Install xprin-helpers
go install github.com/crossplane-contrib/xprin/cmd/xprin-helpers@latest

# Convert a Claim to XR
xprin-helpers convert-claim-to-xr claim.yaml

# Patch an XR with defaults and connection secret
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

### Using Go

```bash
# Install from source
go install github.com/crossplane-contrib/xprin/cmd/xprin-helpers@latest

# Or build locally
git clone https://github.com/crossplane-contrib/xprin
cd xprin
go build -o xprin-helpers ./cmd/xprin-helpers
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
brew install tampakrap/tap/xprin-helpers
```

## Verify Installation

```bash
# Check xprin-helpers installation
xprin-helpers version
```

## Getting Help

Each tool provides detailed help information:

```bash
xprin-helpers convert-claim-to-xr --help
xprin-helpers patch-xr --help
```