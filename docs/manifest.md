# Manifest and project discovery

MVP **0011**. Read and edit `package.json` without destructive reformatting, discover
project roots, and expand workspace globs.

## Discovery

`project.FindRoot(cwd)` walks up until it finds `package.json`. Missing root →
`ERR_M_NOT_FOUND`.

`project.Open` / `OpenAt` load identity ([`identity.md`](identity.md)), parse the
manifest, validate, and expose a normalized dependency list.

## package.json codec

- **Strict JSON only** (no json5 / JSONC).
- Duplicate keys are rejected (`ERR_M_MANIFEST`).
- Load keeps original `Source` bytes.
- Field updates **splice** JSON values into `Source` (preserve key order, indent,
  trailing newline). Dependency-object rewrites may reorder keys inside that object.
- `Write` uses temp file + rename (best-effort `Sync`).

### Normalized view

`manifest.ToNormalized` flattens `dependencies` / `devDependencies` /
`optionalDependencies` / `peerDependencies` into sorted
`manifest.Manifest` / `Dependency` (0007 resolver shape). Scoped names like
`@scope/pkg` are preserved.

### Validation

When present: lowercase npm package `name`, non-empty whitespace-free `version`,
`bin` as string or string map. Failures → `ERR_M_MANIFEST`.

### Cache

`LoadCached(dir)` keys by absolute dir + mtime. `Invalidate(root)` is the hook for
a future file watcher (no watcher in 0011).

## Workspaces

Only `package.json` `workspaces` (string array or `{ "packages": [...] }`).
Root `catalog` maps catalog entry names to version ranges (`catalog:` specifiers).
Optional `pnpm-workspace.yaml` `catalog:` block merges over `package.json` catalog.

See [`workspaces.md`](workspaces.md) for install, filters, and catalogs.

Globs: path segments, `*`, `{a,b}` braces, `!` negation. Members are root-relative
slash paths, sorted, each requiring its own `package.json`. Cyclic / escaping
member `workspaces` → `ERR_M_MANIFEST`.

## CLI

```text
m project info [--json]
m pkg get <field> [--json]
```

Fields: `name`, `version`, `private`, `packageManager`, dependency maps, `scripts`,
`engines`, `workspaces`, `bin`.

`m init` remains an unimplemented stub until MVP **0070**.

## Errors

| Code | Use |
|---|---|
| `ERR_M_NOT_FOUND` | No `package.json` / project root |
| `ERR_M_MANIFEST` | Parse, validate, workspaces, cycles |
| `ERR_M_IO` | Filesystem failures |
