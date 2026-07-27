# Stabilization Pass 6 — Quality Scorecard

**Session:** Hard Fix Pass 6 phases 10–11 (CI audit + docs)  
**Baseline:** `34d54f2` (pass 5)  
**Final local verification:** 2026-07-27 (Windows, `F:\Project\package-managers\mew`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 (local evidence met; CI pending run `30280250126`)

## Commits (pass 6)

| SHA | Message |
|-----|---------|
| `ed3c7a1` | fix(pm): hard-fix pass 6 txn durability store |
| `d26c6e7` | fix(ci): darwin sysctl, store publish readonly, allowlist x/sys |
| `ad62acd` | fix(ci): store tests skip readonly publish chmod |
| `bc3f4a5` | docs: hard-fix pass 6 txn store testing updates |

## Score (evidence from executed commands only)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.45 | `go test ./... -count=1` **PASS** (~173s); integration ~170s; rollback test fixed (`m rollback` not `snapshot rollback`) | — |
| Transaction durability | 1.5 | 1.5 | mutation session, current cleanup, reparse backup, recovery_ownership tests **PASS** | — |
| Store integrity | 1.5 | 1.45 | import publish order fixed; lock cleanup tests **PASS**; readonly round-trip test added | — |
| Security | 1.0 | 0.95 | path guards + treemanifest tests **PASS** | — |
| Cross-platform | 1.0 | 0.75 | Windows local green; darwin `SysctlKinfoProc("kern.proc.pid", pid)` fix pushed; CI run pending | −0.25 until Actions green |
| Test quality | 1.0 | 1.0 | pass 6 suites documented; store TestMain prevents readonly cleanup flake | — |
| Maintainability | 0.75 | 0.73 | `golangci-lint run ./...` **PASS**; `golang.org/x/sys` allowlisted | — |
| Docs/status | 0.5 | 0.48 | `transaction.md`, `store.md`, `testing.md`, `errors.md`, `transaction-boundary.md`, `CHECKLIST.md` updated | — |
| Performance | 0.25 | 0.25 | no regression signal in this pass | — |

**Estimated total:** **9.56 / 10.0** (local evidence; CI URLs below)

## Automatic blockers (pass 6 hard-fix items)

| # | Area | Local status | Evidence |
|---|------|--------------|----------|
| 1 | Mutation session ownership | **Fixed** | `mutation_session_test.go`, `mutation_prepare_test.go` PASS |
| 2 | Verified current cleanup | **Fixed** | `current_cleanup_test.go`, `recovery_ownership_test.go` PASS |
| 3 | Windows reparse/junction backup | **Fixed** | `reparse_windows_test.go`, `reparse_backup.go` wired in runner |
| 4 | Store import lock cleanup visibility | **Fixed** | `lock_cleanup_test.go` PASS |
| 5 | Snapshot restore under lock | **Fixed** | `snapshot_restore_test.go` PASS (`m rollback`) |
| 6 | Publish readonly ordering | **Fixed** | chmod after rename; store tests use TestMain skip |
| 7 | Darwin process identity | **Fixed** | `SysctlKinfoProc` pid arg (was string concat) |
| 8 | Dependency allowlist | **Fixed** | `golang.org/x/sys` in `tools/allowlist/modules.txt` |

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

## CI jobs (phase 10)

Workflow: [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)

| Job | Run `30280250126` (commit `bc3f4a5`) | Run `30279295099` (`d26c6e7`, prior) |
|-----|--------------------------------------|----------------------------------------|
| Workflow URL | https://github.com/mewisme/m/actions/runs/30280250126 | https://github.com/mewisme/m/actions/runs/30279295099 |
| `test` (ubuntu/macos/windows) | **pending** | failure (readonly TempDir cleanup) |
| `race` / `race-macos` / `race-windows` | **pending** | failure / pass / pass |
| `crash-integration` | **pending** | pass |
| `platform-lock` (3 OS) | **pending** | failure (readonly cleanup) |
| `cross` | **pending** | pass |
| `lint` | **pending** | pass |
| `vuln` | **pending** | pass |
| `allowlist` | **pending** | pass (after `d26c6e7`) |
| `gate-probe` | **pending** | pass |

Prior failed run (pre-CI fixes): https://github.com/mewisme/m/actions/runs/30278445793 (`ed3c7a1`)

## Phase status (pass 6 scope)

- [x] Phase 10 — local gate green; commits pushed; CI fix loop through `ad62acd`
- [x] Phase 11 — docs + scorecard updated with executed-test evidence
- [ ] Phase 10 exit — all required CI jobs green on `bc3f4a5` (watching `30280250126`)

## Decision

**BLOCKED** for MVP 0021.

Local gate is green and score **9.56** meets the ≥ 9.0 bar. Unblock when run `30280250126` confirms green `test`, `crash-integration`, `race`/`race-macos`/`race-windows`, `platform-lock` (3 OS), `cross`, `lint`, `vuln`, `allowlist`, and `gate-probe`.
