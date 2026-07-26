# Transaction boundary

Every install-family mutation follows one pipeline. The old manifest, lockfile,
and `node_modules` remain usable until commit. On failure, rollback restores the
pre-mutation state.

## Pipeline

```text
inspect -> resolve -> plan -> fetch -> verify -> stage -> validate -> commit
 \-> rollback on failure
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
| commit | `transaction` | Atomic promote of staged state |
| rollback | `transaction` | Restore journaled prior state |

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

`internal/transaction` (and later `internal/journal`) must journal enough
information to recover from interruption. Persistent journal formats are
versioned when introduced.
