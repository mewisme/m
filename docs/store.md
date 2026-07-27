# Global content store

MVP **0018** adds a global **unpacked** package store keyed by npm integrity
(`sha512-…` / `sha256-…`). Verified tarball bytes still live in the blob cache
(`<cache>/blobs`, MVP 0014); the store holds extracted package trees once per
integrity.

## Layout

```text
<store>/
  index.json                      # rebuildable import metadata cache (optional)
  .locks/<algo>/<hex>/            # transient cross-process import lock (owner.json)
  packages/<algo>/<hex>/          # immutable unpacked package tree
    .mew-tree-manifest.json       # content index v2 (path, kind, hash, mode, symlinkTarget)
    .mew-package-integrity        # npm SRI integrity marker
  .quarantine/<algo>/<hex>/       # quarantined corrupt trees awaiting re-import
  .staging/<id>/                  # transient import staging (removed after publish)
```

Default store roots follow [`naming.md`](naming.md):

| OS | Default |
|---|---|
| Linux | `$XDG_DATA_HOME/github.com/mewisme/m/store` |
| macOS | `~/Library/Application Support/github.com/mewisme/m/store` |
| Windows | `%LocalAppData%\mew\store` |

Override with `store.dir`, `MEW_STORE_DIR`, or `MEW_HOME/store`.

## Import and publication

1. Download and verify tarball into blob cache (0014).
2. Validate package key.
3. Acquire external import lock at `<store>/.locks/<algo>/<hex>/` (directory lock via
   `owner.json`, schema v2: `lockId`, `processStart`, `pid`, `packageKey`, `createdAt`;
   stale recovery, bounded wait with context cancellation). Legacy `<hex>.lock` files are
   removed when stale.
4. Recheck destination under lock; quarantine corrupt trees only while holding
   the lock.
5. Extract into `<store>/.staging/<id>/`.
6. Write `.mew-package-integrity`, generate `.mew-tree-manifest.json` schema **v2**
   (files, directories, symlinks), verify staged tree bidirectionally.
7. Set tree read-only (best-effort per OS).
8. Atomically rename into `packages/<algo>/<hex>/`.
9. Verify published tree; upsert `index.json` while still holding the import lock.
   Index write failures are reported via the optional store reporter and do not fail
   import.
10. Release import lock.

Re-import of the same integrity is a no-op when the existing entry verifies.
Corrupt entries are quarantined under `.quarantine/<algo>/<hex>/` and
re-imported from the verified tarball on the next install.

`m store status` and import both run stale `.staging/` cleanup (dirs older than
one hour).

## Index contract (rebuildable cache)

`index.json` is an **optional, rebuildable cache** of import metadata. Import,
verify, and prune decisions use the filesystem under `packages/` as source of
truth. A missing, corrupt, or stale `index.json` does not block imports.

- **Upsert on import:** after a successful publish, Mew upserts the entry while
  holding the import lock. Failures are warned through the optional store
  reporter (`Debug` with `key` and `error` attrs) and do not fail the import.
- **Rebuild:** `ReconcileIndex()` scans `packages/`, reads `.mew-package-integrity`
  markers, and rewrites `index.json`. Use this to repair missing entries or drop
  orphan index rows for packages that no longer exist on disk.
- **Status fallback:** when the index is empty, `m store status` falls back to a
  filesystem scan under `packages/`.

## Verification

`VerifyPackage` walks `.mew-tree-manifest.json` (schema **v2**) on every reuse.
Manifests with `schemaVersion: 0` or unsupported versions are rejected.
Bidirectional checks detect:

- modified file content (per-file sha256)
- symlink target changes
- permission mode drift
- extra files or directories not listed in the manifest
- type swaps (file ↔ symlink ↔ directory)
- duplicate or escaping manifest paths (including Windows reserved names)
- invalid file hashes (sha256 hex) or symlink targets

Legacy trees without `.mew-tree-manifest.json` **fail verification** with an
actionable error (`re-import tarball to restore content index`). The next
`ImportFromTarball` re-imports from the tarball when it is still available;
marker-only reuse is not allowed.

## Project reference manifest

After a successful install with the global store enabled, Mew writes
`<project>/.mew/store-manifest.json` listing integrity keys for packages in the
current graph. `m store prune` uses these manifests, active transaction journals
(staged `store-manifest.json` under `.mew/txn/<id>/stage/`), and scan roots under
`MEW_HOME` to decide which store entries are still referenced.

## Commands

| Command | Purpose |
|---|---|
| `m store path` | Print resolved store root |
| `m store status` | Path, package count, bytes on disk; cleans stale staging |
| `m store prune [--dry-run]` | Remove unreferenced `packages/` entries |
| `m development doctor filesystem` | Probe hardlink/reflink/symlink/junction support |

## Experimental gate

Global store + smart linker are **off by default**:

- Config: `link.use_global_store` (default `false`)
- Env: `MEW_EXPERIMENTAL_GLOBAL_STORE=1`

When disabled, installs keep the 0016/0017 behavior (per-transaction extract +
copy into `node_modules`).

## Link planner

When the gate is on, the hoisted linker asks the filesystem planner for the
fastest safe strategy per package tree.

**Hardlink policy:** writable `node_modules` package trees use **reflink → copy
only**. Hardlink is disabled for project-facing package content so mutations in
`node_modules` cannot alter immutable store bytes. Store-internal dedup may still
hardlink within the read-only store when both paths live under `packages/`.

On the same volume: reflink when supported, otherwise copy. Across devices:
copy only. Failures on a single path fall back to copy.

Install diagnostics emit a `link` phase summary:
`hardlink=N reflink=N copy=N …`.

## Non-goals (0018)

- No mid-fetch resume into the store (blob cache only during download).
- Prune does not scan arbitrary repos — manifest + active txn journal based only.
- Isolated virtual store layout is MVP **0019**.

## Garbage collection rules

- Never mutate a published `packages/<algo>/<hex>/` tree in place.
- Corrupt entries are quarantined and re-imported on next install (`ERR_M_STORE`).
- Packages with an active `.locks/<algo>/<hex>/` directory are never pruned.
- `m store prune --dry-run` previews removals in deterministic key order; without
  `--dry-run`, unreferenced package directories are removed. Tarball blobs in
  `<cache>/blobs` are unaffected.
