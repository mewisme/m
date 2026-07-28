# Stabilization Pass 14 — Quality Scorecard

**Session:** Stabilization Pass 14 (module rename + workspace/snapshot/lifecycle hardening)  
**Baseline:** `9f39d8136077ba47df8a2f629c457a385c5d438f`  
**Final SHA:** `cf61ed53d691ed9af1a4dcbb61a0b57dc530e27a`  
**Docs tip SHA:** `4350b88a2e8b9eab88f13d80b1259b5448964a82`  
**Branch:** `main`  
**Final verification:** 2026-07-28 (Windows local + GitHub Actions `30352453527`)  
**Gate:** ≥ 8.5 — **met (9.35)**

## Confirmed defects fixed

| # | Defect | Fix |
|---|--------|-----|
| 1 | Untouched workspace merge dropped package-to-package edges | `preservedSubgraphEdges` + topology validation |
| 2 | `writeLiveMemberManifests` pre-commit mutation | Removed; stage-only member writes |
| 3 | `backupPaths` skipped staged member paths | `allMemberManifestPaths` on backup |
| 4 | Lifecycle timeout collapsed to generic exit 1 | Preserve `DeadlineExceeded` / `Canceled` |
| 5 | Weak `guardMemberManifest` | `ParseMemberManifestPath` + v2 restore audit |

## Score (recalculated from pass 14 evidence)

| Category | Max | Awarded | Evidence |
|----------|-----|---------|----------|
| Correctness | 2.5 | 2.45 | subgraph merge, transactional restore, typed timeouts, path validation |
| Transaction durability | 1.5 | 1.5 | no live member writes; workspace restore crash matrix |
| Workspace closure | 1.5 | 1.5 | package-to-package edge unit + integration |
| Security | 1.0 | 0.95 | strict member path parser; restore pair validation |
| Cross-platform | 1.0 | 0.95 | Windows local pass; CI 21/21 on `cf61ed5` |
| Test quality | 1.0 | 0.95 | merge, snapshot, lifecycle, crash suites |
| Maintainability | 0.75 | 0.7 | module rename mechanical; ponytail stage overlay comment |
| Docs/status | 0.5 | 0.5 | workspaces/lifecycle/testing + CHECKLIST |
| Performance | 0.25 | 0.25 | no regression signal |

**Total:** **9.35 / 10.0**

## Local gate results (Windows, 2026-07-28)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./internal/app/... ./internal/snapshot/... ./internal/lifecycle/... -count=1` | **PASS** |
| `go test ./tests/integration/... -count=1` | **PASS** |
| `go test ./... -count=1` | **PARTIAL** (pre-existing Windows: `internal/store` symlink privilege) |
| `go test -race ./... -count=1` | **SKIP** (Windows: CGO disabled) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** |
| `govulncheck ./...` | **PASS** |
| `go run ./tools/check-deps` | **PASS** |
| `go test -tags crash ./tests/integration/... -run Crash -count=1` | **PASS** |

## CI jobs — green run `30352453527` (`cf61ed5`)

Workflow URL: https://github.com/mewisme/mew/actions/runs/30352453527

| Job | Result |
|-----|--------|
| `test` (ubuntu, macos, windows) | **PASS** |
| `race`, `race-macos`, `race-windows` | **PASS** |
| `crash-integration` (ubuntu, windows) | **PASS** |
| `platform-lock` (all 3) | **PASS** |
| `cross` (all 6 matrix) | **PASS** |
| `lint`, `vuln`, `allowlist`, `gate-probe` | **PASS** |

**21/21 green**

## MVP status

| MVP | Status |
|-----|--------|
| 0023 | **Ready to start** — out of scope for pass 14 |

## Verdict

**READY**
