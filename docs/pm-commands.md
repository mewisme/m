# Package-manager commands

Index of shipped PM commands (MVPs **0026–0030**) and stabilization surfaces
(MVP **0031**). Install-family basics remain in [`install.md`](install.md).

Certification evidence: [`core-certification.md`](core-certification.md).

## Command index

| Command | MVP | Mutates disk | Doc |
|---|---|---|---|
| **Install family** | | | |
| `m install` / `i` | 0016 | yes | [`install.md`](install.md) |
| `m add` | 0016 | yes | [`install.md`](install.md) |
| `m remove` / `rm` | 0016 | yes | [`install.md`](install.md) |
| `m update` | 0016 | yes | [`install.md`](install.md) |
| `m ci` | 0026 | yes | below |
| `m dedupe` | 0026 | yes | below |
| `m prune` | 0026 | yes | below |
| **Query / explain** | | | |
| `m outdated` | 0026 | no | below |
| `m ls` / `m list` | 0026 | no | below |
| `m explain` | 0028 | no | [`explain.md`](explain.md) |
| `m resolve` | 0013 | no | [`resolver.md`](resolver.md) |
| `m plan` | 0028 | no | [`plan.md`](plan.md) |
| `m plan update` | 0028 | no | [`plan.md`](plan.md) |
| `m history` | 0028 | no | [`plan.md`](plan.md) |
| `m diff lock` / `m lock diff` | 0028 | no | [`explain.md`](explain.md) |
| **Lockfile** | | | |
| `m lock validate` | 0015 | no | [`lockfile.md`](lockfile.md) |
| `m lock format` | 0015 | yes | [`lockfile.md`](lockfile.md) |
| `m lock migrate` | 0023 | yes | [`lockfile.md`](lockfile.md) |
| **Fetch / pack / publish** | | | |
| `m fetch` | 0027 | cache only | [`fetch.md`](fetch.md) |
| `m pack` | 0027 | no (writes tarball) | [`pack-publish.md`](pack-publish.md) |
| `m publish` | 0027 | no | [`pack-publish.md`](pack-publish.md) |
| `m patch` | 0027 | yes | [`patch.md`](patch.md) |
| **Snapshot / recovery** | | | |
| `m snapshot list` | 0017 | no | [`transaction.md`](transaction.md) |
| `m snapshot restore` | 0017 | yes | [`transaction.md`](transaction.md) |
| `m recover` | 0017 | yes | [`transaction.md`](transaction.md) |
| `m rollback` | 0017 | yes | [`transaction.md`](transaction.md) |
| **Store / cache / config** | | | |
| `m store` | 0012 | varies | [`store.md`](store.md) |
| `m cache` | 0012 | varies | [`registry.md`](registry.md) |
| `m config` | 0006 | yes (set) | [`config.md`](config.md) |
| `m registry view` | 0012 | no | [`registry.md`](registry.md) |
| **Lifecycle trust** | | | |
| `m trust` / `m approve-builds` | 0021 | yes | [`lifecycle.md`](lifecycle.md) |
| **Capsule / bench** | | | |
| `m capsule create` | 0029 | yes | [`offline.md`](offline.md) |
| `m capsule restore` | 0029 | yes | [`offline.md`](offline.md) |
| `m bench install` | 0029 | yes (fixture) | [`performance.md`](performance.md) |
| `m benchmark install` | 0029 | yes (fixture) | alias of `m bench` (0031) |
| **Security** | | | |
| `m audit` | 0030 | no | [`audit.md`](audit.md) |
| `m sbom` | 0030 | no | [`sbom.md`](sbom.md) |
| `m verify provenance` | 0030 | no | [`pack-publish.md`](pack-publish.md) |
| `m policy check` | 0030 | no | [`policy.md`](policy.md) |
| **Stabilization (0031)** | | | |
| `m doctor` | 0031 | no | [`core-certification.md`](core-certification.md) |
| `m conformance list` | 0031 | no | [`core-certification.md`](core-certification.md) |
| `m conformance run core` | 0031 | no | [`core-certification.md`](core-certification.md) |

