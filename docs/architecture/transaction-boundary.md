# Transaction boundary

Every install-family mutation follows one pipeline. The old manifest, lockfile,
and `node_modules` remain usable until commit. On failure before the committed
marker, rollback restores the pre-mutation state from backups.

## Pipeline

```text
inspect -> resolve -> plan -> fetch -> verify -> stage -> validate -> plan journal
  -> backup -> commit (all live publishes) -> post-commit cleanup
 \-> rollback on failure (before committed)
```

| Phase | Owner (typical) | Effect |
|---|---|---|
| inspect | `project`, `manifest`, `workspace`, `config` | Read-only discovery |
| resolve | `resolver`, `registry` | Immutable graph; no disk mutation |
| plan | `plan` types / installer orchestration | Desired state + operations |
| fetch | `fetch`, `archive` | Download into staging/store |
| verify | `store`, integrity checks | Fail closed on mismatch |
| stage | `linker`, `transaction` | Write under staged roots |
| validate | `transaction`, policy | Pre-commit checks |
| plan journal | `transaction` | Persist forward op list before live mutation |
| backup | `transaction` | Copy prior live state with kind metadata |
| commit | `transaction` | Publish staged artifacts; mark committed last |
| rollback | `transaction` | Restore from backups + phase-tracked rollback |
| post-commit | `snapshot` | Prune old snapshots (non-rollback, warn-only on failure) |

## Install-family mutations

All of the following must use the transaction boundary when they mutate the
project or global store:

- `install` / `i` / `add` / `remove` / `update` / `upgrade`
- Lockfile rewrite that accompanies an install
- Relink / prune that changes `node_modules`
- Global package install into the content store when tied to a project mutation
- Lifecycle script runs that are part of an install commit (sandbox under policy — **0021**)

Read-only commands (`why`, `list`, `outdated` when non-mutating) must not open a
mutating transaction.

## Resolve-complete-before-mutate

Resolver decisions are independent from disk mutation. Resolve a complete
immutable graph before fetch/link/commit. Do not stream partial resolution into
live `node_modules`.

## Journaling and locking

`internal/transaction` journals install-family mutations under
`<project>/.mew/txn/<id>/journal.v3.json` with a forward **plan**, per-op
**progress** and **phase** sub-states, and backup metadata (including prior
`node_modules` kind).

**Single mutation entrypoint:** `BeginMutation` acquires the project lock,
runs `RecoverScanned` (directory scan + idempotent rollback/discard), refuses to
begin when incomplete journals remain, then creates the new transaction.

**Session ownership (pass 6):** `BeginMutationSession` wraps `BeginMutation` for
install-family commands. Live manifest and lock reads happen only after the session
holds the project lock. `Finish` / `Abort` release the lock with verified `current`
cleanup.

A project-level lock at `.mew/txn/lock` (schema v2, exclusive create with process
identity) prevents concurrent install transactions. Acquire waits honor caller
`context` cancellation. Stale lock takeover renames the observed lock into
`.lock-tombstones/` after verifying `owner.json` (ABA-safe); normal release never
deletes a lock without verified ownership.

Recovery and rollback always restore live state from `backups/`; rename ops do not
use symmetric inverse renames for directories.

## Path security

`internal/fsx.GuardAncestors` walks existing path components with `Lstat` and
rejects symlinks and junctions under `.mew`, `node_modules`, and snapshot paths.
Transaction `GuardPath` delegates to this guard for mutation targets.

## Failure semantics

- Before `committed`: `m recover` or automatic rollback restores prior live state.
- After `committed`: install artifacts are durable; post-commit errors (e.g.
  snapshot prune) surface as warnings/errors but do not roll back lock/`node_modules`.
- Concurrent install while lock is held: `ERR_M_TRANSACTION`.

See [`transaction.md`](../transaction.md) for format, recovery, and snapshot
retention.
