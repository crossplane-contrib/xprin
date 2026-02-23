# Documentation

This directory contains comprehensive documentation for xprin and its related tools.

## Documentation Flow

Follow this path for a complete learning journey:

1. **[Installation](installation.md)** - Install xprin and verify your setup
   - Optional [Configuration](configuration.md) if you need custom subcommands or repositories
   - Optional [IDE integration](ide-integration.md) for autocompletion and validation in your editor
2. **[Getting Started](getting-started.md)** - Run your first test and learn basic commands
3. **[Examples](../examples/README.md)** - Step-by-step examples with real outputs
4. **[Test Suite Specification](testsuite-specification.md)** - Complete reference for all test suite fields and options
5. **[Assertions](assertions.md)** - Complete guide to declarative resource validation
6. **[How It Works](how-it-works.md)** - Deep dive into how xprin works under the hood
7. **[xprin-helpers](xprin-helpers.md)** - Helper utilities overview
   - [convert-claim-to-xr](xprin-helpers/convert-claim-to-xr.md) - Convert Claims to XRs
   - [patch-xr](xprin-helpers/patch-xr.md) - Apply patches to XRs

## Quick Reference

### Main Commands

```bash
# Test Compositions
xprin test <targets>

# Check dependencies and configuration
xprin check

# Show version
xprin version
```

### Helper Tools

```bash
# Convert Claims to XRs
xprin-helpers convert-claim-to-xr <claim-file>

# Patch XRs
xprin-helpers patch-xr <xr-file> [options]
```

## Key Concepts

- **Test Suite Files**: YAML files named `xprin.yaml` or `*_xprin.yaml`
- **XR vs Claim Inputs**: XRs are used directly, Claims are converted to XRs
- **Template Variables**: Use `{{ .Repositories.name }}` for dynamic paths
- **Hooks**: Execute shell commands before and after tests
- **XR Patching**: Apply defaults and connection secrets to XRs
- **Assertions**: Declarative validation of rendered resources (count, existence, field type/value checks)
