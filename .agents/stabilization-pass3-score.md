# Stabilization Pass 8 — Quality Scorecard

**Session:** Stabilization Pass 8 (config ordering + cleanup error chain)  
**Baseline:** `867a8ba79762f73a4043c01460704ddb6391dae2` (merged Pass 7 on `main`)  
**Branch:** `stabilization-pass-8`  
**Final verification:** 2026-07-28 (Windows local + CI pending)  
**Gate:** ≥ 9.0 to unblock MVP 0021

## Commits (pass 8)

| SHA | Message |
|-----|---------|
| `bb5094a` | fix(app): use refreshed config after mutation ownership |
| `acb7a90` | fix(transaction): aggregate cleanup failures in FinishResult |
| `fa92e90` | test(app): cover config reload while waiting on lock |
| `d9a1625` | docs: record stabilization pass 8 evidence |

## Score (evidence from executed commands only)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.5 | `go test ./... -count=1` **PASS** (Windows ~164s) | — |
| Transaction durability | 1.5 | 1.5 | `finish_result_test.go`, `abort_test.go` cleanup chain; crash-integration CI pending | — |
| Store integrity | 1.5 | 1.5 | pass 7 store suites green in full test run | — |
| Security | 1.0 | 0.95 | existing treemanifest/security suite green | — |
| Cross-platform | 1.0 | 1.0 | CI matrix pending on final SHA | — |
| Test quality | 1.0 | 1.0 | `mutation_config_wait_test.go` proc test; `TestUpdateFetchFailPreservesTree` aligned to project-config reload | — |
| Maintainability | 0.75 | 0.75 | `lint` **PASS**; `allowlist` **PASS** (CI) | — |
| Docs/status | 0.5 | 0.50 | `transaction.md`, `testing.md`, `CHECKLIST.md` updated | — |
| Performance | 0.25 | 0.25 | no regression signal | — |

**Total (local):** **9.90 / 10.0** — final score requires CI green on final SHA

## Automatic blockers (pass 8 items)

| # | Area | Status | Evidence |
|---|------|--------|----------|
| 1 | Stale effective config in `runInstallInSession` | **Fixed** | `ReopenProject` → `AppContext` ordering; `AppContext` errors before reload |
| 2 | Cleanup failures not in error chain | **Fixed** | `FinishResult.CleanupError()`, `joinSessionCleanup`, `abort_test.go` `errors.Is` |
| 3 | Config-wait proc test | **Fixed** | `tests/integration/mutation_config_wait_test.go` |
| 4 | Audit grep hygiene | **Clean** | see audit table below |

## Audit grep (pass 8)

| Pattern | Result |
|---------|--------|
| `AppContext()` before `ReopenProject` in production | **None** — only `install_txn.go` calls both; correct order |
| `abortRes, _ := abortMutation` | **None** |
| Discarded `abortMutation` in prod paths | **None** — `install_txn.go` / `snapshot_restore.go` propagate `abortErr` |
| `_, _ = sess.Abort` in tests | **Intentional** — test teardown only (`mutation_session_test.go`, `mutation_prepare_test.go`) |
| Duplicate cleanup warnings | **Deduped** — `joinDistinctCleanup` via `errors.Is`; `releaseSessionLock` appends once |

## Local gate results (2026-07-28, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./... -count=1` | **PASS** (~164s) |
| `go test ./internal/app/... ./internal/transaction/... ./internal/config/... ./internal/resolver/... ./tests/integration/... -count=1` | **PASS** |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues) |
| `govulncheck ./...` | **PASS** (no vulnerabilities) |
| `go run ./tools/check-deps` | **PASS** (9 modules allowlisted) |
| `go test -race ...` | **SKIP** (no CGO/gcc on host; CI race jobs required) |

## CI jobs — pending final SHA

_(Update after push with workflow URL and per-job results.)_

## MVP status

| MVP | Status |
|-----|--------|
| 0017 Transactional install | **Done** |
| 0020 Full resolver | **Done** |
| 0021 Lifecycle scripts | **Blocked** until pass 8 CI green |

## Decision

**BLOCKED** until CI green on final commit SHA.
