# npm lockfile compatibility

Mew reads npm `package-lock.json` and `npm-shrinkwrap.json` for npm-identity projects while preserving project identity (no silent conversion to `m.lock`). **Semantic lock mutation is not supported** — incumbent locks are read-only except for byte-preserving no-ops when the resolved graph is unchanged.

## Supported versions

| Version | Status | Notes |
|---------|--------|-------|
| **v2** | Read + validate | `packages` map + optional legacy `dependencies` tree |
| **v3** | Read + validate | Workspaces via workspace package paths and `link` entries |
| **v1** | **Rejected** | `ERR_M_LOCK_UNSUPPORTED` — regenerate with npm 7+ |

Forward-unknown majors (for example v4+) fail closed with `ERR_M_LOCK_UNSUPPORTED`.

## Incumbent file precedence

When both exist, **`npm-shrinkwrap.json` wins** over `package-lock.json` for read, install staging, and validation. Mew does not rewrite incumbent npm locks.

## Write policy

| Scenario | Behavior |
|----------|----------|
| Parse / validate / inspect | Supported |
| Graph-equal incumbent (`EncodePreserving` no-op) | Byte-preserving |
| Frozen `m ci` / `m install --frozen-lockfile` | Supported when graph matches |
| Semantic mutation (add / update / remove / drift rewrite) | **`ERR_M_UNSUPPORTED`** |
| Greenfield (no prior lock) | Generates `package-lock.json` v3 |

Semantic compatibility is guaranteed on read paths; byte-identical formatting is not.

## Graph mapping

- Root importer: `packages[""]`
- Workspace importers: non-`node_modules/` paths with a version (for example `packages/pkg-a`)
- Registry packages: `name@version` from `packages` entries with `resolved` / `integrity`
- Workspace links: `link: true` entries preserved on encode; not expanded into separate graph packages
- `bundledDependencies`: preserved in lock encode; bundle expansion at resolve time is deferred

## Install integration

- npm identity uses the **hoisted** linker when `install.linker=auto` (including workspaces)
- First install with no lock generates `package-lock.json` v3
- Incumbent lock rewrites are rejected with `ERR_M_UNSUPPORTED`; use npm or migrate to `m.lock`

## CLI

```bash
m install                  # preserves package-lock or shrinkwrap
m lock validate --frozen   # manifest drift → ERR_M_LOCKFILE
m lock migrate --from npm [--dry-run]
```

## Fixtures

| Path | Purpose |
|------|---------|
| `fixtures/locks/npm/v2-basic` | Simple prod dependency, lockfileVersion 2 |
| `fixtures/locks/npm/v3-workspaces` | Root + workspace package, lockfileVersion 3 |
| `fixtures/projects/npm-app` | End-to-end install project |
| `testdata/lockfile/npm-roundtrip/` | Golden decode/encode pairs |

## Conformance evidence

| Test | Covers |
|------|--------|
| `TestLockBridgeNpm` | Parse, graph conversion, byte-preserving no-op |
| `TestLockBridgeNpmMutationRejected` | `m add` on npm-lock fixture → `ERR_M_UNSUPPORTED` |

## CI

`conformance-npm` job runs `go test ./tests/conformance/... -run Npm`.

## Intentional limits

- No package-lock v1 read/write/migrate
- No semantic incumbent `package-lock.json` / `npm-shrinkwrap.json` mutation (`ERR_M_UNSUPPORTED`)
- No byte-identical npm formatting guarantee
- Full `bundledDependencies` graph expansion deferred
- Bun/Yarn locks: MVP 0025
