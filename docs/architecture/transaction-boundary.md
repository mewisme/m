# Transaction boundary

Every install-family mutation follows one pipeline. The old manifest, lockfile,
and `node_modules` remain usable until commit. On failure before the committed
marker, rollback restores the pre-mutation state.

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
| rollback | `transaction` | Inverse applied ops + restore backups |
| post-commit | `snapshot` | Prune old snapshots (non-rollback) |

## Install-family mutations

All of the following must use the transaction boundary when they mutate the
project or global store:

- `install` / `i` / `add` / `remove` / `update` / `upgrade`
- Lockfile rewrite that accompanies an install
- Relink / prune that changes `node_modules`
- Global package install into the content store when tied to a project mutation
- Lifecycle script runs that are part of an install commit (sandbox under policy)

Read-only commands (`why`, `list`, `outdated` when non-mutating) must not open a
mutating transaction.

## Resolve-complete-before-mutate

Resolver decisions are independent from disk mutation. Resolve a complete
immutable graph before fetch/link/commit. Do not stream partial resolution into
live `node_modules`.

## Journaling

`internal/transaction` journals install-family mutations under
`<project>/.mew/txn/<id>/journal.v2.json` with a forward **plan**, per-op
**progress**, and inverse backup metadata (including prior `node_modules` kind).

A project-level lock at `.mew/txn/lock` prevents concurrent install transactions.

See [`transaction.md`](../transaction.md) for format, recovery, and snapshot
retention.

## Failure semantics

- Before `committed`: `m recover` or automatic rollback restores prior live state.
- After `committed`: install artifacts are durable; post-commit errors (e.g.
  snapshot prune) surface as errors but do not roll back lock/`node_modules`.
