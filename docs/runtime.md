# Node runtime (MVP 0050–0051)

Mew can run `.js`, `.mjs`, `.cjs`, `.ts`, `.mts`, and `.cts` files directly
through the `m` CLI. 0050 provides Node launch, augmentation, and preload
injection. 0051 adds the Go transform service (esbuild), TypeScript execution,
and a content-addressed transpile cache.

## Quick start

```text
MEW_EXPERIMENTAL_RUNTIME=1 m app.js
MEW_EXPERIMENTAL_RUNTIME=1 m app.ts
MEW_EXPERIMENTAL_RUNTIME=1 m --node app.js          # zero-augmentation
MEW_EXPERIMENTAL_RUNTIME=1 m node-args -- --trace-warnings app.js
```

## Gate

`MEW_EXPERIMENTAL_RUNTIME=1` enables all invocation styles:

| Style | Example | Augmentation |
|---|---|---|
| File-run (default) | `m app.js` | Full (preloads injected) |
| File-run (TS) | `m app.ts` | Full + transform service |
| File-run (zero-aug) | `m --node app.js` | None (stock Node) |
| Node-args | `m node-args -- <v8-flags> <entrypoint>` | Full (preloads injected) |

## Dispatch precedence

File-run dispatch sits between exact `package.json` script matching and bin
dispatch:

1. Built-in command (`install`, `run`, …)
2. Built-in alias
3. Exact `package.json` script
4. **JS/TS entrypoint** (`.js`, `.mjs`, `.cjs`, `.ts`, `.mts`, `.cts`) when `MEW_EXPERIMENTAL_RUNTIME=1`
5. Verified local bin
6. Typed suggestions

Built-in commands always win over same-named JS files. Use `m run <script>` to
force a script when a name collides.

Deferred extensions (`.tsx`, `.jsx`) return an actionable plan-0052 deferral
message instead of "unknown command".

## Node discovery

Mew discovers Node from the system `PATH` on every invocation. There is no
persistent version mapping, no download, and no network access (deferred to
plan 0060).

- `m app.js` → finds `node` on `PATH`
- `m app.ts` → finds `node` on `PATH`, starts transform service on localhost

After discovery, the Node version is parsed and capability flags are assigned:

| Capability | Minimum Node | Required for |
|---|---|---|
| `require-preload` | ≥ 12 | All entrypoints |
| `import-preload` | ≥ 16 | All entrypoints |
| `module-register` | ≥ 18.19 | TypeScript entrypoints |
| `source-maps` | ≥ 20.6 | `--enable-source-maps` (auto-injected) |

### Supported Node versions

| Version | Status | Notes |
|---|---|---|
| Node 22 | Supported (LTS) | Full 0050/0051 capabilities |
| Node 20 | Supported (LTS) | Full capabilities |
| Node 18 | Supported (maintenance) | Requires ≥ 18.19 for TS |
| Node < 18 | Unsupported | Missing `module-register` for TS; JS-only may work |

## TypeScript transform

When a `.ts`, `.mts`, or `.cts` entrypoint is detected, Mew starts a local
Go transform service (esbuild) on a random TCP port. The service communicates
with the Node loader bridge over a length-prefixed JSON frame protocol.

```
m app.ts
  → Go starts esbuild transform service on 127.0.0.1:<random>
  → Node launches with loader hooks (--import loader-register.mjs)
  → loader-register.mjs registers ts-loader.mjs via module.register()
  → Node loads app.ts → loader hook fires → source sent to transform service
  → Transformed JS returned → Node executes
```

Transform credentials (`MEW_TRANSFORM_ENDPOINT`, `MEW_TRANSFORM_TOKEN`) are
captured by the Node loader thread at its creation time. The main-thread
preload (`preload.mjs`) strips them before any user module executes.

**Protocol**: JSON length-prefixed frames (u32le header + JSON body) over TCP.
Max frame size 48 MiB. Protocol version 2. Auth via bearer token.

**Cache**: Content-addressed SHA-256(`code || map`). Atomic temp+rename
publication. Metadata is commit record. Missing code/map with committed
metadata → corruption → cleaned up for re-transform.

### Module format determination

