<!--
Ownership: curated terminal help for `m help runner`.
Authoritative: docs/runner.md
-->

# Runner

## Surfaces

| Command | Role |
|---|---|
| `m run <script>` | Run a `package.json` script |
| `m exec <bin>` | Execute a local package binary |
| `mx` / `mewx` | Package executable runner (local or temporary) |

Direct `m <script>` shortcuts are an optional extension and require the direct-scripts gate.

## Sources

- Project `node_modules` / package scripts
- Temporary `mx` environments
- Snapshot / capsule restore paths where applicable

## Behavior notes

- Child stdout/stderr ownership follows the runner contract.
- Signals and exit codes are preserved from the child when possible.
- Offline / no-network guarantees apply only where documented for the specific path.

## Examples

```text
m run build
m run test -- --watch
m exec eslint .
mx cowsay hello
```

## See also

- docs/runner.md
- docs/cli.md
- `m help configuration`
