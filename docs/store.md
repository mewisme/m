# Global content store

MVP **0018** adds a global **unpacked** package store keyed by npm integrity
(`sha512-…` / `sha256-…`). Verified tarball bytes still live in the blob cache
(`<cache>/blobs`, MVP 0014); the store holds extracted package trees once per
integrity.

## Layout

```text
<store>/
  index.json                      # optional import metadata (status, prune hints)
  packages/<algo>/<hex>/          # immutable unpacked package tree
    .mew-tree-manifest.json       # content index (path, kind, hash, mode, symlinkTarget)
    .mew-package-integrity        # npm SRI integrity marker
    .import.lock                  # transient cross-process import lock
  packages/<algo>/.quarantine/    # quarantined corrupt trees awaiting re-import
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
2. Acquire per-package `.import.lock` under `packages/<algo>/<hex>/`.
3. Extract into `<store>/.staging/<id>/`.
4. Write `.mew-package-integrity`, set tree read-only (best-effort per OS).
5. Generate `.mew-tree-manifest.json` listing every file and symlink.
6. Verify staged tree against the manifest.
7. Atomically rename into `packages/<algo>/<hex>/`.
8. Upsert `index.json` (best-effort).

Re-import of the same integrity is a no-op when the existing entry verifies.
Corrupt entries are quarantined under `packages/<algo>/.quarantine/` and
re-imported from the verified tarball on the next install.

`m store status` and import both run stale `.staging/` cleanup (dirs older than
one hour).

## Verification

`VerifyPackage` walks `.mew-tree-manifest.json` on every reuse. It detects:

- modified file content (per-file sha256)
- symlink target changes
- permission mode drift

Legacy trees without a tree manifest still pass when the integrity marker and
`package.json` are present.

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
- Packages with an active `.import.lock` are never pruned.
- `m store prune --dry-run` previews removals in deterministic key order; without
  `--dry-run`, unreferenced package directories are removed. Tarball blobs in
  `<cache>/blobs` are unaffected.
