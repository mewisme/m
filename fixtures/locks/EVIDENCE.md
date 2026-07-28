# Lock bridge evidence — Stabilization Pass 16

**Fetched:** 2026-07-28  
**Baseline:** `be85cf2a582af01c468b38772f57e3eee02e80be`

## pnpm references

| Major | Pinned version | Docs | Releases |
|-------|----------------|------|----------|
| 9 | 9.15.9 | https://pnpm.io/9.x | https://github.com/pnpm/pnpm/releases |
| 10 | 10.14.0 | https://pnpm.io/10.x | https://github.com/pnpm/pnpm/releases |
| 11 | 11.0.2 | https://pnpm.io/11.x | https://github.com/pnpm/pnpm/releases |

Generation: `pwsh tools/conformance/generate-lock-fixtures.ps1 -Generate`  
Pins: `tools/conformance/pnpm-versions.env`  
Families per major: `basic`, `transitive`, `optional`, `peer-context`, `multi-version`, `scoped`, `workspace`, `catalog`, `override`, `platform`, `importer-meta`.

Each `fixtures/locks/generated/pnpm-{9,10,11}/{family}/metadata.json` records the exact `corepack prepare` + `pnpm install --lockfile-only` command.

Legacy v5–v8 locks live under `fixtures/locks/pnpm/unsupported/` for rejection tests only.

## Nub references

| Item | Source |
|------|--------|
| Lock layout | pnpm v9-shaped YAML in `nub.lock` |
| Site | https://nubjs.com |
| Fixtures | `fixtures/locks/generated/nub-{basic,transitive,workspace,catalog,peer,optional}/` derived from pnpm-9 binary output + `nubVersion` marker |

Nub adapter selects pnpm encode policy from incumbent `nub.lock` bytes via `DetectPnpmWithContext`, not a hardcoded v9 writer.

## Detection policy (Pass 16)

1. `package.json` `packageManager` (majors 9/10/11 only; ranges/tags rejected)
2. `devEngines.packageManager`
3. `--pnpm-major` / `InstallOptions.PnpmMajor`
4. Adapter-recorded metadata in lock extensions
5. Generation-specific structural evidence (policy-owned)
6. Else ambiguous (`DetectionInferred`, fail closed on write)

Manifest vs flag conflict → `DetectionConflictError`.

## Conformance

CI jobs: `conformance-pnpm-{9,10,11}` (parse + mutation + frozen), `conformance-pnpm-unsupported`, `conformance-nub-fixtures`.

Mutation suite: Mew install txn add → lock bytes change → `m lock validate --frozen` → pnpm `--frozen-lockfile --lockfile-only` accepts Mew-written lock → repeat deterministic → commit interrupt restores incumbent.
