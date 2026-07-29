# Stabilization Pass 19 — MVP 0023 Lock Bridge Scorecard

**Date:** 2026-07-29  
**Baseline:** `c5ea47aa6510b72af0a91e39c2b24834b786d4a7`  
**Final SHA:** `c0536bf` (`fix(fsx): distinct stub reparse tags for non-Windows builds`)  
**Overall score:** 9.5 / 10  
**Status:** READY

## CI evidence (run `30453214621` on `c0536bf`)

| Job | Result |
|-----|--------|
| conformance-pnpm-9 | **pass** (includes workspace mutation suite) |
| conformance-pnpm-10 | **pass** |
| conformance-pnpm-11 | **pass** |
| conformance-nub-fixtures | pass |
| lint | pass |
| fixture-verify | pass |
| test (ubuntu-latest) | pass |
| test (macos-latest) | pass |
| test (windows-latest) | pass |
| race / race-macos | pass |
| no-cgo-gate | pass |
| crash-integration (all shards) | pass |

## CI follow-up fixes (`91c8b9e` → `c0536bf`)

| Commit | Root cause | Jobs restored |
|--------|------------|---------------|
| `c1445aa` | `manifestDriftsFromLock` errored on missing `m.lock` (greenfield); `buildUpdateClosure` pulled non-target deps into incremental updates | test, race, no-cgo-gate, crash-integration |
| `445791b` | Absolute symlink/junction targets broke after `stage/node_modules` publish rename | isolated integration tests |
| `d9969c1` | Windows directory-symlink reparse tag `0xA000000C` unsupported in `node_modules` backup | windows workspace mutation + snapshot crash matrix |
| `c0536bf` | Stub reparse tag constants collided on `!windows` builds | macOS compile / vet gates |

## Root cause fixed (workspace)

1. **Linker mode:** pnpm workspaces use `pnpm-workspace.yaml`, not `package.json#workspaces`. `resolveLinkerMode` only checked the latter, so mutation installs used **hoisted** layout and kept `node_modules/ms` at the root after removing the root dependency.
2. **Manifest drift:** External `package.json` edits (conformance `mutateRemoveDependency`) did not set `manifestChanged`, so install reused stale lock graph hints without full reconcile.
3. **Hoisted mkdir:** `OpMkdir` on full nested dest failed when incumbent pollution left a leaf (`pnpm frozen` between mutations) — fixed via parent-dir mkdir (prior commits).
4. **Workspace remove verify:** `require('ms')` can still succeed when `pkg-a` declares `ms`; conformance now checks root `node_modules/ms` removal only.

## Pass 18 deferrals closed

| Deferral | Pass 19 fix | Evidence |
|----------|-------------|----------|
| `runPnpmFrozenAfterMutation` skips | Removed; always `runPnpmFrozen` | `lock_bridge_pnpm_test.go` |
| `validateFrozenAfterMutation` skips | Always `--frozen` validate | `lock_bridge_pnpm_test.go` |
| Patch parse-only mutation | Full add/update/remove matrix | `lock_bridge_pnpm_test.go` |
| Workspace graph verify skip | `verifyStage` + link check | `lock_bridge_pnpm_test.go` |
| Missing update graph verify | `verifyStage` after update | `lock_bridge_pnpm_test.go` |
| Non-frozen repeat install | Always `--frozen-lockfile` repeat | `lock_bridge_pnpm_test.go` |
| Alias+peer encode bug | `refencode.go` actualName path | `refencode_test.go` |
| Workspace `~` unsupported | manifest + resolver | `workspace_test.go`, `specifier_test.go` |
| Workspace link encode | `preservePriorWorkspaceRef` | `graph.go` |
| Patch resolver-owned | `resolver/patch.go` + apply | `patch_encode_test.go`, `patchapply.go` |
| Manifest mutation loses hints | `ropts.Hints = prior` | `install_helpers.go` |

## Family capability matrix

| Family | Parse | Graph | Mew frozen | pnpm frozen | Installed graph | Runtime |
|--------|-------|-------|------------|-------------|-----------------|---------|
| basic | yes | yes | yes | yes | yes | yes |
| transitive | yes | yes | yes | yes | yes | yes |
| optional | yes | yes | yes | yes | yes | yes |
| peer-context | yes | yes | yes | yes | yes | yes |
| workspace | yes | yes | yes | yes | yes | yes |
| alias | yes | yes | yes | yes | yes | yes |
| patch | yes | yes | yes | yes | yes | yes (`ms-patched`) |

pnpm majors: 9.15.9, 10.34.5, 11.17.0 (pinned in `tools/conformance/pnpm-versions.env`).

## Residual product risks

1. **Nub executable conformance** — derived-format validation only
2. **Fixture provenance** — phase 7 deferred
3. **pnpm patch hash algorithm** — reuse from prior lock hints
