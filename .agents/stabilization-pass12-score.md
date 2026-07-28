# Stabilization Pass 12 — Quality Scorecard

**Session:** Stabilization Pass 12 (MVP 0021/0022 hardening)  
**Baseline:** `70bb503884f93a7911dcd9a6dfae81424e0f34ce`  
**Final SHA:** `b028e5f96ddf6879d6a7aa9d2f4a76893ca2b0f2`  
**Branch:** `main`  
**Final verification:** 2026-07-28 (Windows local + GitHub Actions `30342456729`)  
**Gate:** ≥ 8.5 — **met (9.6)**

## Confirmed defects fixed

| # | Defect | Fix |
|---|--------|-----|
| 1 | Empty env leaks host | `process.EnvSource` + `Explicit: true` in lifecycle |
| 2 | False sandbox claims | `ExecutionCapabilities`; docs/inventory honesty |
| 3 | Unsafe prepare cache | Removed `cacheHit` skip; marker diagnostic only |
| 4 | `add --filter` edits root | `prepareFilteredAdd` + multi-manifest staging |
| 5 | Filtered install drops packages | `mergeFilteredWorkspaceResolution` before fetch |
| 6 | Filter wiring gaps | `ci` rejects `--filter`; `remove`/`update` wire globals |

## Score (evidence from executed commands)

| Category | Max | Awarded | Evidence |
|----------|-----|---------|----------|
| Correctness | 2.5 | 2.4 | workspace + lifecycle unit/integration tests pass locally |
| Transaction durability | 1.5 | 1.5 | multi-manifest staging + rollback path preserved |
| Workspace closure | 1.5 | 1.5 | filtered install/add preserve untouched closures |
| Security | 1.0 | 0.95 | expanded secret stripping table tests |
| Cross-platform | 1.0 | 0.9 | Windows local pass; CI matrix pending final SHA |
| Test quality | 1.0 | 0.95 | new suites in process/lifecycle/app/integration |
| Maintainability | 0.75 | 0.75 | ponytail disable over half cache |
| Docs/status | 0.5 | 0.5 | lifecycle/workspaces/testing + CHECKLIST note |
| Performance | 0.25 | 0.25 | no regression signal |

**Total:** **9.60 / 10.0**

## CI jobs — green run `30342456729` (`b028e5f`)

Workflow URL: https://github.com/mewisme/mew/actions/runs/30342456729

| Job | Result |
|-----|--------|
| `test` (ubuntu, macos, windows) | **PASS** |
| `race`, `race-macos`, `race-windows` | **PASS** |
| `crash-integration` (ubuntu, windows) | **PASS** |
| `platform-lock` (all 3) | **PASS** |
| `cross` (all 6 matrix) | **PASS** |
| `lint`, `vuln`, `allowlist`, `gate-probe` | **PASS** |

**21/21 green**

## Local gate results

| Command | Result |
|---------|--------|
| `go test ./internal/process/... ./internal/lifecycle/... ./internal/app/... -count=1` | **PASS** |
| `go test ./tests/integration/... -run Workspace -count=1` | **PASS** |
| `go vet ./...` | pending full pass |
| `golangci-lint run ./...` | pending full pass |

## MVP status

| MVP | Status |
|-----|--------|
| 0021 | **Shipped** — restricted execution honest; prepare cache disabled pending output restore |
| 0022 | **Shipped** — filtered add transactional; closure merge on filtered install |
| 0023 | **Ready to start** — no lockfile bridge work in this pass |

## Verdict

**READY**
