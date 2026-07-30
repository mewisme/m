<!--
Ownership: curated terminal help for `m help errors ERR_M_POLICY`.
Authoritative: docs/errors.md, docs/lifecycle.md
-->

# ERR_M_POLICY

## Meaning

A lifecycle trust or org supply-chain policy blocked the operation.

## Common causes

- Untrusted lifecycle scripts with `lifecycle.script_trust=deny|ask`
- Non-TTY / CI / structured mode cannot prompt for trust
- Org policy file rejected a package or install plan
- `mx` remote consent denied

## How to diagnose

```text
m builds
m policy check
m trust --help
```

## Safe recovery

1. Review the blocked package with `m builds` or `m policy check --json`.
2. Approve deliberately with `m trust <package>` or `m approve-builds <package>`.
3. Prefer project-local trust over broad allow modes.
4. Do not disable integrity or trust checks to force an install.

## Related commands

```text
m trust <package>
m approve-builds <package>
m builds
m policy check
```

## Related configuration

- `lifecycle.enabled`
- `lifecycle.script_trust` (`allow` | `deny` | `ask`)
- `--interactive` / `ui.interactive`

## When to report a bug

Report when an already-trusted package is blocked without a policy reason.

## See also

- docs/errors.md
- docs/lifecycle.md
- `m help lifecycle-trust`
