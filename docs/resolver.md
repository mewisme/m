# Resolver

Dry dependency resolution: registry metadata in, deterministic `graph.Graph` and
decision traces out. No `node_modules` mutation (0016). Lockfile write is via
`app.WriteLock` / `m lock` (0015).

## Inputs

| Input | Source |
|---|---|
| Root manifest | `project.Open` → `manifest.ToNormalized` |
| Registry | `registry.Client.Packument` via scoped URL routing |
| Policy | `ResolveOptions.Policy` — release age, deprecated rejection, peer policy |
| Hints / prior graph | `ResolveOptions.Hints` / `ResolveOptions.Prior` — incremental reuse (0015, 0020) |
| Workspace index | `workspace.Index` when `workspaces` is declared |

## Outputs

`resolver.Resolution`:

- `SchemaVersion`
- `Graph` — importers, packages, edges (validated / sorted)
- `Decisions` — per-request candidates, selected version, reason, policy rejects, peer context
- `Extensions` — e.g. `mew.resolver/local` for workspace and local-source placeholders

## Rules

### Core (0013)

- Root importer resolves `dependencies` and `devDependencies`.
- Transitive packages expand **prod** `dependencies` only — never their `devDependencies`.
- Semver: `internal/semver` over Masterminds/v3 (`^`, `~`, `*`, `x`, unions, hyphen ranges). Dist-tags resolve in the registry layer before range selection.
- Fail closed: missing packument, unsatisfiable range, dependency cycle, or limit exceeded → `ERR_M_RESOLVE`.
- Limits: `maxDepth=64`, `maxPackages=10000` (raise later for large monorepos).

### Peer dependencies (0020)

- Collect `peerDependencies` (and optional `peerDependenciesMeta`) from packuments.
- Match peers reachable from the importing context.
- **Strict by default** (`resolve.strictPeerDependencies=true`): missing required peers fail with `ERR_M_RESOLVE` and a structured peer conflict (see `m explain peer`).
- **Auto-install** (`resolve.autoInstallPeers=true`): enqueue missing peers as prod edges from the importer (npm 7+ escape hatch).
- When peer sets diverge, assign `graph.PeerContext` on `PackageID`. Key format: `name@version#peer@range,...` (sorted peer names). Golden: `testdata/graph/peers.json`.

### Optional and platform (0020)

- Root `optionalDependencies` are seeded like prod deps but marked optional.
- Optional edges skipped when `os` / `cpu` / `libc` on the packument version do not match the current target (`resolver.CurrentTarget()`).
- Optional transitive resolution failures are recorded (`optional-failed`) and do not fail the install graph.

### Overrides and aliases (0020)

- npm-style nested `overrides` rewrite specifiers before queueing; nearest importer override wins.
- `npm:bar@^1` aliases resolve `bar` while keeping the declared edge label.

### Workspace protocol (0020)

- `workspace:*` → exact member version from the member `package.json`.
- `workspace:^` → `^memberVersion` range satisfied only by that member.
- Missing target → `ERR_M_RESOLVE` with `workspace target "pkg" not found`.
- Workspace members register as graph nodes with empty `integrity` / `tarballUrl` and `mew.resolver/local` `{protocol:"workspace", path:"..."}`.

### Local sources (0020 placeholders)

- `file:`, `link:`, `portal:` resolve into the graph and lock extensions only.
- Install defers with actionable `ERR_M_INSTALL` until linker MVP wires copy/link.
- `portal` and `link` record distinct `protocol` values in the extension payload.

### Incremental resolve (0020)

- `ResolveOptions.Prior` + `Hints` reuse pinned versions when specifiers, overrides, and update closure are unchanged.
- `m update [pkg...]` re-resolves with `UpdateTargets`; empty args refresh direct deps only while preserving unrelated subgraph.

## CLI

```text
m resolve [--plan] [--json] [--trace]
m update [pkg...] [--latest]
m explain peer <name> [--json]
```

`--plan` is the dry-resolve mode (default). `--json` emits `Resolution`.
`--trace` prints peer context, override rewrites, skipped optional deps, and workspace/local resolutions.
Full `m explain` beyond the peer subcommand is MVP 0028.

## Lockfile handoff (0015 / 0020)

`app.WriteLock` builds `m.lock` from `Resolution` plus manifest specifiers,
config settings (including resolver policy), and extensions. `app.ReadLockGraph`
feeds `ResolveOptions.Hints` / `Prior` for incremental resolve. See
[`lockfile.md`](lockfile.md).

## Fixtures

| Path | Exercises |
|---|---|
| `fixtures/resolver/peers/react-ecosystem/` | Peer contexts, strict failure, auto-install |
| `fixtures/resolver/optional-platform/` | OS/CPU optional skip |
| `fixtures/resolver/overrides-nested/` | Nested transitive override |
| `fixtures/projects/workspace-protocol/` | `workspace:*` / `workspace:^` |

## Related

- [`docs/errors.md`](errors.md) — `ERR_M_RESOLVE`
- [`docs/architecture/interfaces.md`](architecture/interfaces.md) — `Resolver`
- [`docs/data-model.md`](data-model.md) — canonical graph
