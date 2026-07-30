<!--
Ownership: curated terminal help for `m help errors ERR_M_CANCELLED`.
Authoritative: docs/errors.md
-->

# ERR_M_CANCELLED

## Meaning

The operation was cancelled by interrupt or context cancel. Exit status is **130**.

## Common causes

- Ctrl+C / SIGTERM during install, run, or lock wait
- Parent context cancelled during workspace orchestration

## How to diagnose

Check whether cancellation was intentional. Incomplete mutations should leave recovery instructions.

## Safe recovery

1. If a mutation was in flight, run `m recover`.
2. Re-run the command when ready.
3. Do not force-delete transaction state to "clear" a cancel.

## Related commands

```text
m recover
m history
```

## When to report a bug

Report when cancellation leaves an authoritative partial tree without a recover path.

## See also

- docs/errors.md
- `m help errors`
