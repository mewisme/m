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

## Project lock (schema v2)

| Artifact | Path |
|---|---|
| Project lock | `.mew/txn/lock` — exclusive create (`O_EXCL`), schema v2 metadata |
| Active pointer | `.mew/txn/current` |
| Transaction root | `.mew/txn/<id>/` |

Lock document fields: `schemaVersion`, `pid`, `processStart`, `txnId`, `createdAt`,
`projectRoot`, optional `hostname`.

- **Acquire:** `AcquireProjectLock(ctx, projectRoot, txnID)` — bounded wait with
  context cancellation (`ERR_M_CANCELLED`).
- **Stale recovery:** remove only when process identity is provably dead (not PID
  alone).
- **Release:** verify `txnId` + process identity before `Remove`; never delete
  another owner's lock.
- Wired through install, add, remove, update, frozen install, snapshot restore,
  and recover.

## Journal v3

| Artifact | Path |
|---|---|
| Journal | `.mew/txn/<id>/journal.v3.json` |
| Staging | `.mew/txn/<id>/stage/{extract,node_modules,package.json,m.lock,snapshots,...}` |
| Backups | `.mew/txn/<id>/backups/` |

Document states: `staging`, `validated`, `committing`, `committed`, `aborted`.

### Plan and op progress

Before the first live mutation, the journal records a **plan**: ordered forward ops
(lock/manifest/`node_modules`/store-manifest/snapshot publishes).

Each plan op tracks **progress**: `pending` → `applying` → `applied` (or
`rolling_back` → `rolled_back` on recovery).

Journal v3 adds **phase** sub-states per plan op:

```text
pending → prior_identified → prior_backed_up → prior_moved_aside
  → publish_started → published → applied
rollback: rollback_started → prior_restored → rollback_completed
```

The journal is persisted after every state, progress, and phase transition.

Backup ops record **prior kind** (`none`, `file`, `dir`, `symlink`, `junction`) so
recovery restores `node_modules` trees, files, or symlinks from `backups/` only —
never via symmetric inverse rename for directories.

`journal.v2.json` and `journal.v1.json` remain readable for recovery of older
interrupted transactions.

`m install --journal` keeps the transaction directory after a successful commit
(debug).

## Commit boundary

All live publishes happen inside the journal commit phase:

- `package.json`, `m.lock`
- `node_modules` (staged rename)
- `.mew/store-manifest.json` (when global store is enabled)
- `.mew/snapshots/<id>/` and `index.json`

The journal is marked `committed` only after the last plan op succeeds.
Post-commit snapshot **prune** is best-effort cleanup (warn-only on failure) and
does not roll back a committed install.

## Snapshots

Each successful commit stores `package.json`, `m.lock`, and `graphDigest`
metadata (not a full `node_modules` copy). Default retention is **10** snapshots
(`transaction.snapshot_retention` in config).

Restore copies manifest + lock, then runs `m install --frozen-lockfile` to relink
from the blob cache (offline when blobs are present).

**Known gap:** `RestoreSnapshot` writes manifest and lock before opening the
install transaction. A crash between those writes and install completion can leave
manifest/lock ahead of `node_modules`. Recovery is `m recover` + `m install`.

## Recovery commands

| Command | Behavior |
|---|---|
| `m recover` | Idempotent: roll back or discard an incomplete transaction |
| `m snapshot list` | List snapshots (`--json`) |
| `m snapshot restore <id>` | Restore a snapshot |
| `m rollback` | Restore the previous snapshot |

`m recover` handles interrupted commits (including partial `node_modules` rename
on Windows) by restoring from backups and replaying rollback phases.

## Path guards

Sensitive paths (`.mew`, `node_modules`, snapshots) are guarded with ancestor
`Lstat` walks that reject symlinks and junctions in existing components. Guards
run before transaction mutations and are revalidated immediately before publish.
See `internal/fsx` and [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).

## Non-atomicity (explicit)

Mew does **not** claim multi-file filesystem atomicity. Recovery relies on ordered
journal ops, backups with prior-kind metadata, and idempotent `m recover`.

Mid-fetch resume is not supported — only rollback/recover after staging validation.

Rich `m history` UX — **0028**.

Full `node_modules` snapshot copies — restore relinks from cache.

See also: [`install.md`](install.md), [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).
