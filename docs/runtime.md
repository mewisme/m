# Node runtime (MVP 0050)

Mew can run `.js`, `.mjs`, and `.cjs` files directly through the `m` CLI. This
is the foundation for TypeScript/JSX transforms (plan 0051) and full runtime
augmentation.

## Quick start

```text
MEW_EXPERIMENTAL_RUNTIME=1 m app.js
MEW_EXPERIMENTAL_RUNTIME=1 m --node app.js          # zero-augmentation
MEW_EXPERIMENTAL_RUNTIME=1 m node-args -- --trace-warnings app.js
```

## Gate

`MEW_EXPERIMENTAL_RUNTIME=1` enables all three invocation styles:

| Style | Example | Augmentation |
|---|---|---|
| File-run (default) | `m app.js` | Full (preloads injected) |
| File-run (zero-aug) | `m --node app.js` | None (stock Node) |
| Node-args | `m node-args -- <v8-flags> <entrypoint>` | Full (preloads injected) |

## Dispatch precedence

File-run dispatch sits between exact `package.json` script matching and bin
dispatch:

1. Built-in command (`install`, `run`, …)
2. Built-in alias
3. Exact `package.json` script
4. **JS entrypoint** (`.js`, `.mjs`, `.cjs`) when `MEW_EXPERIMENTAL_RUNTIME=1`
5. Verified local bin
6. Typed suggestions

Built-in commands always win over same-named JS files. Use `m run <script>` to
force a script when a name collides.

## Node discovery

Mew discovers Node from the system `PATH` on every invocation. There is no
persistent version mapping, no download, and no network access (deferred to
plan 0060).

- `m app.js` → finds `node` on `PATH`
- Explicit override via `m node-args` or future `MEW_NODE_HOME` are deferred to
  plan 0060.

After discovery, the Node version is parsed and capability flags are assigned:
- Node ≥ 12: `require-preload` (CommonJS preload support)
- Node ≥ 16: `import-preload` (ESM preload support)

## Augmentation

### Default (full)

When augmentation is enabled, Mew injects preload assets into the Node argv:

```
node --require <cache>/preload.cjs --import <cache>/preload.mjs <entrypoint> [app-args]
```

The preload files are minimal bootstrap boundaries. Plan 0051 adds TypeScript
transform hooks via these preloads.

### Zero-augmentation (`--node`)

`m --node app.js` bypasses all injection and runs stock Node:

```
node <entrypoint> [app-args]
```

This is the escape hatch for programs that conflict with Mew's runtime or need
precise control over Node flags.

## Runtime assets

Embedded assets live in `internal/runtime/assets/`:

| Asset | Module type | Purpose |
|---|---|---|
| `preload.cjs` | CommonJS | CJS bootstrap boundary |
| `preload.mjs` | ESM | ESM bootstrap boundary |
| `manifest.json` | — | Content-addressed asset catalog |

Assets are extracted to `<cache-root>/runtime/<bundle-version>/` on first use
with SHA-256 verification and atomic writes.

## Error codes

See [`errors.md`](errors.md#runtime-mvp-0050) for the full error table.

## Architecture

| Package | Purpose |
|---|---|
| `internal/runtime/` | LaunchRequest → LaunchPlan → ProcessSupervisor |
| `internal/runtime/assets/` | Embedded JS and manifest |
| `internal/node/` | PATH discovery and version parsing |

Runtime and node packages must not import `resolver`, `linker`, `store`, or
`fetch`. See [`forbidden-imports.md`](architecture/forbidden-imports.md).

## Related

- [`cli.md`](cli.md) — `--node` flag, `node-args` subcommand, command precedence
- [`runner.md`](runner.md) — script runner and exec dispatch
- [`errors.md`](errors.md) — error code catalog
- [`architecture/package-map.md`](architecture/package-map.md) — package boundaries
