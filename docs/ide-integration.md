# IDE integration (JSON Schema)

A [JSON Schema](https://json-schema.org/) for the xprin test suite format is provided so editors can offer autocompletion, validation, and hover documentation for `xprin.yaml` and `*_xprin.yaml` files.

**Schema file:** A single combined schema is provided at `data/xprin-testsuite.json` (in the xprin repository root). It is generated from the Go types in `internal/api` using [invopop/jsonschema](https://github.com/invopop/jsonschema). After changing the API types, regenerate with `earthly +generate` or `earthly +reviewable`. See [Test suite specification](testsuite-specification.md) for the full format.

## Editor setup

You can apply the setup below in this repo as well (e.g. create `.vscode/settings.json` to check the schema support on the provided [examples](../examples/)).

1. **VS Code IDEs (IntelliSense):** Add to your user or workspace `settings.json` (e.g. in the repo create `.vscode/settings.json`; this file is not committed):
   ```json
   "yaml.schemas": {
     "https://raw.githubusercontent.com/crossplane-contrib/xprin/main/data/xprin-testsuite.json": ["*_xprin.yaml", "xprin.yaml"]
   }
   ```
   or point to a local path: `"/path/to/xprin/data/xprin-testsuite.json": ["*_xprin.yaml", "xprin.yaml"]`.

2. **JetBrains IDEs:** In **Settings → Languages & Frameworks → Schemas and Dictionaries**, add a schema and set the URL or file path; then add a mapping for `*_xprin.yaml` and `xprin.yaml`.

3. **Per-file (any YAML-capable editor):** Add at the top of the test suite YAML file:
   ```yaml
   # yaml-language-server: $schema=https://raw.githubusercontent.com/crossplane-contrib/xprin/main/data/xprin-testsuite.json
   ```
   or a local path like `$schema=./data/xprin-testsuite.json` if the schema lives next to your file.
