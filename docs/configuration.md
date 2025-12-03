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

Optional map defining render and validate subcommands:

```yaml
subcommands:
  render: render --include-full-xr
  validate: beta validate --error-on-missing-schemas
```

This allows compatibility with different Crossplane CLI versions.

## Example Configuration

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

## Validation

Check your configuration:

```bash
# Display current configuration
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
