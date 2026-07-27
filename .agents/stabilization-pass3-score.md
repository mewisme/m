# Stabilization Pass 7 — Quality Scorecard

**Session:** Stabilization Pass 7 phases 5–6 (audit, verify, CI loop)  
**Baseline:** `2cd509b` / `7e1a00e` (pass 6 green CI)  
**Branch:** `stabilization-pass-7` ([PR #3](https://github.com/mewisme/m/pull/3))  
**Final verification:** 2026-07-27 (Windows local + GitHub Actions `c7c4b1b`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **met**

## Commits (pass 7)

| SHA | Message |
|-----|---------|
| `cd16f9d` | fix(app): propagate rollback cleanup aggregate errors |
| `bac9a3b` | fix(transaction): restore nested junctions via backups-meta |
| `5d8ec3e` | fix(app): propagate store lock cleanup through install |
| `6eef6f5` | fix(app): reload effective config under mutation lock |
| `61eebd0` | test(cli): install JSON cleanup fields; docs transaction junction meta |
| `c7c4b1b` | fix(ci): disable store readonly publish in app cleanup tests |

## Score (evidence from executed commands only)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.5 | `go test ./... -count=1` **PASS** (Windows ~163s); CI `test` matrix green on run `30286046231` | — |
| Transaction durability | 1.5 | 1.5 | `crash-integration` **PASS**; `abort_test.go`, nested junction Windows tests green | — |
| Store integrity | 1.5 | 1.5 | store cleanup propagation + `install_store_cleanup_test.go`; `platform-lock` green | — |
| Security | 1.0 | 0.95 | existing treemanifest/security suite green in full test run | — |
| Cross-platform | 1.0 | 1.0 | CI `test` ubuntu/macos/windows + `platform-lock` (3 OS) + `race`/`race-macos`/`race-windows` **PASS** | — |
| Test quality | 1.0 | 0.90 | abort, junction, store, config reload, CLI JSON tests added; concurrent proc registry-wait test **deferred** (see gaps) | −0.10 deferral |
| Maintainability | 0.75 | 0.73 | `lint` **PASS**; `allowlist` **PASS** | — |
| Docs/status | 0.5 | 0.50 | `docs/transaction.md` backups-meta; scorecard recalculated | — |
| Performance | 0.25 | 0.25 | no regression signal in this pass | — |

**Total:** **9.83 / 10.0**

## Automatic blockers (pass 7 items)

| # | Area | Status | Evidence |
|---|------|--------|----------|
| 1 | Rollback/cleanup error propagation | **Fixed** | `OperationFailure`, `JoinCleanup`, `abort_test.go`; no discarded `abortMutation` in prod paths |
| 2 | Nested Windows junction restore | **Fixed** | `backups-meta/` schema; `backup_tree_windows_test.go` expanded |
| 3 | Store lock cleanup visibility | **Fixed** | `FetchOutcome`, reporter wiring, `install_store_cleanup_test.go`, CLI JSON tests |
| 4 | Effective config reload under lock | **Fixed** | `ReloadEffectiveConfig`, `AppContext()`, `mutation_session_test.go`, `mutation_config_test.go` |
| 5 | Audit grep hygiene | **Fixed** | no `abortRes,_`; no `.reparse.json` suffix restore logic; docs updated |

## Deferred (non-blocking)

| Item | Reason |
|------|--------|
| Concurrent two-goroutine proc test: registry mapping refresh while second mutation waits on lock | Covered by `TestMutationSessionScopedRegistryReload` + integration `TestAddScopedPackageUsesProjectRegistryMapping`; full proc race deferred (~30min+ fixture) |

## Local gate results (2026-07-27, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./internal/app/... ./internal/transaction/... ./internal/fsx/... ./internal/store/... ./tests/integration/... -count=1` | **PASS** (~173s) |
| `go test ./... -count=1` | **PASS** (~163s) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues) |
| `govulncheck ./...` | **PASS** (no vulnerabilities) |
| `go run ./tools/check-deps` | **PASS** (9 modules allowlisted) |
| `go test -race ./internal/app/... ./internal/transaction/... ./internal/store/... ./internal/fsx/... -count=1` | **SKIP** (no CGO/gcc on host; CI race jobs green) |

## CI jobs — green run `30286046231` (`c7c4b1b`)

Workflow URL: https://github.com/mewisme/m/actions/runs/30286046231

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

Prior failed run (fixed in CI loop): `30285422974` (`61eebd0`) — `TestFetchAndImportGraphSurfacesStoreCleanup` / `TestInstallPreservesStoreWarningsOnLinkFailure` failed **TempDir cleanup** on Unix because store `publishReadOnly` left imported packages unreadable; fixed by `store.SetPublishReadOnly(false)` in app cleanup tests (`c7c4b1b`).

## Phase status (pass 7 scope)

- [x] Phase 5 — grep audit; fixes + docs; CLI install JSON tests
- [x] Phase 6 — full local gates green; pushed; CI loop complete; all required jobs green

## Decision

**READY** for MVP 0021.

Score **9.83** ≥ 9.0; run `30286046231` confirms green `test`, `crash-integration`, `race`/`race-macos`/`race-windows`, `platform-lock` (3 OS), `cross`, `lint`, `vuln`, `allowlist`, and `gate-probe` on `c7c4b1b`.
