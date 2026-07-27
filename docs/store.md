# Global content store

MVP **0018** adds a global **unpacked** package store keyed by npm integrity
(`sha512-…` / `sha256-…`). Verified tarball bytes still live in the blob cache
(`<cache>/blobs`, MVP 0014); the store holds extracted package trees once per
integrity.

## Layout

```text
<store>/
  index.json                 # optional import metadata (status, prune hints)
  packages/<algo>/<hex>/     # immutable unpacked package tree
  .staging/<id>/             # transient import staging (removed after publish)
```

Default store roots follow [`naming.md`](naming.md):

| OS | Default |
|---|---|
| Linux | `$XDG_DATA_HOME/github.com/mewisme/m/store` |
| macOS | `~/Library/Application Support/github.com/mewisme/m/store` |
| Windows | `%LocalAppData%\mew\store` |

Override with `store.dir`, `MEW_STORE_DIR`, or `MEW_HOME/store`.

## Import

1. Download and verify tarball into blob cache (0014).
2. Extract into `<store>/.staging/<id>/`.
3. Verify `package.json` and write `.mew-package-integrity`.
4. Atomically rename into `packages/<algo>/<hex>/`.
5. Update `index.json` (best-effort).

Re-import of the same integrity is a no-op when the existing entry verifies.

## Project reference manifest

After a successful install with the global store enabled, Mew writes
`<project>/.mew/store-manifest.json` listing integrity keys for packages in the
current graph. `m store prune` uses these manifests (plus scan roots under
`MEW_HOME`) to decide which store entries are still referenced.

## Commands

| Command | Purpose |
|---|---|
| `m store path` | Print resolved store root |
| `m store status` | Path, package count, bytes on disk |
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
fastest safe strategy per package tree: reflink → hardlink → copy on the same
volume; copy across devices. Failures on a single path fall back to copy.

Install diagnostics emit a `link` phase summary:
`hardlink=N reflink=N copy=N …`.

## Non-goals (0018)

- No mid-fetch resume into the store (blob cache only during download).
- No cross-process store leases (best-effort, same as blob cache).
- Prune does not scan arbitrary repos — manifest-based only.
- Isolated virtual store layout is MVP **0019**.

## Garbage collection rules

- Never mutate a published `packages/<algo>/<hex>/` tree in place.
- Corrupt entries are deleted and re-imported on next install (`ERR_M_STORE`).
- `m store prune --dry-run` previews removals; without `--dry-run`, unreferenced
  package directories are removed. Tarball blobs in `<cache>/blobs` are unaffected.
