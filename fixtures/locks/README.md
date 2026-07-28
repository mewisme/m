# Lock fixture corpus (MVP 0023)

Reference lockfiles for Nub and pnpm adapter conformance. Each fixture includes
`metadata.json` with `producerMajor` and a pinned `pnpmVersion` for CI.

## pnpm v6 vs v9/10/11

| Aspect | v6 (`lockfileVersion` 5.x / 6.0) | v9–v11 (`lockfileVersion` 9.0) |
|---|---|---|
| Layout | Flat `packages` keyed by `/name/version`; root `dependencies` + `specifiers` | `importers`, `packages`, `snapshots` sections |
| Importers | None | Per-workspace importer dependency specifiers |
| Snapshots | None | Per-package snapshot metadata |
| Detection | `lockfileVersion` prefix + absence of `importers`/`snapshots` | Shared 9.0 marker; **do not trust version alone** |

### Producer-major hints (9.0-shaped locks)

pnpm 9, 10, and 11 share `lockfileVersion: '9.0'`. Distinguish via field presence:

| Generation | Heuristic markers (fixture + runtime detection) |
|---|---|
| v9 | 9.0 shape without v10/v11-only root or settings keys |
| v10 | Root `patchedDependencies` or `configDependencies`; package checksum extensions |
| v11 | `settings.onlyBuiltDependencies` or `settings.ignoredBuiltDependencies` (build-policy) |

Ambiguous 9.0-shaped locks without resolvable markers require explicit
`--pnpm-major` before encode/write (fail closed otherwise).

## Generated corpus (`generated/`)

Binary-generated (or honestly metadata-tagged committed) families live under
`fixtures/locks/generated/` with full `metadata.json` per family. Refresh via
`tools/conformance/generate-lock-fixtures.ps1` (use `-Generate` when pnpm is
available). Evidence log: `fixtures/locks/EVIDENCE.md`.

CI conformance: `conformance-pnpm-{9,10,11}`, `conformance-nub-fixtures`.

## Nub

`nub.lock` uses the pnpm v9-shaped YAML layout; identity and filename differ only.
