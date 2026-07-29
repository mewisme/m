# Package-manager commands (MVP 0026)

Read-only and maintenance commands that complete the core `m` grammar alongside
the install family ([`install.md`](install.md)).

## Command summary

| Command | Mutates disk | Transaction |
|---|---|---|
| `m ci` | yes | yes |
| `m outdated` | no | — |
| `m ls` / `m list` | no | — |
| `m dedupe` | yes | yes |
| `m prune` | yes | yes |

`mx` does not expose these commands (executor-only surface).

## `m ci`

Clean install from the incumbent lockfile (npm `npm ci` parity).

1. Validate manifest specifiers match the lock (`ERR_M_LOCKFILE` on drift).
2. Remove the live `node_modules` tree (and isolated `.pnpm` when present).
3. Install from the lock without manifest-driven re-resolve.

| Flag | Notes |
|---|---|
| `--prod` | Omit devDependencies |
| `--linker` | `hoisted` or `isolated` |
| `--ignore-scripts` | Skip lifecycle scripts |
| `--json` | Install result JSON |
| `--frozen-lockfile` | Accepted alias (ci is always frozen) |

Not supported: `--dry-run`, `--filter`.

## `m outdated`

Read-only version drift report from the lock graph vs registry metadata.

| Flag | Notes |
|---|---|
| `-r` / `--recursive` | All workspace importers (workspaces gate required) |
| `--json` | JSON array of `{package, current, wanted, latest, ...}` |
| `--filter` | Limit importers (global flag; workspaces gate) |

Respects global `--offline` / `--prefer-offline`. Without cached metadata,
offline mode returns `ERR_M_NETWORK`.

`m why` (npm parity name) is not implemented; use `m explain <pkg>` or
`m resolve --trace`. See [`explain.md`](explain.md).

## `m plan` and `m history` (0028)

| Command | Mutates disk | Notes |
|---|---|---|
| `m plan` | no | Install preview; mirrors `m install --dry-run` |
| `m plan update` | no | Update preview |
| `m history` | no | Snapshot timeline (newest first) |
| `m diff lock` / `m lock diff` | no | Semantic lock graph diff |

See [`plan.md`](plan.md) and [`explain.md`](explain.md).

## `m ls` / `m list`

| Mode | Trigger | Output |
|---|---|---|
| Dependency tree (default) | no `-r` | Lock graph tree for root importer |
| Workspace members | `-r` + workspaces gate | Member name, version, path |

Tree flags: `--depth N` (default unlimited), `--prod`, `--json`.

## `m dedupe`

Re-resolve and collapse duplicate package names in the lock where semver ranges
allow consolidation, then relink via the install transaction.

| Flag | Notes |
|---|---|
| `--dry-run` | Plan only; no disk mutation |
| `--prod` | Omit devDependencies |
| `--json` | Install result JSON |

v1 uses a name-grouping heuristic (see `ponytail:` comment in
`internal/resolver/dedupe.go`); full npm dedupe parity is not guaranteed.

## `m prune`

Remove extraneous packages under `node_modules` that are not expected from the
lock + linker plan. Distinct from `m store prune` (global content store).

| Flag | Notes |
|---|---|
| `--prod` | Ignore dev-only extraneous paths |
| `--dry-run` | List removals only |
| `--json` | Install result JSON |

## Mew vs npm / pnpm (intentional differences)

| Area | npm / pnpm | Mew |
|---|---|---|
| `ci` | Removes `node_modules` | Same |
| `ci --dry-run` | unsupported | `ERR_M_USAGE` |
| `ci --filter` | pnpm supports filtered ci in some versions | `ERR_M_USAGE` (full tree) |
| `ls` default | dependency tree | dependency tree; `-r` switches to workspace list when workspaces enabled |
| `prune` | `node_modules` extraneous | `m prune` = node_modules; `m store prune` = global store |
| `dedupe` | full tree rewrite | lock-centric heuristic v1 |
| `outdated` | direct deps default | direct deps per importer; `-r` for all importers |
| `why` | built-in | use `m explain` (name deferred) |
| `plan` | built-in | shipped (0028) |
| `history` | built-in | shipped (0028) |
| `explain` | built-in | shipped (0028) |
| `import` / `rebuild` | shipped | deferred (not 0026) |

## Flag aliases

| Alias | Canonical |
|---|---|
| `m ci --frozen-lockfile` | always frozen (no-op confirmation) |
| `m install --frozen-lockfile` | `InstallOptions.Frozen` |

## Fixtures

| Fixture | Purpose |
|---|---|
| `fixtures/projects/ci-clean-install` | clean ci + extraneous removal |
| `fixtures/projects/outdated-tree` | outdated JSON report |
| `fixtures/projects/dedupe-duplicates` | dedupe lock reduction |

See [`testing.md`](testing.md) for integration test locations.
