<!--
Ownership: curated terminal help for `m help errors ERR_M_INTEGRITY`.
Authoritative: docs/errors.md
-->

# ERR_M_INTEGRITY

## Meaning

Checksum, artifact, provenance, or ambiguous recovery-state verification failed.

## Common causes

- Tarball digest mismatch
- Corrupt store entry or offline cache
- Ambiguous incomplete transaction state
- Provenance attestation mismatch

## How to diagnose

```text
m verify
m doctor
m store status
m recover
```

## Safe recovery

1. Re-run `m verify` or `m doctor` and read the subject path.
2. For incomplete transactions, use `m recover` before starting a new install.
3. Re-fetch only after confirming the failure is not an intentional policy block.
4. Do not delete the content store casually.

## Related commands

```text
m verify
m doctor
m recover
m store status
```

## When to report a bug

Report when a freshly fetched artifact fails integrity with a trusted registry and unchanged lockfile.

## See also

- docs/errors.md
- `m help errors`
