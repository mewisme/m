# Native lockfile (`m.lock`)

Mew's native lockfile is deterministic JSON at the project root. Format decision:
[`adr/0004-m-lock-json.md`](adr/0004-m-lock-json.md).

## Location and identity

| Item | Value |
|---|---|
| Filename | `m.lock` |
| Project root only | yes |
| Identity signal | `m.lock` present → `mew` (see [`identity.md`](identity.md)) |
| Schema version field | `lockfileVersion` (currently `3`) |

`lockfileVersion` is independent of `graph.SchemaVersion` (also `3`).

## Document shape (v3)

```json
{
  "lockfileVersion": 3,
  "checksum": "<sha256-hex>",
  "settings": {
    "linker": "auto",
    "overridesFingerprint": "<hex>",
    "resolverPolicyFingerprint": "<hex>",
    "targetPlatformFingerprint": "<hex>",
    "policy": {
      "schemaVersion": 1,
      "scriptTrust": "ask",
      "autoInstallPeers": false,
      "strictPeerDependencies": true
    }
  },
  "importers": [
    {
      "id": ".",
      "name": "app",
      "path": ".",
      "specifiers": [
        { "name": "lodash", "range": "^4.17.21", "kind": "prod" }
      ]
    }
  ],
  "packages": [],
  "edges": [
    {
      "from": ".",
      "name": "lodash",
      "to": "lodash@4.17.21",
      "kind": "prod",
      "range": "^4.17.21"
    }
  ]
}
```

| Field | Purpose |
|---|---|
| `settings.linker` | Snapshot of `install.linker` (`auto` \| `hoisted` \| `isolated`) |
| `settings.overridesFingerprint` | Hash of effective overrides for incremental invalidation |
| `settings.resolverPolicyFingerprint` | Hash of effective resolver policy (`PolicyFromEffective`: peers, release age, deprecated, offline) |
| `settings.targetPlatformFingerprint` | Hash of OS/CPU/libc target for optional/platform edges |
| `settings.policy` | Trust and resolver policy snapshot for install handoff (see [`policy`](../internal/policy/policy.go)) |
| `settings.policy.autoInstallPeers` | Snapshot of `resolve.autoInstallPeers` (0020) |
| `settings.policy.strictPeerDependencies` | Snapshot of `resolve.strictPeerDependencies` (0020) |
| `importers[]` | Workspace packages with declared `specifiers[]` |
| `packages[]` | Resolved `graph.Package` entries (`id`, `integrity`, `tarballUrl`) |
| `edges[]` | `graph.Edge` dependency links including `name` (exposed dependency key) |
| `extensions` | Forward-compatible unknown top-level fields (omitted when empty) |

Package `id` may include `peerProviders` when peer resolution supplies resolved provider identity (MVP 0020 / schema v3).
Each entry is `{name, version, key}` where `key` is the resolved provider package key. IDs sort providers by name and append `#providerKey,...` to the
base `name@version` key (see `testdata/graph/peers.json`).

### Migration

| From | Behavior |
|---|---|
| v1 with range-based `peerContext` | Rejected with `ERR_M_LOCKFILE`; run `m lock` to regenerate |
| v2 | Loaded via adapter; edges without `name` infer `name` from target package name |
| v3 | Current format |

`extensions.mew.resolver/local` maps package keys (`name@version`) to local source
metadata:

```json
{
  "lib@2.4.0": { "protocol": "workspace", "path": "packages/lib" }
}
```

Protocols: `workspace`, `file`, `link`, `portal`. Registry packages omit this extension.

## Checksum

`checksum` is lowercase hex SHA-256 of canonical JSON over the semantic subset:

- `settings`
- `importers`
- `packages`
- `edges`

Excludes `checksum` and `lockfileVersion`. Encoding uses the same settings as
output (2-space indent, trailing newline, `SetEscapeHTML(false)`).

## Frozen semantics

`m lock validate --frozen` compares manifest specifiers to lock `importers[]`
per importer (`kind` + `name` + `range`). Drift returns `ERR_M_LOCKFILE` with
an actionable diff.

`m install --frozen-lockfile` is MVP **0016**; use `m lock validate --frozen`
today. Library entry point: `app.ValidateFrozenLock`.

When `m.lock` `settings.linker` is set, frozen installs use that linker mode
(including `isolated` when the experimental gate is enabled).

## CLI

```text
m lock format [--json]
m lock validate [--frozen] [--json]
```

- **format** — parse, normalize, re-encode; atomic write when changed.
- **validate** — parse, checksum verify, graph validate; `--frozen` adds manifest drift check.

## Go API

| Package / symbol | Role |
|---|---|
| `internal/lockfile/mlock` | Codec, checksum, frozen drift, v2→v3 migration |
| `mlock.Adapter` | `lockfile.LockfileAdapter` for `m.lock` |
| `app.WriteLock` | Resolve (if needed) and write incumbent lock |
| `app.ReadLockGraph` | Read incumbent lock into `*graph.Graph` (hints / validate) |
| `app.ValidateFrozenLock` | Manifest vs lock specifier drift (identity-aware) |

## Incumbent write policy (0023)

Install-family commands write the **incumbent** lockfile only (`nub.lock` or
`pnpm-lock.yaml` when identity is Nub or pnpm). `m.lock` is created only via
`m lock migrate --to m`.

