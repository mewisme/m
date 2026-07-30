# Runner certification waivers

MVP **0046** stores machine-readable waivers beside the runner conformance
manifest. Waivers document known experimental or follow-up work; they cannot
hide integrity failures, security probe failures, unexpected skips, or
zero-match preflight errors.

## File location

```
tests/conformance/runner-matrix/
  manifest.json
  waivers.v1.json
```

Each suite declares `waiverPolicy`:

| Value | Meaning |
|---|---|
| `forbidden` | Any waiver application fails certification |
| `allowed` | Listed waiver IDs may produce `pass-with-waiver` |

The harness validates waiver references at load time: unique IDs, suite exists,
platform coverage, ISO dates (`openedDate <= reviewDate <= expiryDate`), and
`allowPassWithWaiver: true` before application.

## Active waivers (0046)

| ID | Suite | Follow-up | Expiry |
|---|---|---|---|
| `waiver-direct-dispatch` | `runner-dispatch-collisions` | MVP 0050 | 2026-10-31 |
| `waiver-direct-dispatch-gates` | `runner-direct-dispatch-gates` | MVP 0050 | 2026-10-31 |

Both waivers cover experimental direct-dispatch behavior shipped in MVP **0042**.
They document gate and collision semantics only; they do not waive mx consent,
snapshot/capsule offline boundaries, event schema, inspect schema, import
boundaries, or network-forbidden suites.

## Digest policy

`waiverManifestDigest` in conformance reports is SHA-256 over canonical semantic
JSON (sorted waivers, normalized platforms). Raw file bytes are not hashed.

## Review process

1. Open or extend a waiver with owner, reason, and `followUpMVP`.
2. Set `reviewDate` before `expiryDate`.
3. Update [`runner-compatibility.md`](runner-compatibility.md) status tables.
4. Re-run `m conformance verify runner` across all platform reports before
   claiming certification.

Expired waivers fail aggregation even when suite results were previously
`pass-with-waiver`.
