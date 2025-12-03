# convert-claim-to-xr

Converts a Crossplane Claim YAML file to a Crossplane XR (Composite Resource) format.

## Why This Tool?

Claims are not supported by the `crossplane render` command. This tool bridges that gap by converting Claims to XRs, allowing you to test Compositions with Claim inputs.

## Installation

See [Installation](../xprin-helpers.md#installation).
 
## Command Options

| Option | Description |
|--------|-------------|
| `--kind=KIND` | Custom kind for the XR (default: "X" + Claim kind) |
| `--direct` | Create direct XR without Claim references |
| `-o, --output-file=PATH` | Output file (default: stdout) |
| `--version` | Print version information |

## Default Conversion Behavior

The converter by default assumes that the produced XR derives from a Claim, thus it will:
- Set a random suffix in `.metadata.name`
- Set the `kind`'s value to the same as the Claim's prefixed by an "X"
- Set [the appropriate labels](https://docs.crossplane.io/v1.20/concepts/composite-resources/#composite-resource-labels)
- Set `.spec.claimRef`

The last two show the relation between the Claim and the XR.

## Examples

```bash
# Convert claim.yaml to XR format and write to stdout (kind will be 'X' + Claim's kind)
xprin-helpers convert-claim-to-xr claim.yaml

# Convert claim.yaml to XR format with a specific kind
xprin-helpers convert-claim-to-xr claim.yaml --kind MyCompositeResource

# Convert claim.yaml to XR format and write to xr.yaml
xprin-helpers convert-claim-to-xr claim.yaml -o xr.yaml

# Convert claim.yaml to a directly created XR (no Claim references, no name suffix)
xprin-helpers convert-claim-to-xr claim.yaml --direct

# Convert Claim from stdin to XR format
cat claim.yaml | xprin-helpers convert-claim-to-xr -

# Show detailed help
xprin-helpers convert-claim-to-xr --help
```

## Integration with other tools
### crossplane render

```bash
crossplane render <(xprin-helpers convert-claim-to-xr claim.yaml) composition.yaml functions.yaml
```

### xprin

This tool is automatically used by `xprin` when testing with Claim inputs, which can be found in the debug output:

```yaml
# tests/claim_to_xr_example_xprin.yaml
tests:
- name: "Claim to XR"
  inputs:
    claim: claim.yaml
    composition: composition.yaml
    functions: functions.yaml
```

```bash
xprin test tests/claim_to_xr_example_xprin.yaml --debug
```
