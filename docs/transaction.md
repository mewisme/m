# Transactions

MVP **0017** journals install-family mutations under `<project>/.mew/txn/<id>/` and
records restorable snapshots under `<project>/.mew/snapshots/`.

## Phases

```text
resolve → fetch → link → validate → plan → backup → commit → snapshot prune
 \-> rollback on failure (through commit boundary only)
```

Progress events (`resolve`, `fetch`, `link`, `validate`, `commit`, `rollback`) are
emitted through the diagnostics reporter when configured.

## Journal v2

| Artifact | Path |
|---|---|
| Project lock | `.mew/txn/lock` (PID + start time; stale locks broken on acquire) |
| Active pointer | `.mew/txn/current` |
| Transaction root | `.mew/txn/<id>/` |
| Journal | `.mew/txn/<id>/journal.v2.json` |
| Staging | `.mew/txn/<id>/stage/{extract,node_modules,package.json,m.lock,snapshots,...}` |
| Backups | `.mew/txn/<id>/backups/` |

States: `staging`, `validated`, `committing`, `committed`, `aborted`.

### Plan and op progress

Before the first live mutation, the journal records a **plan**: ordered forward ops
(lock/manifest/`node_modules`/store-manifest/snapshot publishes).

Each plan op tracks **progress**: `pending` → `applying` → `applied` (or
`rolling_back` → `rolled_back` on recovery). The journal is persisted after every
state and progress transition.

Backup ops record **prior kind** (`none`, `file`, `dir`, `symlink`) so recovery
can restore `node_modules` trees, files, or symlinks correctly.

`journal.v1.json` remains readable for recovery of older interrupted transactions.

`m install --journal` keeps the transaction directory after a successful commit
(debug).

## Commit boundary

All live publishes happen inside the journal commit phase:

- `package.json`, `m.lock`
- `node_modules` (staged rename)
- `.mew/store-manifest.json` (when global store is enabled)
- `.mew/snapshots/<id>/` and `index.json`

The journal is marked `committed` only after the last plan op succeeds.
Post-commit snapshot **prune** is best-effort cleanup and does not roll back a
committed install.

## Snapshots

Each successful commit stores `package.json`, `m.lock`, and `graphDigest`
metadata (not a full `node_modules` copy). Default retention is **10** snapshots
(`transaction.snapshot_retention` in config).

Restore copies manifest + lock, then runs `m install --frozen-lockfile` to relink
from the blob cache (offline when blobs are present).

## Recovery commands

| Command | Behavior |
|---|---|
| `m recover` | Idempotent: roll back or discard an incomplete transaction |
| `m snapshot list` | List snapshots (`--json`) |
| `m snapshot restore <id>` | Restore a snapshot |
| `m rollback` | Restore the previous snapshot |

`m recover` handles interrupted commits (including partial `node_modules` rename
on Windows) by replaying inverse plan ops and restoring backups.

## Non-atomicity (explicit)

Mew does **not** claim multi-file filesystem atomicity. Recovery relies on ordered
journal ops, backups with prior-kind metadata, and idempotent `m recover`.

Mid-fetch resume is not supported — only rollback/recover after staging validation.

Rich `m history` UX — **0028**.

Full `node_modules` snapshot copies — restore relinks from cache.

See also: [`install.md`](install.md), [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).
