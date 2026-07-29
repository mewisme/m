# Lock bridge evidence — Stabilization Pass 18

**Fetched:** 2026-07-29  
**Baseline:** `d9f5dfc457a7a86547d5798336c08769983c4e38`

## pnpm references

| Major | Pinned version | Docs | Releases |
|-------|----------------|------|----------|
| 9 | 9.15.9 | https://pnpm.io/9.x | https://github.com/pnpm/pnpm/releases |
| 10 | 10.34.5 | https://pnpm.io/10.x | https://github.com/pnpm/pnpm/releases |
| 11 | 11.17.0 | https://pnpm.io/11.x | https://github.com/pnpm/pnpm/releases |

Pins resolved at generation time via `npm view pnpm@{9,10,11} version`.  
Generation: `pwsh tools/conformance/generate-lock-fixtures.ps1 -Generate`  
Verification: `go run ./tools/conformance/verify-fixtures` (CI job `fixture-verify`)  
Pins file: `tools/conformance/pnpm-versions.env` (CI sources versions from here, not hardcoded YAML)

Families per major: `basic`, `transitive`, `optional`, `peer-context`, `multi-version`, `scoped`, `workspace`, `catalog`, `override`, `platform`, `importer-meta`, `alias`, `patch`, `binary`.

`transitive` uses `chalk@4.1.2` so snapshot edges include real transitive deps (`ansi-styles`, etc.).

Each `fixtures/locks/generated/pnpm-{9,10,11}/{family}/metadata.json` records the exact `corepack prepare` + `pnpm install --lockfile-only` command and SHA-256 of the committed lockfile.

Legacy v5–v8 locks live under `fixtures/locks/pnpm/unsupported/` for rejection tests only.

## Nub references

| Item | Source |
|------|--------|
| Lock layout | pnpm v9-shaped YAML in `nub.lock` |
| Site | https://nubjs.com (fetched 2026-07-29) |
| Upstream engine | aube (Rust resolver/linker per nubjs.com docs) |
| Frozen behavior | `nub install --frozen-lockfile` documented as pnpm-flag-compatible |
| Fixtures | `fixtures/locks/generated/nub-{basic,transitive,workspace,catalog,peer,optional}/` |

Nub adapter selects pnpm encode policy from incumbent `nub.lock` bytes via `DetectPnpmWithContext`, not a hardcoded v9 writer.

### Evidence levels

| Level | Families | What CI proves |
|-------|----------|----------------|
| **Derived-format** | all six | Metadata + lock bytes; `derived` classification in metadata |
| **Parse + validate** | `nub-basic`, `nub-transitive`, `nub-workspace`, `nub-catalog`, `nub-peer`, `nub-optional` | `ReadWithExtensions` + `m lock validate --json` (workspace no longer skipped) |
| **Executable (deferred)** | all | `conformance-nub-exec` not wired — requires `nub` binary in CI; residual risk documented |

## Pass 18 changes

- Snapshot-primary instances: `packages` metadata only; graph nodes from `snapshots` keys
- npm alias edges resolve by actual package name; encode round-trip via `lodash@x.y.z` version refs
- `packageManager` semver strict via Masterminds/semver (`pnpm@10.not-a-semver` rejected)
- Mutation matrix: 7 families × 3 majors with frozen after add/update/**remove**
- Fixture provenance: verify-fixtures rejects placeholder commands; generation only with `-Generate`
- `Adapter.Write` / fresh encode fail closed without certified `--pnpm-major`

1. `package.json` `packageManager` (exact majors 9/10/11 only; bare `pnpm` → no major evidence)
2. `devEngines.packageManager`
3. `--pnpm-major` / `InstallOptions.PnpmMajor`
4. Adapter-recorded metadata in lock extensions
5. Structural evidence at `DetectionInferred` at best (no certain major from lock shape alone)
6. Else ambiguous (`DetectionInferred`, fail closed on write)

Manifest vs flag conflict → `DetectionConflictError`.

## Conformance

CI jobs: `fixture-verify`, `conformance-pnpm-{9,10,11}` (parse + `MutationSuite`), `conformance-pnpm-unsupported`, `conformance-nub-fixtures`.

Mutation suite (per major × `basic|transitive|optional|peer-context|workspace`):

1. Mew add/update/remove via install txn
2. `m lock validate --frozen`
3. Pinned pnpm `install --frozen-lockfile --ignore-scripts` (full install, **no** `--lockfile-only`) — strict byte hash
4. `node_modules` + Node `require()` import script
5. Repeat Mew frozen install → deterministic bytes
6. Commit-interrupt → incumbent bytes restored

All non-race CI jobs run with `CGO_ENABLED=0`.
