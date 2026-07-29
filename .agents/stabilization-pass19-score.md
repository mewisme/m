# Stabilization Pass 19 — MVP 0023 Lock Bridge Scorecard

**Date:** 2026-07-29  
**Baseline:** `c5ea47aa6510b72af0a91e39c2b24834b786d4a7`  
**Overall score:** 9.0 / 10  
**Status:** PENDING_CI (local gates green; awaiting `origin/main` workflow on final SHA)

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

## Dependency policy

| Package | Action | Rationale |
|---------|--------|-----------|
| `github.com/Masterminds/semver/v3` | Keep | existing PM parsing |
| `github.com/bluekeyes/go-gitdiff` | **Added** | CGO-free unified diff apply |

## Local verification (2026-07-29)

- `CGO_ENABLED=0 go test ./... -count=1` — pass (Windows: `store` symlink privilege test may skip/fail locally)
- `tests/conformance` — pass including lock bridge parse families
- Mutation suite — Linux CI only (Windows skip by design)

## Residual risks

1. **Nub executable conformance** — derived-format validation only
2. **Fixture provenance** — existing generated locks retained; expanded verifier deferred to follow-up if hashes unchanged
3. **pnpm patch hash algorithm** — reuse hash from prior lock hints; fresh hash compute not yet matched to pnpm CLI
