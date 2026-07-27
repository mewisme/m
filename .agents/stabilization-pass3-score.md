# Stabilization Pass 3 — Quality Scorecard

**Baseline commit:** `b1fe1977b13ccaccab9a785d251b3fb913a7972e`  
**Final local verification:** 2026-07-27 (Windows)  
**Gate:** ≥ 9.0 to unblock MVP 0021

## Score (phases 1–15 complete)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.45 | `BeginMutation` preflight; atomic `RestoreSnapshot`; `PolicyFromEffective` parity; full-identity incremental merge | — |
| Transaction durability | 1.5 | 1.5 | ABA tombstone takeover; owner-safe release; journal phases; 57-boundary crash matrix (install/update/restore) | — |
| Store integrity | 1.5 | 1.45 | cross-process index lock; partial index status fallback; portable path collision reject | — |
| Security | 1.0 | 0.95 | fsx guards; tree manifest path validation; collision policy | — |
| Cross-platform | 1.0 | 0.85 | `go test ./...` green on Windows; CI adds `race-windows`, `crash-integration`, fsx ABA in `platform-lock` | Actions URLs not recorded this session |
| Test quality | 1.0 | 1.0 | `txn_crash_test`, `snapshot_crash_test`, `update_crash_test`, proc + journal_crash suites | — |
| Maintainability | 0.75 | 0.72 | single `BeginMutation`; shared `fsx/lockdir`; unified policy loader | — |
| Docs/status | 0.5 | 0.48 | transaction/store/resolver/lockfile/testing/errors synced; CHECKLIST blocks 0021 | — |
| Performance | 0.25 | 0.25 | bounded workers unchanged | — |

**Estimated total:** **9.15 / 10.0**

## Automatic blockers (plan gap table)

All 13 hard-fix blockers from pass 4 plan are addressed in code (phases 1–12) and covered by tests/docs (phase 13–15). No remaining automatic blockers identified locally.

## Local gate results (2026-07-27, Windows)

| Command | Result |
|---------|--------|
| `go test ./... -count=1` | **PASS** (integration ~72s) |
| `go test ./tests/integration/... -run Crash -count=1` | **PASS** (~62s) |
| `go test -race ./... -count=1` | **SKIP** (no gcc/CGO on host) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (after removing unused `loadJournal`) |
| `govulncheck ./...` | **PASS** (no vulnerabilities) |

## CI jobs added (phase 14)

| Job | Platform | Command |
|-----|----------|---------|
| `race-windows` | windows-latest | `go test -race ./internal/transaction/... ./internal/store/... ./internal/fsx/...` |
| `crash-integration` | ubuntu-latest | `go test ./tests/integration/... -run Crash` |
| `platform-lock` (updated) | matrix | includes `./internal/fsx/... -run 'Proc\|Identity\|Takeover\|Import\|ABA'` |

**CI URLs:** pending first green run on `main`/PR after push.

## Phase status

- [x] Phase 0 — baseline recorded, 0021 blocked
- [x] Phases 1–12 — hard fixes (preflight, ABA locks, atomic restore, policy, store, phases, cleanup)
- [x] Phase 13 — full recovery crash matrix
- [x] Phase 14 — CI jobs + local gate (race deferred on Windows host)
- [x] Phase 15 — docs + scorecard

## Decision

**BLOCKED** for MVP 0021 until GitHub Actions confirms green `crash-integration`, `race-windows`, and full matrix jobs. Local evidence supports **9.15** score; cross-platform deduction remains until CI URLs are recorded.
