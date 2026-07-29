# npm lockfile compatibility

Mew reads and writes npm `package-lock.json` and `npm-shrinkwrap.json` for npm-identity projects while preserving project identity (no silent conversion to `m.lock`).

## Supported versions

| Version | Status | Notes |
|---------|--------|-------|
| **v2** | Supported | `packages` map + optional legacy `dependencies` tree |
| **v3** | Supported | Workspaces via workspace package paths and `link` entries |
| **v1** | **Rejected** | `ERR_M_LOCK_UNSUPPORTED` — regenerate with npm 7+ |

Forward-unknown majors (for example v4+) fail closed with `ERR_M_LOCK_UNSUPPORTED`.

## Incumbent file precedence

When both exist, **`npm-shrinkwrap.json` wins** over `package-lock.json` for read, write, install staging, and validation. Encode writes back to the path that was read.

## Write policy

| Scenario | `lockfileVersion` |
|----------|-------------------|
| Greenfield (no prior lock) | **3** |
| Mutation (prior lock exists) | Preserve incumbent **2** or **3** |

Semantic compatibility is guaranteed; byte-identical formatting is not.

## Graph mapping

- Root importer: `packages[""]`
- Workspace importers: non-`node_modules/` paths with a version (for example `packages/pkg-a`)
- Registry packages: `name@version` from `packages` entries with `resolved` / `integrity`
- Workspace links: `link: true` entries preserved on encode; not expanded into separate graph packages
- `bundledDependencies`: preserved in lock encode; bundle expansion at resolve time is deferred

## Install integration

- npm identity uses the **hoisted** linker when `install.linker=auto` (including workspaces)
- First install with no lock generates `package-lock.json` v3
- Incumbent lock writes go through the install transaction (same as pnpm/nub)

## CLI

```bash
m install                  # preserves package-lock or shrinkwrap
m lock validate --frozen   # manifest drift → ERR_M_LOCKFILE
m lock migrate --from npm --to m [--dry-run]
```

## Fixtures

| Path | Purpose |
|------|---------|
| `fixtures/locks/npm/v2-basic` | Simple prod dependency, lockfileVersion 2 |
| `fixtures/locks/npm/v3-workspaces` | Root + workspace package, lockfileVersion 3 |
| `fixtures/projects/npm-app` | End-to-end install project |
| `testdata/lockfile/npm-roundtrip/` | Golden decode/encode pairs |

## CI

`conformance-npm` job runs `go test ./tests/conformance/... -run Npm`.

## Intentional limits

- No package-lock v1 read/write/migrate
- No byte-identical npm formatting guarantee
- Full `bundledDependencies` graph expansion deferred
- Bun/Yarn locks: MVP 0025
