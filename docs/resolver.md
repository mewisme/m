# Resolver

Dry dependency resolution: registry metadata in, deterministic `graph.Graph` and
decision traces out. No `node_modules` mutation (0016). Lockfile write is via
`app.WriteLock` / `m lock` (0015).

## Inputs

| Input | Source |
|---|---|
| Root manifest | `project.Open` → `manifest.ToNormalized` |
| Registry | `registry.Client.Packument` via scoped URL routing |
| Policy | `resolver.PolicyFromEffective` — unified install/update path loads strict peers, auto-install peers, minimum release age, reject deprecated, and offline registry blocking |
| Hints / prior graph | `ResolveOptions.Hints` / `ResolveOptions.Prior` — incremental reuse (0015, 0020) |
| Workspace index | `workspace.Index` when `workspaces` is declared |

## Outputs

`resolver.Resolution`:

- `SchemaVersion` (graph schema v3)
- `Graph` — importers, packages, edges with `Edge.Name` (validated / sorted)
- `Decisions` — per-request candidates, selected version, reason, policy rejects, peer context
- `Extensions` — e.g. `mew.resolver/local` for workspace and local-source placeholders

## Rules

### Core (0013)

- Root importer resolves `dependencies` and `devDependencies`.
- Transitive packages expand **prod** `dependencies` only — never their `devDependencies`.
- Semver: `internal/semver` over Masterminds/v3 (`^`, `~`, `*`, `x`, unions, hyphen ranges). Dist-tags resolve in the registry layer before range selection.
- **Intentional gaps vs npm** (see `testdata/semver/corpus.json` + `internal/semver/conformance_test.go`):
  - Build metadata (`+build`) is stripped before comparison; npm treats build metadata as opaque.
  - Loose / partial ranges (`1.2`, `1`) are not accepted — use exact, caret, tilde, hyphen, or unions.
  - Prerelease satisfaction follows Masterminds defaults (^ excludes prereleases unless the range explicitly includes them).
  - `latest` and other dist-tags are not parsed inside `semver`; registry resolves tags first.
- Fail closed: missing packument, unsatisfiable range, dependency cycle, or limit exceeded → `ERR_M_RESOLVE`.
- Limits: `maxDepth=64`, `maxPackages=10000` (raise later for large monorepos).

### Peer dependencies (0020)

- Collect `peerDependencies` (and optional `peerDependenciesMeta`) from packuments.
- Match peers via **ancestor walk** from the importing context.
- **Nearest provider wins:** walk contexts nearest→farthest; first compatible provider
  is selected (no global max-version scan across branches).
- **Nearest incompatible:** fail under strict peers; do not skip to a farther branch
  unless optional/auto-install policy applies.
- **Strict by default** (`resolve.strictPeerDependencies=true`): missing required peers fail with `ERR_M_RESOLVE` and a structured peer conflict tree (see `m explain peer`).
- **Auto-install** (`resolve.autoInstallPeers=true`): enqueue missing peers as prod edges from the **requesting importer/context** (not always root).
- Optional peers (`peerDependenciesMeta.optional`) may be absent without error.
- **Multiple instances:** provisional instance keys before dedup keep distinct
  `plugin@1.0.0` nodes under different peer environments (e.g. react@18 vs
  react@19). Canonical `PackageID` + `PeerProviderContext` are assigned when peer
  sets are final.
- Dependency cycles keyed by full package identity add edges to in-progress nodes instead of false-positive name-path errors.

### Optional and platform (0020)

- Root `optionalDependencies` are seeded like prod deps but marked optional.
- Optional edges skipped when `os` / `cpu` / `libc` on the packument version do not match the current target (`resolver.CurrentTarget()`). Supports npm-style `!os` negative selectors and mixed positive/negative lists.
- Optional transitive resolution failures are recorded (`optional-failed`) and do not fail the install graph.

### Overrides and aliases (0020)

- npm-style nested `overrides` rewrite specifiers before queueing; nearest importer override wins.
- `npm:bar@^1` aliases resolve `bar` while keeping the declared edge label in `Edge.Name`.
- `Edge.Range` retains the full specifier string.

