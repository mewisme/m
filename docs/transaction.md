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
- **Release:** `ReleaseDirLock` verifies `owner.json` ownership before delete.
  Missing or malformed owner metadata returns typed `ReleaseResult` without
  removing the lock directory. Stale takeover uses tombstone rename (ABA-safe),
  not blind `RemoveAll`.
- Wired through install, add, remove, update, frozen install, snapshot restore,
  and recover.

## Mutation session (pass 6)

Install-family commands acquire a `MutationSession` via `BeginMutationSession` before
reading live `package.json` or `m.lock`. The session owns the project lock until
`Finish` or `Abort`:

1. Acquire project lock and recover incomplete journals (`BeginMutation`)
2. `ReopenProject` — first live manifest/lock/config read after ownership is held
3. `AppContext` — session copy with reloaded effective config (required after step 2)
4. Run resolve/fetch/link/validate through `runInstallInSession`
5. `Finish` — commit journal, verified `current` cleanup, release lock

**Ordering contract:** `AppContext()` returns an error until `ReopenProject` (or
`ReloadEffectiveConfig`) has run. Production paths must never call `AppContext`
before `ReopenProject`; the shared `Context` from `BeginMutationSession` input is
never mutated.

`Abort` rolls back and releases the lock. Snapshot restore and `m rollback` use the
same session boundary so restore cannot race an in-flight install.

`FinishOpts` splits cleanup between the runner (clear `current` only) and the session
(verified lock release). `FinishResult.HasCriticalCleanupFailure` surfaces incomplete
lock release or `current` pointer cleanup after commit.

## Mutation preflight

`BeginMutation` is the single entrypoint for new install-family transactions:

1. Generate transaction ID and acquire project lock
2. `RecoverScanned` — directory scan (not only `current`) rolls back or discards
   any authoritative incomplete journal
3. Re-scan; refuse to begin if incomplete state remains (`ERR_M_INTEGRITY`)
4. Create txn dirs, journal, and `current` pointer

`m recover` and automatic preflight on the next install/update/restore share the
same scan + recovery logic.

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
never via symmetric inverse rename for directories. Windows **junction** backups store
metadata under `backups-meta/` (schema v1: reparse tag, substitute, print name, relPath)
so rollback can recreate the mount point without copying the target tree. Regular files
such as `foo.reparse.json` stay in the mirrored `backups/` tree as ordinary files.

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

Restore runs through the same `runInstallTxn` path as install: staged manifest,
lock, and `node_modules` publish inside one journal commit. A failure before
`committed` leaves the prior live tree intact.

## Recovery commands

| Command | Behavior |
|---|---|
| `m recover` | Idempotent: roll back or discard an incomplete transaction |
| `m snapshot list` | List snapshots (`--json`) |
| `m snapshot restore <id>` | Restore a snapshot |
| `m rollback` | Restore the previous snapshot |

`m recover` handles interrupted commits (including partial `node_modules` rename
on Windows) by restoring from backups and replaying rollback phases.

## Cleanup visibility

`Finish` / `Discard` / `Rollback` return `FinishResult` with `LockReleased`,
`CurrentCleared`, and `CleanupWarnings` (each tagged with a cleanup code).

Severity is classified centrally via `CleanupCodeSeverity`:

| Severity | Codes | Error chain |
|---|---|---|
| Critical | `transaction_lock_release`, `transaction_current_cleanup` | `FinishResult.CriticalCleanupError()` → `apperr.JoinCleanup` on abort/finish |
| Warning | `txn_dir_remove`, `finish_hook`, store lock release codes | `FinishResult.WarningErrors()` → result fields + debug only |

`FinishResult.CleanupError()` is an alias for `CriticalCleanupError()` only.
`HasCriticalCleanupFailure()` is true for structural lock/current failures or any
critical cleanup code.

On successful install finish, warning-only cleanup sets `CleanupIncomplete` on the
result but not `RecoveryRequired` or `TransactionCleanupIncomplete`. Critical
cleanup after commit returns a non-zero exit with recover guidance.

Human output (`FormatInstallSummary`) suggests `m recover` only when
`TransactionCleanupIncomplete` or `RecoveryRequired` is set. Warning-only
committed installs print a non-critical warning line without recover guidance.
JSON output (`--json`) carries recovery/store hints only in result fields — no
prose after the JSON document.

Abort cleanup (`populateAbortCleanup`) applies the same severity split: critical
codes set recovery flags; warning codes append to result warnings only.

The installer emits debug warnings for non-critical housekeeping failures.
`m recover` may report critical cleanup when post-recovery metadata is incomplete.

## Crash injection (tests)

`MEW_TXN_CRASH_AT` kills the process at named boundaries for subprocess crash
tests: `journal_created`, `post_resolve`, `post_fetch`, `post_patch_copy`,
`post_patch_preflight`, `post_patch_apply`, `post_patch_publish`, `post_link`,
`post_lockfile`, `post_validate`, `backup:N`, `publish:N`, `commit:N`,
`pre_committed`, `committed`, `finish`, `recovery`, `rollback:N`. See
[`testing.md`](testing.md).

## Path guards

Sensitive paths (`.mew`, `node_modules`, snapshots) are guarded with ancestor
`Lstat` walks that reject symlinks and junctions in existing components. Guards
run before transaction mutations and are revalidated immediately before publish.
See `internal/fsx` and [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).

## Non-atomicity (explicit)

Mew does **not** claim multi-file filesystem atomicity. Recovery relies on ordered
journal ops, backups with prior-kind metadata, and idempotent `m recover`.

Mid-fetch resume is not supported — only rollback/recover after staging validation.

**Windows durability:** directory `fsync` via `SyncDir` treats `ERROR_ACCESS_DENIED`
as a no-op (NTFS does not support directory flush the same way as Unix). File
publishes still call `Sync` on open write handles.

`m history` lists install snapshots with delta summaries (0028). Snapshot restore:
`m snapshot restore <id>`. Ephemeral `m shell --snapshot` — MVP **0045**.

Full `node_modules` snapshot copies — restore relinks from cache.

See also: [`install.md`](install.md), [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).