`mx` exposes executor commands only (no PM surface).

## MVP 0026 — maintenance commands

Read-only and maintenance commands alongside the install family.

### Command summary

| Command | Mutates disk | Transaction |
|---|---|---|
| `m ci` | yes | yes |
| `m outdated` | no | — |
| `m ls` / `m list` | no | — |
| `m dedupe` | yes | yes |
| `m prune` | yes | yes |

### `m ci`

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

### `m outdated`

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

### `m ls` / `m list`

| Mode | Trigger | Output |
|---|---|---|
| Dependency tree (default) | no `-r` | Lock graph tree for root importer |
| Workspace members | `-r` + workspaces gate | Member name, version, path |

Tree flags: `--depth N` (default unlimited), `--prod`, `--json`.

### `m dedupe`

Re-resolve and collapse duplicate package names in the lock where semver ranges
allow consolidation, then relink via the install transaction.

| Flag | Notes |
|---|---|
| `--dry-run` | Plan only; no disk mutation |
| `--prod` | Omit devDependencies |
| `--json` | Install result JSON |

v1 uses a name-grouping heuristic (see `ponytail:` comment in
`internal/resolver/dedupe.go`); full npm dedupe parity is not guaranteed.

### `m prune`

Remove extraneous packages under `node_modules` that are not expected from the
lock + linker plan. Distinct from `m store prune` (global content store).

| Flag | Notes |
|---|---|
| `--prod` | Ignore dev-only extraneous paths |
| `--dry-run` | List removals only |
| `--json` | Install result JSON |

## MVP 0027 — fetch, pack, publish, patch

See [`fetch.md`](fetch.md), [`pack-publish.md`](pack-publish.md), [`patch.md`](patch.md),
[`sources.md`](sources.md).

| Command | Summary |
|---|---|
| `m fetch <pkg>` | Download tarball to cache without install |
| `m pack` | Create reproducible `name-version.tgz` |
| `m publish` | Upload to registry (`--dry-run`, `--provenance`) |
| `m patch <pkg>` | Extract for edit or commit patch to lock |

## MVP 0028 — plan, explain, history, diff

See [`plan.md`](plan.md), [`explain.md`](explain.md).

| Command | Mutates disk | Notes |
|---|---|---|
| `m plan` | no | Install preview; mirrors `m install --dry-run` |
| `m plan update` | no | Update preview |
| `m history` | no | Snapshot timeline (newest first) |
| `m diff lock` / `m lock diff` | no | Semantic lock graph diff |
| `m explain [name]` | no | Resolution trace |
| `m explain peer <name>` | no | Peer conflict tree |

## MVP 0029 — bench and capsule

See [`performance.md`](performance.md), [`offline.md`](offline.md).

```text
m bench install [--cold|--warm] [--fixture <path>] [--json]
m capsule create [--output <path>]
m capsule restore <path>
```

## MVP 0030 — audit, SBOM, provenance, policy

See [`audit.md`](audit.md), [`sbom.md`](sbom.md), [`policy.md`](policy.md).

```text
m audit [--json] [--fix]
m sbom [--format cyclonedx|spdx] [--redact-internal]
m verify provenance [<pkg>]
m policy check [--json]
```

## MVP 0031 — stabilization

```text
m doctor [--json] [--strict]
m conformance list
m conformance run core [--json] [--filter <id>]
make core-cert
```

Contributor tooling (`m development doctor`) remains separate; see
[`development-doctor.md`](development-doctor.md).

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
| `m benchmark` | `m bench` (0031 compatibility surface) |

## Fixtures

| Fixture | Purpose |
|---|---|
| `fixtures/projects/ci-clean-install` | clean ci + extraneous removal |
| `fixtures/projects/outdated-tree` | outdated JSON report |
| `fixtures/projects/dedupe-duplicates` | dedupe lock reduction |
| `fixtures/bench/medium-graph` | bench + SBOM golden |
| `fixtures/soak/representative-projects/` | soak install loop corpus |

See [`testing.md`](testing.md) for integration test locations.
