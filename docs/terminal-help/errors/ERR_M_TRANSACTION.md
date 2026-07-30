<!--
Ownership: curated terminal help for `m help errors ERR_M_TRANSACTION`.
Authoritative: docs/errors.md
-->

# ERR_M_TRANSACTION

## Meaning

Transaction journal, commit, rollback, recovery, or project lock failed.

## Common causes

- Another process holds `.mew/txn/lock`
- Commit or publish interrupted mid-flight
- Lock release without ownership
- Symlink/junction inside a guarded path

## How to diagnose

```text
m recover
m doctor
m history
```

## Safe recovery

1. Wait for concurrent installs to finish, or stop the other process deliberately.
2. Run `m recover` and follow its reported state.
3. Prefer recover/rollback over deleting `.mew` or `node_modules`.
4. After a successful recover, re-run the original command.

## Related commands

```text
m recover
m rollback
m history
m doctor
```

## When to report a bug

Report when recover leaves an incomplete authoritative tree with no actionable next step.

## See also

- docs/errors.md
- `m help errors`
