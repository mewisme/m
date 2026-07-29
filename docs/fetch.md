# Tarball fetch, integrity, and safe extraction

MVP **0014**. Download registry tarballs with bounded concurrency, verify
integrity before any extraction, store verified bytes in a blob cache, and
extract npm `.tgz` archives with path-traversal guards.

Installer MVPs (0016+) call these packages through `internal/app`; the CLI does
not import `fetch` or `store` directly.

## Flow

```text
plan / resolve graph
  → app.Fetch
  → fetch.Downloader (HTTP + offline cache)
  → stream hash verify (ERR_M_INTEGRITY on mismatch)
  → store.Dir Put (<cache>/blobs/<algo>/<hex>)
  → archive.Extract → dest/<name>@<version>/
```

## Integrity

| Source | Format |
|---|---|
| `dist.integrity` | SRI `sha512-…` or `sha256-…` (base64 or hex digest) |
| `dist.shasum` | Legacy sha1 hex when integrity is absent |

Hashing runs while the response body streams. Extraction never starts until the
digest matches. Signed registry URLs are redacted in error subjects (`?…`).

## Blob cache

```text
<cache.dir>/blobs/<algo>/<hex>   # verified tarball bytes only
<cache.dir>/staging/             # download temps (removed on failure/cancel)
```

`m cache verify` re-hashes every blob under `blobs/` and reports `ok` / `bad` /
`skip` counts.

## Offline

| Mode | Tarball behavior |
|---|---|
| `--offline` | Blob cache hit only; miss → `ERR_M_NETWORK` |
| `--prefer-offline` | Cache first; miss uses network |
| online | GET with Bearer auth when `registry.auth_token_env` is set |

Before fetch, `m install --offline` runs an offline preflight that verifies
every resolved tarball blob is present in the blob cache (plus registry
packuments and local/git sources). See [`offline.md`](offline.md).

Metadata offline behavior is unchanged (see [`registry.md`](registry.md)).

## Safe extraction

- gzip + tar via stdlib only
- Strip leading `package/` (npm pack layout)
- Reject absolute paths, `..`, Windows drive paths, symlink/hardlink escapes,
  and non-regular entries (devices, fifos, etc.)
- Normalize modes (dirs `0755`, files `0644`, `+x` when owner-exec set)
- Zero mtime to Unix epoch for reproducible trees

### Limits (v1 constants)

| Limit | Value |
|---|---|
| Download body | 512 MiB |
| Tar members | 100,000 |
| Expansion ratio | 10× compressed size |

HTTP **Range resume is not implemented**; retries re-download the full body
once on truncated/network errors, never on hash mismatch.

## CLI

```text
m fetch --plan-file plan.json [--dir dest] [--json]
m cache verify [--json]
```

`plan.json` shape:

```json
{
  "packages": [
    {
      "name": "lodash",
      "version": "4.17.21",
      "tarballUrl": "https://registry.example/lodash/-/lodash-4.17.21.tgz",
      "integrity": "sha256-…"
    }
  ]
}
```

Map packages from `m resolve --json` (`tarballUrl`, `integrity` on each package).

## Public Go surfaces

| Package | Entry points |
|---|---|
| `internal/fetch` | `Downloader`, `VerifyReader`, `ParseIntegrity` |
| `internal/store` | `Dir` (`Get` / `Put` on `algo/hex` keys) |
| `internal/archive` | `Extract(ctx, tgzPath, destDir, opts)` |
| `internal/app` | `Fetch`, `VerifyBlobCache`, `LoadFetchPlan` |

## Handoff

- **0016** install calls `fetch.Downloader` + `archive.Extract` after resolve
- **0018** extends/replaces `store.Dir` with the full content-addressed package store

## Fixtures

- `fixtures/registry/v1/tarballs/lodash-4.17.21.tgz`
- `fixtures/archives/traversal-attack.tgz`
- `fixtures/archives/corrupt-hash.tgz`
- `testdata/fetch/offline-cache-hit/`
- `testdata/archive/expected-lodash/files.txt`
