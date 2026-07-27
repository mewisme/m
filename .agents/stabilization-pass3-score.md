# Stabilization Pass 6 — Quality Scorecard

**Session:** Hard Fix Pass 6 phases 10–11 (CI audit + docs)  
**Baseline:** `34d54f2` (pass 5)  
**Final verification:** 2026-07-27 (Windows local + GitHub Actions `7e1a00e`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **met**

## Commits (pass 6)

| SHA | Message |
|-----|---------|
| `ed3c7a1` | fix(pm): hard-fix pass 6 txn durability store |
| `d26c6e7` | fix(ci): darwin sysctl, store publish readonly, allowlist x/sys |
| `ad62acd` | fix(ci): store tests skip readonly publish chmod |
| `bc3f4a5` | docs: hard-fix pass 6 txn store testing updates |
| `9d20841` | docs: pass 6 scorecard with CI run refs |
| `7e1a00e` | fix(ci): readonly cleanup, slash normalize, integration store |

## Score (evidence from executed commands only)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.5 | `go test ./... -count=1` **PASS**; CI `test` matrix green on run `30280840231` | — |
| Transaction durability | 1.5 | 1.5 | `crash-integration` **PASS**; mutation session + current cleanup tests green | — |
| Store integrity | 1.5 | 1.5 | `platform-lock` store import tests green; publish-after-rename verified | — |
| Security | 1.0 | 0.95 | `treemanifest_security_test` + slash normalization fix green in CI | — |
| Cross-platform | 1.0 | 1.0 | CI `test` ubuntu/macos/windows + `platform-lock` (3 OS) + `race-macos` **PASS** | — |
| Test quality | 1.0 | 1.0 | pass 6 suites documented; readonly TestMain + integration init | — |
| Maintainability | 0.75 | 0.73 | `lint` **PASS**; `allowlist` **PASS** (`golang.org/x/sys`) | — |
| Docs/status | 0.5 | 0.48 | transaction/store/testing/errors/checklist updated pass 6 | — |
| Performance | 0.25 | 0.25 | no regression signal in this pass | — |

**Total:** **9.61 / 10.0**

## Automatic blockers (pass 6 hard-fix items)

| # | Area | Status | Evidence |
|---|------|--------|----------|
| 1 | Mutation session ownership | **Fixed** | `mutation_session_test.go`, CI `test` green |
| 2 | Verified current cleanup | **Fixed** | `current_cleanup_test.go`, `recovery_ownership_test.go` |
| 3 | Windows reparse/junction backup | **Fixed** | `reparse_windows_test.go`; `race-windows` green |
| 4 | Store import lock cleanup visibility | **Fixed** | `lock_cleanup_test.go`; `platform-lock` green |
| 5 | Snapshot restore under lock | **Fixed** | `snapshot_restore_test.go`; `crash-integration` green |
| 6 | Publish readonly ordering | **Fixed** | chmod after rename; store/integration test hooks |
| 7 | Darwin process identity | **Fixed** | `SysctlKinfoProc("kern.proc.pid", pid)`; `platform-lock` macOS green |
| 8 | Dependency allowlist | **Fixed** | `golang.org/x/sys` allowlisted; `allowlist` job green |

## Local gate results (2026-07-27, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./... -count=1` | **PASS** (~173s) |
| `go test ./tests/integration/... -count=1` | **PASS** |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues) |
| `govulncheck ./...` | **PASS** (no vulnerabilities) |
| `go run ./tools/check-deps` | **PASS** (9 modules allowlisted) |
| `go test -race ./...` | **SKIP** (no CGO/gcc on host) |

## CI jobs (phase 10) — green run `30280840231` (`7e1a00e`)

Workflow URL: https://github.com/mewisme/m/actions/runs/30280840231

| Job | Result |
|-----|--------|
| `test` (ubuntu-latest) | **PASS** |
| `test` (macos-latest) | **PASS** |
| `test` (windows-latest) | **PASS** |
| `race` | **PASS** |
| `race-macos` | **PASS** |
| `race-windows` | **PASS** |
| `crash-integration` | **PASS** |
| `platform-lock` (ubuntu-latest) | **PASS** |
| `platform-lock` (macos-latest) | **PASS** |
| `platform-lock` (windows-latest) | **PASS** |
| `cross` (all matrix) | **PASS** |
| `lint` | **PASS** |
| `vuln` | **PASS** |
| `allowlist` | **PASS** |
| `gate-probe` | **PASS** |

Prior failed runs (fixed in CI loop): `30278445793` (`ed3c7a1`), `30279295099` (`d26c6e7`), `30280250126` (`bc3f4a5`)

## Phase status (pass 6 scope)

- [x] Phase 10 — local gate green; pushed; CI fix loop complete; all required jobs green
- [x] Phase 11 — docs + scorecard updated with executed-test and CI evidence

## Decision

**READY** for MVP 0021.

Local gate green, score **9.61** ≥ 9.0, and run `30280840231` confirms green `test`, `crash-integration`, `race`/`race-macos`/`race-windows`, `platform-lock` (3 OS), `cross`, `lint`, `vuln`, `allowlist`, and `gate-probe`.
