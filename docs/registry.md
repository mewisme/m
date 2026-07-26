# Registry client and metadata cache

MVP **0012**. Fetch npm-compatible packuments with disk ETag cache, scoped
registries, auth, proxy/CA transport, and offline fail-closed behavior.

## Client contract (for resolver)

```go
type Registry interface {
  Metadata(ctx, name, version) (*PackageMetadata, error)
}

client.Packument(ctx, registryBase, name) (*Packument, error)
```

`PackageMetadata` carries `Name`, `Version`, `Integrity`, `TarballURL`.
`Packument` carries dist-tags and version map (dependencies fields available for 0013).

Construct via `registry.NewFromApp(eff, projectRoot, identity, environ)` or
`registry.NewClient(Options{...})`.

## Cache layout

```text
<cache.dir|/platform-cache>/registry/<originHash8>/<escapedName>/
  meta.json       # schemaVersion, etag, sha256, fetchedAt
  packument.json  # raw body
```

Cache keys use registry origin + package name — **never** auth tokens.
Freshness is **ETag / If-None-Match only** (no TTL). Corrupt entries are evicted.

## Offline

| Mode | Behavior |
|---|---|
| `--offline` | Cache hit only; miss → `ERR_M_NETWORK` subject `offline` |
| `--prefer-offline` | Cache first; miss falls through to network |
| online | Conditional GET when etag known; 304 reuses body |

Online 404 → `ERR_M_NOT_FOUND`.

## Scoped registries

1. `registries.@scope` in Mew config
2. Default `registry`
3. For non-mew identity: minimal `.npmrc` lines `registry=` / `@scope:registry=`

## Auth / transport

- Bearer token from env named by `registry.auth_token_env` (never logged)
- `network.proxy` (http/https) or `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`
- `network.ca_file` PEM for custom CA
- SOCKS proxies are **not** supported yet (`ERR_M_CONFIG`)

Shared HTTP client: [`internal/fetch`](fetch.md) (transport in 0012; tarball
download, integrity, and extraction in 0014).

## CLI

```text
m view <name>[@version] [--json]
m cache dir
m cache verify [--json]
m cache metadata inspect <name> [--json]
```

## Gaps

Full content store (0018), SOCKS, packument signature verify (0030).
