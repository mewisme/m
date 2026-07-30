<!--
Ownership: curated terminal help for `m help compatibility`.
Authoritative: docs/compatibility-axes.md
-->

# Compatibility

Mew evaluates compatibility on independent axes. Support on one axis does not imply support on another.

## Axes

| Axis | Scope |
|---|---|
| CLI | Commands, flags, help, dispatch, exit codes |
| Lockfile | Read / write / validate / migrate per format |
| Config | Project/global config, identity detection, env |
| Runtime | Node launch, loaders, transforms |
| Layout | `node_modules` structure, store, linking |

## Important claims

- Parsing a lockfile is not the same as semantic rewrite support.
- Frozen validation by Mew is not the same as the external tool accepting the file.
- Installed-graph checks require more than lock-only commands.
- Do not assume the latest package-manager major represents older projects.

## Examples

```text
m features
m lock validate
m doctor
```

## See also

- docs/compatibility-axes.md
- docs/charter.md
