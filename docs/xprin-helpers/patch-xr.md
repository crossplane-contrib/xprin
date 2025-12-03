# patch-xr

Applies patches to Crossplane XRs (Composite Resources) for enhanced testing scenarios.

Available features:
- **XRD Defaults**: Apply default values from CompositeResourceDefinition schemas
  - This feature is extracted from the command `crossplane render --xrd` that is available in Crossplane CLI v2
- **Connection Secret**: Add `writeConnectionSecretToRef` to XR spec

## Installation

See [Installation](../xprin-helpers.md#installation).

## Command Options

| Option | Description |
|--------|-------------|
| `--xrd=PATH` | Path to XRD file for default values |
| `--add-connection-secret` | Enable connection secret functionality |
| `--connection-secret-name=NAME` | Custom connection secret name |
| `--connection-secret-namespace=NS` | Custom connection secret namespace |
| `-o, --output-file=PATH` | Output file (default: stdout) |

## Features

### XRD Defaults

The tool can populate missing XR fields with default values from XRD schemas:

```bash
# Apply XRD defaults
xprin-helpers patch-xr xr.yaml --xrd=database-xrd.yaml
```

This feature is extracted from the `crossplane render --xrd` flag that is available in Crossplane CLI v2, so that it is available to users of earlier Crossplane versions.

### Connection Secret Patching

Add connection secret functionality to XRs:

```bash
# Basic connection secret
xprin-helpers patch-xr xr.yaml --add-connection-secret

# Custom name and namespace
xprin-helpers patch-xr xr.yaml --add-connection-secret --connection-secret-name=my-secret --connection-secret-namespace=my-namespace
```

**Important**: Connection secret must be explicitly enabled with `--add-connection-secret` or `--add-connection-secret=true`.

## Examples

```bash
# Apply default values from an XRD
xprin-helpers patch-xr xr.yaml --xrd=composite-resource-definition.yaml

# Add connection secret to the XR
xprin-helpers patch-xr xr.yaml --add-connection-secret

# Add connection secret with custom name and namespace (requires explicit --add-connection-secret=true)
xprin-helpers patch-xr xr.yaml --add-connection-secret=true --connection-secret-name=my-secret --connection-secret-namespace=my-namespace

# Add connection secret with just custom name (requires explicit --add-connection-secret=true)
xprin-helpers patch-xr xr.yaml --add-connection-secret=true --connection-secret-name=my-secret

# Combine patching flags
xprin-helpers patch-xr xr.yaml --add-connection-secret --xrd=xrd.yaml

# Patch XR from stdin
cat xr.yaml | xprin-helpers patch-xr - --add-connection-secret

# Show detailed help
xprin-helpers patch-xr --help
```

## Integration with other tools
### crossplane render

```bash
crossplane render <(xprin-helpers patch-xr xr.yaml --add-connection-secret --xrd=xrd.yaml) composition.yaml functions.yaml
```

It can also be used with an XR converted from a Claim:

```bash
crossplane render <(xprin-helpers convert-claim-to-xr claim.yaml | xprin-helpers patch-xr - --add-connection-secret --xrd=xrd.yaml) composition.yaml functions.yaml
```

### xprin

This tool is automatically used by `xprin` when `patches` are specified in a testcase.

```yaml
# tests/patch_xr_example_xprin.yaml
tests:
- name: "Patch XR"
  patches:
    connection-secret: true
    xrd: xrd.yaml
  inputs:
    xr: xr.yaml
    composition: composition.yaml
    functions: functions.yaml
```

```bash
xprin test tests/patch_xr_example.yaml --debug
```

In case that `inputs.claim` is specified in the testcase, then `xprin` will first convert the Claim to XR and then patch the generated XR.
