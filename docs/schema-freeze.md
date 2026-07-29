# Schema freeze — PM core (0031)

Contract freeze for persistent formats and machine-readable outputs shipped in
MVPs **0010–0030**. Runner MVPs (**0040+**) depend on these shapes remaining
backward compatible unless an ADR authorizes a breaking change.

See also: [`lockfile.md`](lockfile.md), [`core-certification.md`](core-certification.md).

## Frozen artifacts

| Artifact | Version field | Frozen since | Change process |
|---|---|---|---|
| `m.lock` | `lockfileVersion: 3` | MVP 0031 | ADR + migration tool + fixture regen |
| Canonical graph | `graph.SchemaVersion: 3` | MVP 0015 | ADR; lock encoder/decoder must round-trip |
| Install result JSON | implicit (no top-level schemaVersion) | MVP 0016 | Additive fields only; see `app.InstallResult` |
| Mutation plan JSON | `plan.SchemaVersion: 1` | MVP 0028 | ADR for breaking plan shape |
| Audit report | `schemaVersion: 1` | MVP 0030 | ADR; see [`audit.md`](audit.md) |
| SBOM export | format flag (`cyclonedx` / `spdx`) | MVP 0030 | ADR; golden tests in `fixtures/sbom/` |
| Policy report | `schemaVersion: 1` | MVP 0030 | ADR; see [`policy.md`](policy.md) |
| Org policy file | `schemaVersion: 1` | MVP 0030 | ADR |
| Doctor report | `schemaVersion: 1` | MVP 0031 | Additive checks only |
| Core conformance report | `schemaVersion: 1` | MVP 0031 | Additive suite metadata only |
| Transaction journal | `schemaVersion` in lock doc | MVP 0017 | ADR; recovery must handle prior version |

## `m.lock` v3 (native)

Frozen fields and semantics:

- Top-level: `lockfileVersion`, `checksum`, `settings`, `importers`, `packages`, `edges`, optional `extensions`
- Package keys encode peer context, patches, and protocols deterministically
- Checksum covers canonical serialized bytes; `m lock format` must not change ordering rules without ADR
- Incumbent lockfiles (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `bun.lock`, `nub.lock`) are **not** frozen — Mew preserves incumbent bytes on no-op paths; semantic rewrite requires certified producer evidence (see [`lockfile.md`](lockfile.md))

## Install result JSON

Emitted by `m install`, `m add`, `m remove`, `m ci`, `m update`, `m dedupe`,
`m prune`, and related flags with `--json`.

Frozen shape (`internal/app.InstallResult`):

```json
{
  "added": 0,
  "removed": 0,
  "changed": 0,
  "packages": 0,
  "plan": { "schemaVersion": 1 },
  "committed": true
}
```

New fields may be appended. Existing field types and meanings must not change
without an ADR.

## Audit / SBOM / policy reports

| Report | Schema | Doc |
|---|---|---|
| `AuditReport` | `schemaVersion: 1` | [`audit.md`](audit.md) |
| CycloneDX / SPDX SBOM | format version in output | [`sbom.md`](sbom.md) |
| `PolicyReport` | `schemaVersion: 1` | [`policy.md`](policy.md) |

## CLI grammar freeze

Shipped PM command names, primary flags, and exit-code contracts documented in
[`pm-commands.md`](pm-commands.md) and [`cli.md`](cli.md) are frozen. New
commands may be added; renaming or removing shipped commands requires an ADR.

Stabilization-only commands (`m doctor`, `m conformance`) are part of the 0031
surface and follow the same freeze from MVP 0031 onward.

## Non-frozen (explicit)

- Runner and runtime CLI (`m run`, `mx`, loaders) — MVP 0040+
- Full 0080 differential conformance report schema
- Live Sigstore attestation verification protocol
- Advisory feed signature format
