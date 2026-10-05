# convert-claim-to-xr

Converts a Crossplane Claim YAML file to a Crossplane XR (Composite Resource) format.

## Why This Tool?

Claims are not supported by the `crossplane render` command. This tool bridges that gap by converting Claims to XRs, allowing you to test Compositions with Claim inputs.

## Installation

See [Installation](../xprin-helpers.md#installation).
 
## Command Options

| Option | Description |
|--------|-------------|
| `--name=NAME` | Custom name for the XR. Overrides the default behavior (Claim name in direct mode, Claim name + random suffix in non-direct mode). Mutually exclusive with `--golden` |
| `--kind=KIND` | Custom kind for the XR. Mutually exclusive with `--xrd` (default if neither is set: "X" + Claim kind) |
| `--xrd=PATH` | YAML file specifying the CompositeResourceDefinition (XRD) that owns the Claim, used to resolve the XR's real kind. Mutually exclusive with `--kind` |
| `--golden=PATH` | A previous render/golden YAML file. If it has a document of the resolved XR kind, its name is reused instead of a fresh random suffix, making the output reproducible across runs. Mutually exclusive with `--name` |
| `--direct` | Create direct XR without Claim references |
| `--gen-uid` | Set a fresh random `metadata.uid` on the generated XR |
| `-o, --output-file=PATH` | Output file (default: stdout) |

## Default Conversion Behavior

The converter by default assumes that the produced XR derives from a Claim, thus it will:
- Set a random suffix in `.metadata.name` (unless `--name` or `--golden` are given)
- Set the `kind`'s value to the same as the Claim's prefixed by an "X" (unless `--kind` or `--xrd` are set)
- Set [the appropriate labels](https://docs.crossplane.io/v1.20/concepts/composite-resources/#composite-resource-labels)
- Set `.spec.claimRef`

The last two show the relation between the Claim and the XR.

## Examples

```bash
# Convert claim.yaml to XR format and write to stdout (kind will be 'X' + Claim's kind)
xprin-helpers convert-claim-to-xr claim.yaml

# Convert claim.yaml to XR format and write to xr.yaml
xprin-helpers convert-claim-to-xr claim.yaml -o xr.yaml

# Convert claim.yaml using an explicit XR name (overrides the default suffix or claim name)
xprin-helpers convert-claim-to-xr claim.yaml --name my-xr

# Convert claim.yaml and create a reproducible XR by setting the XR's name based on a previous golden file of a render output, or just an XR output
xprin-helpers convert-claim-to-xr claim.yaml --golden render_output.yaml

# Convert claim.yaml to XR format with a specific kind
xprin-helpers convert-claim-to-xr claim.yaml --kind MyCompositeResource

# Convert claim.yaml with the kind resolved from its XRD
xprin-helpers convert-claim-to-xr claim.yaml --xrd xrd.yaml

# Convert claim.yaml to a directly created XR (no Claim references, no name suffix)
xprin-helpers convert-claim-to-xr claim.yaml --direct

# Convert claim.yaml and assign a fresh random metadata.uid to the XR
xprin-helpers convert-claim-to-xr claim.yaml --gen-uid

# Convert Claim from stdin to XR format
cat claim.yaml | xprin-helpers convert-claim-to-xr -

# Convert claim.yaml, reusing the XR name from a previous render if one matches
# (falls back to a random suffix if golden_render.yaml doesn't exist yet, or has no matching document)
xprin-helpers convert-claim-to-xr claim.yaml --golden golden_render.yaml

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

If the test case also sets `patches.xrd`, xprin passes it through to the conversion step automatically, so the generated XR's kind is resolved from the XRD instead of the "X" + Claim kind guess:

```yaml
# tests/claim_to_xr_with_xrd_xprin.yaml
tests:
- name: "Claim to XR with XRD-resolved kind"
  patches:
    xrd: xrd.yaml
  inputs:
    claim: claim.yaml
    composition: composition.yaml
    functions: functions.yaml
```