| Condition | Behavior |
|---|---|
| Graph unchanged | Stage prior bytes unchanged (byte-identical after commit) |
| Graph changed + certified generation | Encode incumbent format inside transaction stage |
| Ambiguous v9-shaped pnpm lock | Fail closed (`ERR_M_LOCK_AMBIGUOUS`) unless `--pnpm-major` |
| Lossy / unsupported encode | Fail closed with `LossReport` (`ERR_M_LOCK_UNREPRESENTABLE`) |
| Persistence | All incumbent writes go through `install_txn` staging → commit |

## Adapter matrix

| Identity | Incumbent file | Adapter package | Generations |
|---|---|---|---|
| `mew` | `m.lock` | `internal/lockfile/mlock` | v3 JSON |
| `nub` | `nub.lock` | `internal/compat/nub` | pnpm v9-shaped YAML |
| `pnpm` | `pnpm-lock.yaml` | `internal/compat/pnpm` | **9, 10, 11 only** (v5–v8 rejected) |

pnpm **5–8** flat or legacy layouts are rejected with `ERR_M_LOCK_UNSUPPORTED` and
remediation to regenerate with pnpm 9, 10, or 11. Unsupported fixtures:
`fixtures/locks/pnpm/unsupported/`.

Detection for pnpm v9-shaped locks uses `packageManager` / `devEngines`, extension
metadata, observed root/settings field evidence, and optional `--pnpm-major` (9, 10,
or 11). Do not trust `lockfileVersion: '9.0'` alone.

CLI: `m lock validate` (incumbent), `m lock diff [other]`, `m lock migrate
--from nub|pnpm --to m` (`--dry-run` emits migration report JSON).

`--pnpm-major` (9, 10, or 11) applies to `m lock validate`, `m lock diff`,
`m lock migrate`, and `m install`.

## Support matrix (Pass 18)

| Generation | Read | Byte no-op | Semantic rewrite | Migrate to m.lock | Frozen pnpm CI |
|---|---|---|---|---|---|
| pnpm v5–v8 | **rejected** | — | — | — | `conformance-pnpm-unsupported` |
| pnpm v9 | yes | yes | yes with `--pnpm-major` | dry-run loss report; non-dry-run fail-closed on semantic loss | `conformance-pnpm-9` + `MutationSuite` |
| pnpm v10 | yes | yes | yes with `--pnpm-major` | dry-run loss report; non-dry-run fail-closed on semantic loss | `conformance-pnpm-10` + `MutationSuite` |
| pnpm v11 | yes | yes | yes with `--pnpm-major` | dry-run loss report; non-dry-run fail-closed on semantic loss | `conformance-pnpm-11` + `MutationSuite` |
| nub | yes | yes | policy from incumbent bytes | yes (loss report) | `conformance-nub-fixtures` (6 families) |
| m.lock v3 | yes | yes | yes | n/a | n/a |

Pinned producer versions: `tools/conformance/pnpm-versions.env` (9.15.9 / 10.34.5 / 11.17.0).
Verify committed fixtures: `go run ./tools/conformance/verify-fixtures` (CI `fixture-verify`).

Mutation conformance runs full `pnpm install --frozen-lockfile` (not `--lockfile-only`),
strict byte hash, `node_modules` import checks, add/update/remove txn paths, and
commit-interrupt restore. Snapshot-primary instances: graph nodes from `snapshots`
keys only (no phantom base package nodes). npm aliases resolve by actual package
name. `Adapter.Write` fails closed without certified `--pnpm-major`. Strict
`packageManager` semver (`pnpm@10.not-a-semver` rejected). Mutation families:
basic, transitive, optional, peer-context, workspace, alias, patch (frozen after
remove). Bare `packageManager: pnpm` does not certify major 9.

Fixtures: `fixtures/locks/generated/` with `metadata.json` (SHA-256, producer,
command). Evidence: `fixtures/locks/EVIDENCE.md`.

## Loss report

`LossReport` (`schemaVersion: 1`) lists fields that cannot round-trip to the
target format. Returned on `m lock migrate --dry-run` and on
`ERR_M_LOCK_UNREPRESENTABLE` encode failures.

## Write safety

`mlock.WriteAtomic` uses temp file + rename (Windows replace fallback), same
pattern as `manifest.Document.Write`.

## Adapter handoff

Other lockfile formats (nub, pnpm, npm, …) implement `lockfile.LockfileAdapter`
and normalize into `graph.Graph`. See [`data-model.md`](data-model.md) for
extensions and loss reports.

## Fixtures

| Path | Purpose |
|---|---|
| `testdata/lockfile/mlock/golden/basic/` | Single-importer round-trip |
| `testdata/lockfile/mlock/golden/workspace/` | Multi-importer round-trip |
| `testdata/lockfile/mlock/corrupt/` | Parse/checksum/version failures |
| `internal/lockfile/mlock/migrate_v2_test.go` | v2→v3 edge name migration |
| `fixtures/projects/mlock-greenfield/` | Greenfield project + `m.lock` |
| `fixtures/locks/nub/`, `fixtures/locks/pnpm/` | Per-generation lock bridge fixtures |

## Out of scope (0015)

- Foreign lockfile adapters beyond Nub/pnpm (0024+)
