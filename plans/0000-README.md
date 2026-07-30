# Mew Implementation Plan

This archive is the implementation program for **MewJS** (abbreviated **Mew**), a Go-based JavaScript toolchain and package manager. The primary binaries are **`m`** (alias **`mew`**) and **`mx`** (alias **`mewx`**).

The plan is intentionally ordered. The package-manager core is delivered first through multiple MVPs, followed by script and executable runners, runtime augmentation, Node and package-manager management, project creation, plugins, distribution, and final certification.

## Product Contract

- `m` is the main JavaScript toolchain and package-manager command.
- `mx` executes local or temporary package binaries.
- New Mew projects use `m.lock`.
- Existing projects preserve their current lockfile format when a certified writer exists.
- `nub.lock` is a first-class compatibility target.
- `m dev`, `m start`, and any other exact package.json script name work as direct shortcuts after built-in commands and aliases are checked.
- The long-term target is complete toolchain capability, implemented with Go-native architecture.
- The package-manager engine, orchestration, storage, compatibility layers, and transform service are implemented in Go. Small embedded JavaScript loader/preload assets remain where Node extension APIs require JavaScript.

## Signature Mew Improvements

1. Recoverable transactional installation.
2. Instant rollback, history, and dependency time travel.
3. Explainable dependency resolution and mutation plans.
4. Semantic lockfile diff, validation, and migration across managers.
5. A smart filesystem planner for hardlinks, reflinks, copies, symlinks, and junctions.
6. Capability-based lifecycle script trust and sandbox policy.
7. Portable verified dependency capsules.
8. Direct script shortcuts with deterministic command precedence.

## How to Use This Plan

1. Read `0001` through `0009` before implementing product code.
2. Track progress in [`CHECKLIST.md`](CHECKLIST.md) (Do now + per-MVP tasks).
3. Implement indexed MVPs in dependency order. Parallel work is allowed only when `Required predecessors` are satisfied and interfaces are frozen.
4. Use the numbered `00xx-*.md` file as the source of truth for scope when driving an agent session.
5. Treat every MVP file as an executable engineering contract.
6. Update `manifest.json`, `INDEX.md`, the feature inventory, tests, and references whenever scope changes.
7. Do not begin a stabilization gate until every predecessor MVP meets its own exit criteria.

## Archive Contents

- [`INDEX.md`](INDEX.md): ordered navigation and phase grouping.
- [`CHECKLIST.md`](CHECKLIST.md): master rollup of MVP status and aggregated tasks.
- `0001`-`0090`: foundation, MVP, stabilization, and cross-cutting plans (enriched contracts).
- [`_ENRICHED_TEMPLATE.md`](_ENRICHED_TEMPLATE.md): required section structure for enriched MVP files.
- [`scripts/enrich-and-generate.ps1`](scripts/enrich-and-generate.ps1): regenerate enrichment blocks, checklist, and `manifest.json`.
- `manifest.json`: machine-readable file inventory with SHA-256 digests.
- `sources/`: concise pinned source notes used to build the plan.

## Regenerate derived artifacts

After editing any `00xx-*.md` scope or an `scripts/enrichment-*.json` catalog entry:

```powershell
.\plans\scripts\enrich-and-generate.ps1
```

Thin wrappers (`generate-checklist.ps1`, `update-manifest.ps1`) call the same entrypoint.

All prose in this archive is English so human contributors and AI coding agents can use the same implementation source of truth.
