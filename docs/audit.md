# Audit and advisories

MVP **0030**. Read-only vulnerability scan of the installed lock graph against a
cached OSV-compatible advisory database.

See also: [`offline.md`](offline.md), [`cli.md`](cli.md), [`errors.md`](errors.md).

## `m audit`

```text
m audit [--json] [--fix]
```

| Flag | Effect |
|---|---|
| `--json` | Emit `AuditReport` JSON (schema v1) |
| `--fix` | Print suggested safe version bumps (no manifest or lock writes) |

The command loads the project lock graph (same source as `m ls`) and matches each
package name+version against `<cache>/advisory/osv.json`. Findings are sorted
by package key. Reachability analysis is **not** performed in v1 — a vulnerable
transitive dependency is reported even when not on the runtime import path.

### Human output

Table columns: advisory id, package@version, severity, title.

With `--fix`, a second section lists suggested bumps when a non-vulnerable
version exists in the registry (requires network unless packuments are cached).

### JSON schema v1

```json
{
  "schemaVersion": 1,
  "scannedAt": "2026-07-30T00:00:00Z",
  "dbDigest": "<sha256-hex of osv.json>",
  "vulnerabilities": [
    {
      "id": "CVE-2026-0001",
      "package": "vuln-pkg",
      "version": "1.0.0",
      "severity": "critical",
      "title": "…",
      "url": "…"
    }
  ],
  "fixes": []
}
```

`dbDigest` is the SHA-256 hex digest of the cached advisory bytes. Full
cryptographic signature verification of the advisory feed is deferred.

## Advisory cache

```text
<cache.dir>/advisory/osv.json
```

| Operation | Behavior |
|---|---|
| `m audit` (online) | Read cache; missing file → `ERR_M_NOT_FOUND` with seed hint |
| `m audit --offline` | Cache hit only; missing file → `ERR_M_NETWORK` |
| Maintainer refresh | `Store.Refresh(ctx, url)` (not exposed as a stable CLI in v1) |

Seed the cache on air-gapped hosts by copying `osv.json` into the advisory
directory after an online machine has refreshed it, or by restoring a capsule
that includes registry metadata and blobs (advisory data is separate — copy
`advisory/osv.json` explicitly).

## Offline workflow

1. On a connected machine, place or refresh `osv.json` under the Mew cache.
2. Copy the advisory directory to the offline host (or share a common cache root).
3. Run `m audit --offline` after `m install --offline`.

Integration fixture: `fixtures/audit/vulnerable-transitive/` with registry
`testdata/registry/audit/v1` and `testdata/advisory/fixture-osv.json`.

## Privacy

Audit is fully local. Mew does not upload lockfiles, source trees, or package
lists to external services. `--fix` may contact the configured registry to list
versions when not offline.

## Limitations (v1)

- Name+version matching only (no reachability, no CVSS re-scoring).
- OSV feed must be supplied or refreshed out-of-band; no live OSV mirror in CI.
- `--fix` suggests bumps only; it does not mutate `package.json` or `m.lock`.

## Related

- [`policy.md`](policy.md) — org license/package deny rules
- [`lifecycle.md`](lifecycle.md) — lifecycle script trust (`m trust`)