The loader determines the Node module format (`"module"` for ESM, `"commonjs"`
for CJS) based on file extension and the nearest `package.json` `"type"` field,
matching Node.js semantics:

| Extension | Package `"type"` | Format | Rationale |
|---|---|---|---|
| `.mts` | any | `module` (ESM) | `.mts` is always ESM |
| `.cts` | any | `commonjs` (CJS) | `.cts` is always CommonJS |
| `.ts`, `.tsx` | `"module"` | `module` (ESM) | Package type overrides |
| `.ts`, `.tsx` | `"commonjs"` or absent | `commonjs` (CJS) | Node default is CJS |
| `.ts`, `.tsx` | no `package.json` found | `commonjs` (CJS) | Node default is CJS |

The same rules apply after extension substitution (Issue 12): if a `.js`
specifier resolves to a `.ts` file, the format of the resolved `.ts` file
governs.

Format is included in the transform cache key. The same TypeScript source
transformed with different module formats produces separate cache entries
and cannot collide.

**Boundary lookup**: The loader walks up the directory tree from the resolved
file to find the nearest `package.json`. A nested `package.json` (e.g. in a
subdirectory with a different `"type"`) overrides the parent. Results are
cached per `package.json` directory for the lifetime of the Node process.

**Invalid `"type"` values**: Treated as `"commonjs"` (Node default). Malformed
or unreadable `package.json` files default to `"commonjs"`.

**Limitations**:
- Format determination applies to files resolved through the ESM loader hooks.
  CJS `require()` calls inside transformed modules bypass the loader hooks and
  use Node's native CJS resolution (no TypeScript extension mapping).
- Full Node16/NodeNext resolver parity (package `exports`/`imports`, custom
  loaders) belongs to Issues 14–16.

## Augmentation

### Default (full)

When augmentation is enabled, Mew injects preload assets into the Node argv:

```
node --require <cache>/preload.cjs --import <cache>/loader-register.mjs --import <cache>/preload.mjs <entrypoint> [app-args]
```

The preload files provide bootstrap boundaries. For TypeScript entrypoints,
`loader-register.mjs` registers the TS loader hooks via `module.register()`.

### Zero-augmentation (`--node`)

`m --node app.js` bypasses all injection and runs stock Node:

```
node <entrypoint> [app-args]
```

## Runtime assets

Embedded assets live in `internal/runtime/assets/`:

| Asset | Module type | Role | Purpose |
|---|---|---|---|
| `preload.cjs` | CommonJS | preload-cjs | CJS bootstrap boundary |
| `preload.mjs` | ESM | preload-esm | ESM bootstrap + credential stripping |
| `loader-register.mjs` | ESM | loader-registration | Registers TS loader hooks |
| `ts-loader.mjs` | ESM | loader-support | TS transform hook implementation |
| `manifest.json` | — | — | Content-addressed asset catalog |

Assets are extracted to `<cache-root>/runtime/<bundle-version>/` on first use
with SHA-256 verification and atomic writes.

## Error codes

See [`errors.md`](errors.md#runtime-mvp-0050) for the full error table.

## Architecture

| Package | Purpose |
|---|---|
| `internal/runtime/` | LaunchRequest → LaunchPlan → ProcessSupervisor |
| `internal/runtime/assets/` | Embedded JS and manifest |
| `internal/transform/` | Go transform service (esbuild), IPC protocol, cache |
| `internal/node/` | PATH discovery and version parsing |

Runtime and transform packages must not import `resolver`, `linker`, `store`, or
`fetch`. See [`forbidden-imports.md`](architecture/forbidden-imports.md).

## Benchmarks

```bash
go test -bench=. ./internal/runtime/... ./internal/transform/... -benchtime=5x -count=1
```

See [`evidence/runtime/0050-0051-certification.md`](evidence/runtime/0050-0051-certification.md)
for recorded results.

## Related

- [`cli.md`](cli.md) — `--node` flag, `node-args` subcommand, command precedence
- [`runner.md`](runner.md) — script runner and exec dispatch
- [`errors.md`](errors.md) — error code catalog
- [`architecture/package-map.md`](architecture/package-map.md) — package boundaries
