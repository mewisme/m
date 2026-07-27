# Transactions

MVP **0017** journals install-family mutations under `<project>/.mew/txn/<id>/` and
records restorable snapshots under `<project>/.mew/snapshots/`.

## Phases

```text
resolve → fetch → link → validate → commit → snapshot
 \-> rollback on failure
```

Progress events (`resolve`, `fetch`, `link`, `validate`, `commit`, `rollback`) are
emitted through the diagnostics reporter when configured.

## Journal

| Artifact | Path |
|---|---|
| Active pointer | `.mew/txn/current` |
| Transaction root | `.mew/txn/<id>/` |
| Journal | `.mew/txn/<id>/journal.v1.json` |
| Staging | `.mew/txn/<id>/stage/{extract,node_modules,package.json,m.lock}` |
| Backups | `.mew/txn/<id>/backups/` |

States: `staging`, `validated`, `committing`, `committed`, `aborted`.

`m install --journal` keeps the transaction directory after a successful commit
(debug).

## Snapshots

Each successful commit stores `package.json`, `m.lock`, and `graphDigest`
metadata (not a full `node_modules` copy). Default retention is **10** snapshots
(`transaction.snapshot_retention` in config).

Restore copies manifest + lock, then runs `m install --frozen-lockfile` to relink
from the blob cache (offline when blobs are present).

## Recovery commands

| Command | Behavior |
|---|---|
| `m recover` | Roll back or discard an incomplete transaction |
| `m snapshot list` | List snapshots (`--json`) |
| `m snapshot restore <id>` | Restore a snapshot |
| `m rollback` | Restore the previous snapshot |

## Non-goals (0017)

- No claim of multi-file filesystem atomicity; recovery uses ordered journal ops.
- No mid-fetch resume — only rollback/recover after staging validation.
- Rich `m history` UX — **0028**.
- Full `node_modules` snapshot copies — restore relinks from cache.

See also: [`install.md`](install.md), [`architecture/transaction-boundary.md`](architecture/transaction-boundary.md).
