<!--
Ownership: curated terminal help for `m help errors ERR_M_USAGE`.
Authoritative: docs/errors.md, docs/cli.md
-->

# ERR_M_USAGE

## Meaning

Invalid arguments, unknown command/selector, or flag misuse. Exit status is **2**.

## Common causes

- Unknown command or script selector
- Missing required args
- Invalid flag values or conflicting flags
- Workspace flags without `-r` / `--filter` when required

## How to diagnose

```text
m <command> --help
m help <topic>
m features
```

## Safe recovery

1. Re-run with `--help` for the intended command.
2. For unknown names, try `m run <script>` or `m exec <bin>`.
3. Check experimental gates when a feature is intentionally disabled.

## Related commands

```text
m --help
m help runner
m help configuration
```

## When to report a bug

Report when documented syntax is rejected without an actionable usage message.

## See also

- docs/errors.md
- docs/cli.md
- `m help errors`
