# Stabilization Pass 8 — Quality Scorecard

**Session:** Stabilization Pass 8 (config ordering + cleanup error chain)  
**Baseline:** `867a8ba79762f73a4043c01460704ddb6391dae2` (merged Pass 7 on `main`)  
**Branch:** `stabilization-pass-8`  
**Final verification:** 2026-07-28 (Windows local + GitHub Actions `4be6354`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **met**

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

**Total:** **9.90 / 10.0**

## CI jobs — green run `30291154930` (`4be6354`)

Workflow URL: https://github.com/mewisme/m/actions/runs/30291154930

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

## Automatic blockers (pass 8 items)

| MVP | Status |
|-----|--------|
| 0017 Transactional install | **Done** |
| 0020 Full resolver | **Done** |
| 0021 Lifecycle scripts | **Unblocked** — pass 8 CI green on `4be6354` |

## Decision

**READY** for MVP 0021.

Score **9.90** ≥ 9.0; run `30291154930` confirms green `test`, `crash-integration`, `race`/`race-macos`/`race-windows`, `platform-lock` (3 OS), `cross`, `lint`, `vuln`, `allowlist`, and `gate-probe` on `4be6354`.

---

# Stabilization Pass 9 — Quality Scorecard

**Session:** Stabilization Pass 9 (config load spec + cleanup severity)  
**Baseline:** `fae9b4855b825474af27b1f907963ceca3c55e56`  
**Branch:** `stabilization-pass-9`  
**Final verification:** 2026-07-28 (Windows local + GitHub Actions `30297084653` on `d148750`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **met**

## Gaps fixed

| # | Gap | Fix |
|---|-----|-----|
| 1 | Mutation reload dropped `--config` path, env snapshot, CLI overlays | `config.LoadSpec` captured at `app.New`; `ReloadEffectiveConfig` clones spec |
| 2 | Non-critical cleanup warnings became command errors | `CriticalCleanupError()` / `WarningErrors()`; `CleanupCodeSeverity` registry |
| 3 | Config-wait proc test did not prove reload-after-lock-wait | Rewritten sync: `app.New` → lock wait → config rewrite → `ReopenProject` reload |

## Audit grep (pass 9)

| Pattern | Result |
|---------|--------|
| `os.Environ()` in mutation reload | **Fixed** — only in `app.New` env snapshot |
| `cliOverlayFromEffective` | **Removed** |
| `config.Load` hand-built in mutation path | **Fixed** — uses `ConfigLoadSpec` |
| `CleanupError()` in app layer | **Fixed** — `CriticalCleanupError()` |
| `RecoveryRequired` on warning-only finish | **Fixed** — `populateWarningCleanup` |
| `internal/cli/config_cmd.go` `loadEffective` | **Safe** — `m config` only; not mutation path |

## MVP status (pass 9)

| MVP | Status |
|-----|--------|
| 0017 Transactional install | **Done** |
| 0020 Full resolver | **Done** |
| 0021 Lifecycle scripts | **Unblocked** — pass 9 CI green on `d148750` |

## CI jobs — green run `30297084653` (`d148750`)

Workflow URL: https://github.com/mewisme/m/actions/runs/30297084653

| Job | Result |
|-----|--------|
| `test` (ubuntu-latest) | **PASS** |
| `test` (macos-latest) | **PASS** |
| `test` (windows-latest) | **PASS** (rerun after 10m timeout flake) |
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

## Local gate results (2026-07-28, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./internal/app/... ./internal/config/... ./internal/transaction/... ./internal/cli/... ./tests/integration/... -count=1` | **PASS** |
| `go test ./... -count=1` | **PASS** (~164s) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues) |
| `govulncheck ./...` | **PASS** |
| `go run ./tools/check-deps` | **PASS** |
| `go test -race ...` | **SKIP** (CGO_ENABLED=0; CI race jobs green) |

## Score

**9.85 / 10.0** (deduct 0.05 for Windows integration timeout flake on first CI attempt)

## Decision

**READY** for MVP 0021.

---

# Stabilization Pass 10 — Quality Scorecard

**Session:** Stabilization Pass 10 (config path resolution + CLI output + abort severity)  
**Baseline:** `ec2f4110031d5b12577ebf010156b2956946c735`  
**Branch:** `stabilization-pass-10`  
**PR:** https://github.com/mewisme/m/pull/7  
**Final SHA:** `3a164d8680ce8a3b5c2603b1577dee03657aba71`  
**Final verification:** 2026-07-28 (Windows local + GitHub Actions `30308833823`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **met (local + CI)**

## Commits (pass 10)

| SHA | Message |
|-----|---------|
| `599cb1b` | fix(config): resolve config sources from invocation context (+ PR #6 CI split) |
| `ff93fef` | test(config): cover cwd project-root and env-snapshot paths |
| `1706fb4` | fix(cli): separate warning output from recovery guidance |
| `347d5ff` | fix(app): apply cleanup severity to abort results |
| `e7c3b24` | test(cli): require clean JSON and correct recovery hints |
| `479864f` | docs: record stabilization pass 10 evidence |
| `5a7a789` | fix(config): snapshot env for cache/store/linker paths |
| `2cac44c` | fix(app): clone session config load specification |
| `940eab0` | fix(app): deduplicate cleanup warning output |
| `31e90cb` | test(config): cover snapshot paths and Windows casing |
| `00c6f01` | test(app): cover combined cleanup warning output |
| `8f86ba6` | fix(test): use temp dirs for StoreRoot snapshot test |
| `3a164d8` | fix(test): stabilize import lock cancel proc test |

## Gaps fixed

| # | Gap | Fix |
|---|-----|-----|
| 1 | Relative `--config` used `filepath.Abs` (process CWD) | `config.ResolveConfigPath(invocationCWD, path)` in `app.New` |
| 2 | Project/global classification used unsafe `HasPrefix(rel, "..")` on invocation CWD | `config.IsPathWithin(projectRoot, path)` after `project.FindRoot` |
| 3 | Empty `GlobalPath` → `os.Getenv` on every reload | `GlobalConfigPathFromEnv` frozen at `app.New` |
| 4 | Warning-only cleanup triggered `m recover` in human + JSON | `FormatInstallSummary` + `writeInstallResult` use critical flags only |
| 5 | `populateAbortCleanup` treated any warning as critical | `CleanupCodeSeverity` per code in abort path |
| 6 | Cache/store paths read ambient `os.Getenv` on reload | `EnvSnapshot` on `LoadSpec` / `Effective`; `CacheRoot` / `StoreRoot` use snapshot |
| 7 | Windows env lookup case-sensitive in snapshot | `EnvSnapshot.Lookup` normalizes keys on Windows |
| 8 | Session `ConfigLoadSpec` shallow-copied | `MutationSession` uses `ConfigLoadSpec.Clone()` |
| 9 | Duplicate store warnings in `FormatInstallSummary` | Categorized txn vs store blocks with dedupe |

## Audit grep (pass 10)

| Pattern | Result |
|---------|--------|
| `filepath.Abs(opts.ConfigPath)` in app | **Fixed** — uses `ResolveConfigPath` |
| `strings.HasPrefix(rel, "..")` in context | **Removed** |
| `GlobalConfigPath()` in mutation reload | **Fixed** — `GlobalPath` always set from env snapshot |
| Post-JSON prose in `writeInstallResult` | **Removed** |
| `config_cmd` `GlobalConfigPath()` for global writes | **Safe** — ambient intentional for `m config set --global` |
| `fsx/guard.go` `HasPrefix` | **Safe** — ancestor guard, different contract |

## MVP status (pass 10)

| MVP | Status |
|-----|--------|
| 0017 Transactional install | **Done** |
| 0020 Full resolver | **Done** |
| 0021 Lifecycle scripts | **Unblocked** — pass 10 CI green on `3a164d8` |

## CI jobs — run `30308833823` (`3a164d8`)

Workflow URL: https://github.com/mewisme/m/actions/runs/30308833823

**All 21 jobs passed.**

| Job | Result |
|-----|--------|
| `test` (ubuntu/macos/windows) | **PASS** |
| `race` / `race-macos` / `race-windows` | **PASS** |
| `crash-integration` (ubuntu/windows) | **PASS** (windows ~4m22s) |
| `platform-lock` (3 OS) | **PASS** |
| `cross` (all matrix) | **PASS** |
| `lint` / `vuln` / `allowlist` / `gate-probe` | **PASS** |

## CI jobs — run `30304689171` (`479864f`) — superseded

Workflow URL: https://github.com/mewisme/m/actions/runs/30304689171

**All jobs failed immediately with `runner_id: 0` (GitHub Actions billing/spending limit).** No test execution occurred.

| Job | Result |
|-----|--------|
| `test` (ubuntu/macos/windows) | **FAIL** (no runner) |
| `race` / `race-macos` / `race-windows` | **FAIL** (no runner) |
| `crash-integration` | **FAIL** (no runner) |
| `platform-lock` (3 OS) | **FAIL** (no runner) |
| `cross` (all matrix) | **FAIL** (no runner) |
| `lint` / `vuln` / `allowlist` / `gate-probe` | **FAIL** (no runner) |

## Local gate results (2026-07-28, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./internal/config/... ./internal/app/... ./internal/cli/... ./internal/transaction/... ./tests/integration/... -count=1` | **PASS** |
| `go test ./... -count=1` | **PASS** (~29s) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues) |
| `govulncheck ./...` | **PASS** |
| `go run ./tools/check-deps` | **PASS** |

## Score

**10.0 / 10.0**

## Decision

**PASS** — local gates and full CI matrix green on `3a164d8` (run `30308833823`). MVP 0021 may proceed after PR #7 merges to `main`.

---

# Stabilization Pass 11 — Quality Scorecard

**Session:** Stabilization Pass 11 (invocation snapshot completeness)  
**Baseline:** `d980e127d4d6a3b44bca952c25c4b2d2c12a008f`  
**Branch:** `main` (direct commits, no PR)  
**Final verification:** 2026-07-28 (Windows local focused gates; full CI pending verify-ci agent)  
**Gate:** ≥ 9.0 to unblock MVP 0021 — **pending CI**

## Commits (pass 11)

| SHA | Message |
|-----|---------|
| `d631eb2` | config: add EnvSnapshot.Initialized() for empty-env semantics |
| `a2ac15b` | app: route registry auth through invocation EnvSnapshot |
| `3462366` | app: snapshot-aware store prune scan roots |
| `3992093` | app: show all install warning sections independently |
| `ff49b37` | docs: pass 11 ambient-env audit and snapshot contracts |

## Gaps fixed

| # | Gap | Fix |
|---|-----|-----|
| 1 | `populated()` treated `[]string{}` as uninitialized | `EnvSnapshot.Initialized()`; initialized-empty never reads ambient |
| 2 | Registry auth via `os.Environ()` at fetch/resolve | `AuthToken(eff)` + `NewFromApp(eff, …)` without environ param |
| 3 | Store prune scan roots used ambient `MEW_HOME` | `DefaultStoreScanRoots(snap, projectRoot)` from invocation snapshot |
| 4 | Critical txn warnings hid non-critical/store sections | Independent section emitters with shared dedupe |

## Audit grep (pass 11)

| Pattern | Result |
|---------|--------|
| `os.Environ()` in resolver/registry/fetch/install paths | **Fixed** — snapshot-only after `app.New` |
| `AuthToken(.*environ` | **Removed** |
| `NewFromApp(.*environ` | **Removed** |
| `DefaultStoreScanRoots` without snapshot | **Fixed** — takes `config.EnvSnapshot` |
| `PruneStore` / `store_cmd` `os.Getenv("MEW_HOME")` | **Fixed** |
| `populated()` | **Removed** — `Initialized()` |
| Retained ambient (marked `// intentional:`) | `app.New` nil Env; `GlobalConfigPath`; `config_cmd` global writes; `execute.go` pre-snapshot flags; unit-test fallbacks in `envLookup`/`AuthToken` |

## Local gate results (2026-07-28, Windows — focused per commit)

| Command | Result |
|---------|--------|
| `go test ./internal/config/... ./internal/app/... -count=1` | **PASS** (commits 1–4) |
| `go test ./internal/config/... ./internal/app/... ./internal/resolver/... ./internal/registry/... ./internal/cli/... -count=1` | **PASS** (commit 2) |
| `go test ./internal/app/... -run StoreScan\|PruneStore -count=1` | **PASS** (commit 3) |
| `go test ./tests/integration/... -run StorePruneSnapshot -count=1` | **PASS** (commit 3) |
| `go test ./internal/app/... -run FormatInstallSummary -count=1` | **PASS** (commit 4) |
| Full `go test ./...`, race, lint, vuln | **Deferred** — verify-ci agent |

## MVP status (pass 11)

| MVP | Status |
|-----|--------|
| 0017 Transactional install | **Done** |
| 0020 Full resolver | **Done** |
| 0021 Lifecycle scripts | **Not started** — blocked on pass 11 CI green |

## Score

**9.40 / 10.0** — deduct 0.40 pending full-repo test + 21-job CI confirmation (no carry-forward from pass 10).

## Decision

**PENDING** — implementation complete on `main`; verify-ci agent must run full gates and CI loop.
