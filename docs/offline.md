# Offline operation

MVP **0029**. Run installs without registry or tarball network access when the
lock graph, registry metadata cache, and blob store are already complete.

## Flags

| Flag | Effect |
|---|---|
| `--offline` | Network disabled; cache miss → typed error before mutation |
| `--prefer-offline` | Use cache when present; fall through to network on miss |

Both flags apply to registry metadata (`internal/registry`) and tarball fetch
(`internal/fetch`). See [`registry.md`](registry.md) and [`fetch.md`](fetch.md).

## Offline preflight

Before fetch, `m install --offline` runs `PreflightOffline` against the
resolved graph:

| Kind | Checks |
|---|---|
| `packument` | Registry metadata on disk for packages missing `TarballURL` |
| `blob` | Verified tarball bytes in `<cache>/blobs/<algo>/<hex>` |
| `git` | Pinned commit in lock (git sources are not fetchable offline today) |
| `local` | `file` / `portal` / `workspace` paths exist under the project root |

On failure Mew aborts before staging with `offline preflight failed` and up to
20 missing lines (`ERR_M_NETWORK` or `ERR_M_NOT_FOUND`). With `--json`, the
full missing list is available from integration diagnostics.

Preflight runs only when `--offline` is set and the graph is non-nil after
resolve.

## Air-gapped workflow

Typical bootstrap on a machine without registry access:

1. **On a connected machine** — install the project once online (or copy an
   existing warm cache):

   ```text
   m install
   m cache verify
   ```

2. **Export a portable capsule** (lock + manifests + cached blobs):

   ```text
   m capsule create --output ./deps.capsule
   ```

3. **Transfer** `deps.capsule` to the air-gapped host (USB, artifact mirror,
   internal registry, etc.).

4. **On the air-gapped machine** — restore and install frozen:

   ```text
   m capsule restore ./deps.capsule
   ```

   Restore extracts blobs into the local store, writes lock/manifest bytes, and
   runs a frozen install (`--offline` semantics via cached content).

5. **Day-to-day offline installs** — when cache is already warm:

   ```text
   m install --offline
   ```

`--prefer-offline` suits flaky or metered links: reuse cache, fetch only misses.

## Cache layout

```text
<cache.dir>/registry/<originHash8>/<escapedName>/packument.json
<cache.dir>/blobs/<algo>/<hex>
```

`m cache verify` re-hashes blobs. `m cache metadata inspect <name>` shows
registry cache entries.

## Capsule bootstrap

Capsules bundle:

- `m.lock` and `package.json` (plus workspace member manifests when present)
- Content-addressed blob refs from the resolved graph
- Manifest metadata: graph digest, platform, schema version

See [`performance.md`](performance.md) for bench and timing tools used to
validate warm-cache behavior.

## Limitations (v1)

- Git remote sources require prior materialization or a pinned local mirror.
- Advisory data: copy `<cache>/advisory/osv.json` separately (see [`audit.md`](audit.md)).
- Capsule archives are uncompressed tar (gzip deferred).
- No `m install --preflight-offline` standalone subcommand; preflight is inline
  on `--offline` install failure.

## Related

- [`install.md`](install.md) — install pipeline and flags
- [`fetch.md`](fetch.md) — tarball offline behavior
- [`registry.md`](registry.md) — metadata cache offline behavior
- [`store.md`](store.md) — content store layout
