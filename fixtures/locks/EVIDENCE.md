# Lock bridge evidence — Stabilization Pass 15

**Fetched:** 2026-07-28  
**Baseline:** `e79cebf0c8be7e3e3e1734dbb181fb29e5e8e40e`

## pnpm references

| Major | Pinned version | Docs | Releases |
|-------|----------------|------|----------|
| 9 | 9.15.9 | https://pnpm.io/9.x | https://github.com/pnpm/pnpm/releases |
| 10 | 10.14.0 | https://pnpm.io/10.x | https://github.com/pnpm/pnpm/releases |
| 11 | 11.0.2 | https://pnpm.io/11.x | https://github.com/pnpm/pnpm/releases |

Generation command per family: see `fixtures/locks/generated/pnpm-*/basic/metadata.json`.

Committed fixtures under `fixtures/locks/generated/` were seeded from
`fixtures/locks/pnpm/v{9,10,11}` with honest metadata (not hand-edited YAML).
Re-run `tools/conformance/generate-lock-fixtures.ps1 -Generate` when pnpm is
available to refresh from live binaries.

## Nub references

| Item | Source |
|------|--------|
| Lock layout | pnpm v9-shaped YAML in `nub.lock` |
| Site | https://nubjs.com |
| Fixture | `fixtures/locks/generated/nub-basic/` (manual evidence; generation not automatable in CI) |

Nub adapter delegates encode policy to pnpm detection inferred from incumbent
`nub.lock` bytes (not a hardcoded v9 writer).

## Detection policy (Pass 15)

1. `package.json` `packageManager`
2. `devEngines.packageManager`
3. `--pnpm-major` / `InstallOptions.PnpmMajor`
4. Adapter-recorded metadata in lock extensions
5. Generation-specific structural evidence (policy-owned)
6. Else ambiguous (`DetectionInferred`, fail closed on write)

Common settings keys (`patchedDependencies`, `configDependencies`,
`onlyBuiltDependencies`) alone are **not** sufficient for certain detection
when shared across generations; v10/v11 fixtures include generation-specific
root/settings markers.

## Conformance

CI jobs `conformance-pnpm-{9,10,11}` and `conformance-nub-fixtures` run
`tests/conformance/lock_bridge_*_test.go` against `fixtures/locks/generated/`.
