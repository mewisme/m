# Yarn lockfile (`yarn.lock`)

Mew supports Yarn Classic (v1) and Yarn Berry (v2+) node-modules locks. Berry PnP is parse-only for validate/migrate; install is blocked.

## Variants

| Variant | Detection | Install | Write |
|---------|-----------|---------|-------|
| Classic | No `__metadata` in lock | Shipped (preserve-only) | Preserve-only no-op |
| Berry node-modules | `__metadata` + `nodeLinker: node-modules` or no `.pnp.cjs` | Shipped (preserve-only) | Preserve-only no-op |
| Berry PnP | `__metadata` + `nodeLinker: pnp` or `.pnp.cjs` present | **Blocked** (`ERR_M_PNP_UNSUPPORTED`) | Parse-only |

## Classic write policy

Yarn Classic supports read and byte-preserving no-op `EncodePreserving` only. Graph-changing mutations return `ERR_M_LOCK_UNREPRESENTABLE` until Yarn immutable CI certifies write.

## Berry PnP install gate

`m install` on a Yarn Berry PnP project fails before fetch/link:

```
ERR_M_PNP_UNSUPPORTED: Yarn Berry PnP install is not supported; use node-modules linker or migrate to m.lock
```

`m lock validate` and `m lock migrate --from yarn` still work on PnP fixtures.

## Migration

```bash
m lock migrate --from yarn --to m
```

## Deferred (MVP 0025)

- Yarn classic mutation write (awaiting Yarn immutable CI)
- Berry PnP install / runtime (MVP 0053)
- Zero-install cache extraction (metadata read-only)
- Differential `yarn install` CI

## Capability level (MVP 0025)

| Capability | Classic | Berry NM | Berry PnP |
|------------|---------|----------|-----------|
| Parse | Shipped | Shipped | Shipped |
| Graph conversion | Shipped | Shipped | Shipped |
| Install | Shipped | Shipped | Blocked |
| Migrate to m.lock | Shipped | Shipped | Shipped |
