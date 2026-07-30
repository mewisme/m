<!--
Ownership: curated terminal help for `m help lifecycle-trust`.
Authoritative: docs/lifecycle.md
-->

# Lifecycle trust

Lifecycle scripts are off by default. When enabled, trust policy controls whether unknown packages may run scripts.

## Modes

| Mode | Behavior |
|---|---|
| `deny` | Fail closed on untrusted packages (default) |
| `ask` | Prompt on interactive TTY; fail closed otherwise |
| `allow` | Allow scripts (still subject to other policy) |

Enable lifecycle with `MEW_EXPERIMENTAL_LIFECYCLE=1` or `lifecycle.enabled: true`.

## Ask prompts

With `lifecycle.script_trust: ask` and interactive policy:

- **Deny** (default / Enter / EOF / cancel)
- **Allow once** (current install only)
- **Trust for this project** (writes `.mew/trusted-packages.json`)

Accessible mode uses numbered prompts. Non-TTY, CI, and structured modes never prompt.

## Review commands

```text
m builds
m trust <package>
m approve-builds <package>
m trust --interactive
```

## Limitations

Sandboxing is best-effort. Trust decisions are auditable project-local state, not a full security boundary.

## See also

- docs/lifecycle.md
- docs/config.md
- `m help errors ERR_M_POLICY`
