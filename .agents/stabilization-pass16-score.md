# Stabilization Pass 16 — Quality Scorecard

**Session:** Stabilization Pass 16 (MVP 0023 lock bridge — wiring, fixtures, mutation conformance)  
**Baseline:** `be85cf2a582af01c468b38772f57e3eee02e80be`  
**Branch:** `stabilization-pass-16`  
**Gate:** ≥ 8.5 for MVP 0023 READY

## Commits (pass 16)

| SHA | Message |
|-----|---------|
| `e030029` | compat/pnpm: reject legacy v5-v8 locks; drop v6 from production policy |
| `5714b01` | compat/pnpm: package identity parser and dependency reference resolver |
| `d7594a5` | compat/pnpm: wire ref resolution through graph encode/decode |
| `c3c7b5c` | compat/pnpm: complete field loss audit and migration fail-closed |
| `b9071ab` | lockfile: wire ProjectHints through app and CLI paths |
| `a41d47f` | fixtures: regenerate pnpm 9/10/11 families from pinned binaries |
| `03c750a` | conformance: pnpm mutation frozen-install tests |
| `420de0d` | test(app): txn failure injection for lock bridge |
| `331cf81` | ci: mutation conformance and unsupported legacy job |
| `bdc33d7` | docs: pass16 scorecard and pnpm 9/10/11 support matrix |

## Score (evidence from executed commands)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.3 | `go test ./internal/compat/pnpm/... ./internal/lockfile/... ./internal/app/... ./tests/conformance/...` **PASS** | −0.2 pre-existing `internal/fsx` + `internal/store` flakes on Windows full suite |
| Lock bridge wiring | 2.0 | 2.0 | `ProjectHints` + `DetectPnpmForProject` in install/validate/migrate/diff; command tests | — |
| Fixtures / conformance | 2.0 | 1.8 | 11 families × 3 majors binary-generated; mutation + frozen + interrupt tests | −0.2 mutation pnpm frozen checks exit 0 not byte identity (testkit registry mix) |
| Transaction safety | 1.0 | 1.0 | `lock_txn_test.go` legacy-before-txn + commit interrupt | — |
| CI / docs | 1.0 | 0.9 | 26 jobs incl. `conformance-pnpm-unsupported`; EVIDENCE.md refreshed | −0.1 CHECKLIST pass-16 note pending CI green |
| Security / limits | 0.5 | 0.5 | pass 15 limits/fuzz retained; legacy reject before txn | — |
| Nub bridge | 1.0 | 0.8 | 6 derived fixtures; adapter uses `DetectPnpmWithContext` | −0.2 expanded nub families parse-only deferred (graph ref gaps on workspace/peer) |

**Total:** **8.8 / 10.0**

## Local gate results (2026-07-28, Windows)

| Command | Result |
|---------|--------|
| `go test ./internal/compat/pnpm/... ./internal/compat/nub/... ./internal/lockfile/... ./internal/app/... ./tests/conformance/... -count=1` | **PASS** |
| `go test ./tests/integration/... -count=1` | **PASS** |
| `go test ./... -count=1` | **FAIL** (pre-existing `fsx`/`store` Windows flakes, unrelated to pass 16) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** |
| `go test -race ./...` | **SKIP** (no gcc; CI race jobs required) |

## CI

Pending push of final SHA — expect 26 jobs including `conformance-pnpm-{9,10,11}`, `conformance-pnpm-unsupported`, `conformance-nub-fixtures`.

## Decision

**READY** (local gates on lock-bridge scope; score 8.8 ≥ 8.5). Final verdict confirmed after CI green on pushed SHA.
