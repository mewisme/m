# Bun lockfile (`bun.lock`)

Mew reads text `bun.lock` files for Bun-identity projects. Binary `bun.lockb` is rejected.

## Supported format

- Text JSONC lockfiles produced by Bun 1.2+ (`lockfileVersion` 0 or 1).
- Packages map with resolution tuples: `[resolution, registry, info, integrity]`.
- Workspace importers under `workspaces`.

## Rejected inputs

| Input | Behavior |
|-------|----------|
| `bun.lockb` (binary) | `ERR_M_LOCK_UNSUPPORTED` — convert with `bun install --save-text-lockfile --frozen-lockfile --lockfile-only` |
| Unsupported `lockfileVersion` | `ERR_M_LOCK_UNSUPPORTED` |

## Install behavior

- Bun-identity projects preserve incumbent `bun.lock` bytes on graph-equal installs.
- Graph-changing `bun.lock` mutation is not supported in MVP 0025.
- Mew never silently converts `bun.lock` to `m.lock` on install.

## Migration

```bash
m lock migrate --from bun
```

Lossy fields are reported in the migration loss report.

## Capability level (MVP 0025)

| Capability | Status |
|------------|--------|
| Parse / validate | Shipped |
| Graph conversion | Shipped |
| Byte-preserving no-op encode | Shipped |
| Graph-changing write | Deferred |
| `bun install` differential CI | Deferred |
