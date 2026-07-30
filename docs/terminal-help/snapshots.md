<!--
Ownership: curated terminal help for `m help snapshots`.
Authoritative: docs/ (snapshot command surfaces); no dedicated long-form doc yet.
-->

# Snapshots

Install snapshots record historical install state for inspection and restore workflows.

## Commands

```text
m snapshot list
m history
m rollback
m recover
```

## Safety

- Treat restore as a mutation: use transaction recovery paths.
- Do not delete `.mew` snapshot data casually to "fix" a failed restore.
- Prefer `m recover` when transaction state is incomplete.

## Labels

Snapshot and capsule outputs use safe labels in human mode. Structured modes keep machine fields unchanged.

## See also

- `m snapshot --help`
- `m help capsules`
- `m help errors ERR_M_TRANSACTION`
