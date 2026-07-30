<!--
Ownership: curated terminal help for `m help errors ERR_M_LOCKFILE`.
Authoritative: docs/errors.md
-->

# ERR_M_LOCKFILE

## Meaning

Lockfile parse, checksum, graph validation, or frozen-manifest drift failed.

## Common causes

- Lockfile bytes do not match `package.json` after a manifest edit
- Corrupt or truncated lockfile
- Frozen install with an outdated lockfile
- Unsupported or malformed lock layout

## How to diagnose

```text
m lock validate
m doctor
m explain <package>
```

## Safe recovery

1. Confirm the intended package manager identity and lockfile path.
2. Run `m install` to refresh the lockfile when mutation is intended.
3. For CI frozen installs, regenerate the lockfile in a normal install first.
4. Do not delete lockfiles, caches, or stores casually.

## Related commands

```text
m install
m lock validate
m doctor
m recover
```

## When to report a bug

Report when a valid lockfile from a supported producer fails validation without a clear drift message.

## See also

- docs/errors.md
- docs/lockfile.md
- `m help errors`
