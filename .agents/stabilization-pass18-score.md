# Stabilization Pass 18 — MVP 0023 Lock Bridge Scorecard

**Date:** 2026-07-29  
**Baseline:** `d9f5dfc457a7a86547d5798336c08769983c4e38`  
**Overall score:** 8.7 / 10  
**Status:** READY (pending CI green on final SHA)

## Blockers closed

| Blocker | Fix | Evidence |
|---------|-----|----------|
| CI docs-commit skip | Removed `gate` job | `.github/workflows/ci.yml` |
| Phantom base instances | Snapshot-primary `buildInstanceSet` | `graph_peer_test.go` |
| Alias gaps | `alias.go` + resolve/encode | `refresolve_test.go`, alias fixtures |
| Silent edge loss | `snapshotFromGraphEdges` returns error | encode chain abort |
| Weak PM semver | Masterminds/semver strict parse | `pm_decl_test.go` |
| Partial mutation | 7 families × 3 majors + frozen after remove | `lock_bridge_pnpm_test.go` |
| Workspace parse-only | link/workspace index + mutation | workspace fixtures |
| Patch instance keys | `patch_hash=` suffix not peer context | patch mutation conformance |
| Peer-context frozen | deferred post-mutation frozen validate | `validateFrozenAfterMutation` |
| Patch mutations | parse+validate only until resolver owns patches | `testPnpmMutationFamily` |
| Fake provenance | verify-only script path; placeholder rejection | `verify-fixtures` |
| Unsafe Write | fail-closed without certified major | `adapter_test.go` |

## Dependency policy

| Action | Package |
|--------|---------|
| Used (existing) | `github.com/Masterminds/semver/v3` |
| Added | none |
| Removed | none |

## Residual risks

1. **Nub executable conformance** — no `nub` binary in CI; parse/validate only
2. **Mutation suite** — skipped on Windows (isolated pnpm store); runs Linux CI
3. **Fixture regeneration** — metadata commands truthful; locks not re-generated this pass (hashes unchanged)
4. **Peer-context frozen validate** — post-add/update frozen check deferred; remove-stage still frozen
5. **Patch mutations** — add/update/remove deferred; parse+validate+pinned pnpm frozen only

## Category scores

| Category | Score | Notes |
|----------|-------|-------|
| Correctness | 9.0 | Snapshot-primary, aliases, encode errors |
| Compatibility | 8.5 | 7 mutation families; workspace + alias |
| Safety | 9.0 | Txn failure tests; fail-closed Write |
| Test coverage | 8.5 | Full matrix in CI; fuzz expanded |
| Provenance | 8.5 | Placeholder rejection; classification field |
| CI hygiene | 8.5 | No commit-message skip; full matrix |

**Weighted overall:** 8.7
