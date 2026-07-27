# Native lockfile (`m.lock`)

Mew's native lockfile is deterministic JSON at the project root. Format decision:
[`adr/0004-m-lock-json.md`](adr/0004-m-lock-json.md).

## Location and identity

| Item | Value |
|---|---|
| Filename | `m.lock` |
| Project root only | yes |
| Identity signal | `m.lock` present → `mew` (see [`identity.md`](identity.md)) |
| Schema version field | `lockfileVersion` (currently `2`) |

`lockfileVersion` is independent of `graph.SchemaVersion` (also `2`).

## Document shape (v2)

```json
{
  "lockfileVersion": 2,
  "checksum": "<sha256-hex>",
  "settings": {
    "linker": "auto",
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
  "edges": []
}
```

| Field | Purpose |
|---|---|
| `settings.linker` | Snapshot of `install.linker` (`auto` \| `hoisted` \| `isolated`) |
| `settings.policy` | Trust and resolver policy snapshot for install handoff (see [`policy`](../internal/policy/policy.go)) |
| `settings.policy.autoInstallPeers` | Snapshot of `resolve.autoInstallPeers` (0020) |
| `settings.policy.strictPeerDependencies` | Snapshot of `resolve.strictPeerDependencies` (0020) |
| `importers[]` | Workspace packages with declared `specifiers[]` |
| `packages[]` | Resolved `graph.Package` entries (`id`, `integrity`, `tarballUrl`) |
| `edges[]` | `graph.Edge` dependency links |
| `extensions` | Forward-compatible unknown top-level fields (omitted when empty) |

Package `id` may include `peerProviders` when peer resolution supplies resolved provider identity (MVP 0020 / schema v2).
Each entry is `{name, version, key}` where `key` is the resolved provider package key. IDs sort providers by name and append `#providerKey,...` to the
base `name@version` key (see `testdata/graph/peers.json`).

**v1 migration:** lockfiles with range-based `peerContext` are rejected with `ERR_M_LOCKFILE`; run `m lock` to regenerate v2.

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
| `internal/lockfile/mlock` | Codec, checksum, frozen drift |
| `mlock.Adapter` | `lockfile.LockfileAdapter` for `m.lock` |
| `app.WriteLock` | Resolve (if needed) and write `m.lock` |
| `app.ReadLockGraph` | Read lock into `*graph.Graph` (hints / validate) |
| `app.ValidateFrozenLock` | Manifest vs lock specifier drift |

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
| `fixtures/projects/mlock-greenfield/` | Greenfield project + `m.lock` |

## Out of scope (0015)

- `m install` / add / remove lock writes (0016)
- Foreign lockfile adapters (0023+)
- `m lock migrate` (0028)