### Workspace protocol (0020)

- `workspace:*` → exact member version from the member `package.json`.
- `workspace:^` → `^memberVersion` range satisfied only by that member.
- Missing target → `ERR_M_RESOLVE` with `workspace target "pkg" not found`.
- Workspace members register as graph nodes with empty `integrity` / `tarballUrl` and `mew.resolver/local` `{protocol:"workspace", path:"..."}`.
- Full workspace install wiring — **0022** (resolve-only today).

### Local sources (0020 placeholders)

- `file:`, `link:`, `portal:` resolve into the graph and lock extensions only.
- Install defers with actionable `ERR_M_INSTALL` until linker MVP wires copy/link.
- `portal` and `link` record distinct `protocol` values in the extension payload.

### Incremental resolve (0020)

- `ResolveOptions.Prior` + `Hints` reuse pinned versions when specifiers, overrides, and update closure are unchanged.
- **Edge-keyed update closure:** seeds by importer + `depName` + kind + range; traverses prior graph by canonical edges and package instance keys (not name-only).
- Reuse keys incorporate importer, edge kind/range, prior parent package key, resolved peer-provider context, override hash, and policy fingerprint from the prior lock.
- **Lock fingerprints** (`overridesFingerprint`, `resolverPolicyFingerprint`,
  `targetPlatformFingerprint`) compared against current effective policy via
  `PolicyFromEffective` — drift or a missing prior fingerprint disables unsafe
  incremental reuse.
- Parent merge during incremental update preserves full package identity keys
  (not name-only).
- Pin reuse requires full packument metadata recovery; Mew does not synthesize dependency trees from name/version/integrity alone during incremental update.
- After resolve, packages outside the update closure are merged verbatim from `Prior` so unrelated subgraphs stay byte-stable.
- `m update [pkg...]` re-resolves with `UpdateTargets`; empty args refresh direct deps only while preserving unrelated subgraph. Routed through the install transaction (`runInstallTxn`).

### Conflict explanation (0020 partial)

Peer resolution failures build a structured `ConflictNode` tree during resolve
(ancestry steps and per-environment provider search). Golden fixtures live under
`testdata/resolver/explain/`. Full product `m explain` beyond the peer
subcommand — **0028**.

### Registry cancellation

`Packument` duplicate wait and `Packuments` worker enqueue honor caller `context`
cancellation (`ERR_M_CANCELLED`).

## CLI

```text
m resolve [--plan] [--json] [--trace]
m update [pkg...] [--latest] [--dry-run] [--json]
m explain peer <name> [--json]
```

`--plan` is the dry-resolve mode (default). `--json` emits `Resolution`.
`--trace` prints peer context, override rewrites, skipped optional deps, and workspace/local resolutions.

## Lockfile handoff (0015 / 0020)

`app.WriteLock` builds `m.lock` v3 from `Resolution` plus manifest specifiers,
config settings (including resolver policy and fingerprints), and extensions.
`app.ReadLockGraph` feeds `ResolveOptions.Hints` / `Prior` for incremental resolve.
See [`lockfile.md`](lockfile.md).

## Fixtures

| Path | Exercises |
|---|---|
| `fixtures/resolver/peers/react-ecosystem/` | Peer contexts, strict failure, auto-install |
| `fixtures/resolver/peers/dual-react-plugin/` | Multiple peer-context instances |
| `fixtures/resolver/aliases/` | npm alias protocol + `Edge.Name` |
| `fixtures/resolver/optional-platform/` | OS/CPU optional skip |
| `fixtures/resolver/overrides-nested/` | Nested transitive override |
| `fixtures/projects/workspace-protocol/` | `workspace:*` / `workspace:^` |

## Related

- [`docs/errors.md`](errors.md) — `ERR_M_RESOLVE`
- [`docs/architecture/interfaces.md`](architecture/interfaces.md) — `Resolver`
- [`docs/data-model.md`](data-model.md) — canonical graph
