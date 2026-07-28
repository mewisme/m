# Workspaces

Mew supports npm-style `workspaces` in `package.json` plus pnpm-inspired catalogs and
`--filter` selection. Multi-importer install is behind an experimental gate.

## Enable workspaces

Set either:

- `MEW_EXPERIMENTAL_WORKSPACES=1`
- `workspaces.enabled: true` in `m.jsonc`

Isolated linking is selected automatically for workspace projects when
`install.linker` is `auto` (requires `MEW_EXPERIMENTAL_ISOLATED_LINKER=1` today).

## Commands

| Command | Description |
|---------|-------------|
| `m install -r` | Install all workspace members in one transaction |
| `m install --filter <pattern>` | Install matched members (+ closure semantics) |
| `m --filter <pattern> install` | Global filter (pnpm-style) |
| `m add <pkg> --filter <pattern>` | Add a dependency scoped to filtered importers |
| `m remove <pkg> --filter <pattern>` | Remove a dependency from filtered members only |
| `m ls` / `m list` | List workspace packages |
| `m ls -r` | List all members (requires workspaces gate) |

Default `m install` resolves and installs **only the root importer**. Workspace
protocol dependencies (`workspace:*`, `workspace:^`) still resolve from the root
graph. Use `-r` to register every member as an importer.

## Catalogs

Root `package.json` may define a default catalog:

```json
{
  "catalog": {
    "react": "^18.2.0",
    "lodash": "4.17.21"
  }
}
```

Members may use `catalog:` (key = dependency name), `catalog:default`, or
`catalog:<entry>` specifiers. Undefined catalog entries fail with `ERR_M_MANIFEST`.

Optional `pnpm-workspace.yaml` at the project root may add or override catalog
entries under a `catalog:` block (merged over `package.json`).

## Filter grammar (v1)

| Pattern | Meaning |
|---------|---------|
| `pkg` | Exact package name |
| `@scope/pkg` | Scoped name |
| `packages/*` | Path glob / prefix |
| `{apps/*,libs/*}` | Brace expansion |
| `...pkg` | Package + workspace dependency closure |
| `pkg...` | Package + workspace dependents |
| `!pkg` | Negation |

**Deferred:** changed-since selectors (`[origin/main]`) — not implemented in v1.

## Lockfile

`m.lock` stores one importer section per workspace member when `-r` or `--filter`
install runs. Filtered installs merge untouched importer sections from the prior
lock so unrelated members are not dropped.

Filtered installs also merge the **directed package closure** (`From → To` edges only)
for untouched importers from the prior lock graph before fetch/link. This includes
**package-to-package** edges inside each untouched member's subgraph (for example
`pkg-b@1.2.0 → pkg-c` when filtering `alpha`). Shared transitive dependencies
owned by an untouched member are preserved; beta-only packages are not dropped.

`m remove <pkg> --filter <pattern>` mirrors filtered add: member `package.json`
files are edited in memory, staged under `stage/`, and committed atomically with
the lock. Root `package.json` is unchanged unless `.` is in the filter.

`m update --filter` and `m ci --filter` return `ERR_M_USAGE` (filtered update
deferred; `ci` is always a frozen full-tree install).

## Filtered add

`m add <pkg> --filter <pattern>` edits matched member `package.json` files in
memory, stages them under the transaction `stage/` tree, and commits atomically
with the lock and `node_modules`. Root `package.json` is unchanged unless `.` is
in the filter match set.

## Out of scope (v1)

- `file:`, `link:`, and `portal:` install (resolve-only)
- Per-member `node_modules` trees at member paths (root virtual store only)
- Workspace script orchestration (`mx` scheduler — MVP 0041)
