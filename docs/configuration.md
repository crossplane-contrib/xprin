# Configuration

`xprin` supports an optional global configuration file to specify dependencies, repositories, and subcommand settings.

## Configuration File Location

- **Default**: `~/.config/xprin.yaml`
- **Custom**: Use `-c` flag to specify a different location

```bash
xprin -c /path/to/yourconfig.yaml test tests/
```

## Configuration Schema

### Dependencies

Required map of dependency names to binary paths:

```yaml
dependencies:
  crossplane: /usr/local/bin/crossplane  # Absolute path
```

- The only supported dependency currently is `crossplane`.
- The value of a dependency can be either an absolute path or just the command name that is in `$PATH`.

### Repositories

Optional map of repository names to local paths:

```yaml
repositories:
  myclaims: /path/to/repos/myclaims
  mycompositions: /path/to/repos/mycompositions
```

Repository keys must match either:
- Directory name
- Directory name with `.git` suffix
- Remote URL

Used for resolving template variables in test suite files, for example `{{ .Repositories.myclaims }}`.

### Subcommands

xprin probes the configured `crossplane` binary at startup and adapts its behavior automatically. No explicit configuration is needed in most cases. Run `xprin check` to see which capabilities were detected for the configured `crossplane` binary.

#### Validate subcommand

The validate subcommand is auto-detected based on the CLI version:

| CLI versions | Validate subcommand used |
|-----|--------------------------|
| ≥ v2.3.0 | `resource validate --error-on-missing-schemas` |
| v2.0–v2.2 | `beta validate --error-on-missing-schemas` |
| v1.x | `beta validate --error-on-missing-schemas` |

Detection is behavior-based (probes `crossplane resource validate --help`), so custom and nightly builds are handled correctly.

#### Render and `--xrd`

The `--xrd` flag for `crossplane render` is also auto-detected. When a test case sets `patches.xrd`, xprin passes the XRD path as `--xrd` to `crossplane render` only on CLIs that support it:

| CLI | `patches.xrd` behaviour |
|-----|-------------------------|
| `crossplane/cli` v2.0+ | XRD defaults applied via `xprin-helpers patch-xr` **and** `--xrd` passed to `crossplane render` |
| `crossplane/crossplane` v1.x | XRD defaults applied via `xprin-helpers patch-xr` only (`--xrd` not supported by this CLI) |

Additionally, passing `--xrd` to `crossplane render` matters in v2+ for **LegacyCluster XRDs**. Without it, the v2 render engine defaults to Modern schema, placing `resourceRefs` at `spec.crossplane.resourceRefs` instead of `spec.resourceRefs`. This causes validation to fail against the XRD schema. Providing `--xrd` lets the engine detect the correct scope and produce output that matches the XRD.

#### Overriding auto-detection

If needed, the render and validate subcommands can be set explicitly in the config file. Explicit values always take precedence over auto-detection:

```yaml
subcommands:
  render: render --include-full-xr
  validate: resource validate --error-on-missing-schemas
```

## Example Configuration

```yaml
dependencies:
  crossplane: /usr/local/bin/crossplane

repositories:
  myclaims: /path/to/repos/myclaims
  mycompositions: /path/to/repos/mycompositions

subcommands:
  render: render --include-full-xr
  validate: resource validate --error-on-missing-schemas
```

## Validation

Check your configuration:

```bash
# Display raw configuration file contents (no path resolution)
# If no config file is found, reports "No configuration file provided."
xprin config

# Validate configuration and dependencies
xprin check

# Or use the config command (equivalent)
xprin config --check
```

Both `xprin check` and `xprin config --check` verify that:
- All dependencies are found and executable
- All repositories exist and are accessible
- Configuration syntax is valid

---

**Next Steps:**
- Continue to [Getting Started](getting-started.md) to run your first test
