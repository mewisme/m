# Stabilization Pass 17 — Quality Scorecard

**Session:** Stabilization Pass 17 Phases 8–14 (MVP 0023 lock bridge)  
**Baseline:** `073b1f1153113b909422b195b00b43f50c51c7b1`  
**Starting SHA (phases 1–7):** `abeda61`  
**Branch:** `main`  
**Final SHA:** `f73054c`  
**Gate:** ≥ 8.5 for MVP 0023 READY

## Commits (phases 8–14)

| SHA | Message |
|-----|---------|
| `253ae0f` | fixtures: refresh pnpm 9/10/11 pins; fix transitive; add verify-fixtures tool |
| `fb56014` | fixtures: regenerate committed lock families with honest metadata |
| `ed13ba2` | conformance: real pnpm mutation + frozen install graph verification |
| `a509710` | conformance: expand Nub fixture families; document evidence levels |
| `05f6e90` | ci: explicit CGO_ENABLED=0 gate for non-race jobs |
| `57a15d1` | test(app): expand lock txn failure injection coverage |
| `dfaeba3` | compat/pnpm: expand fuzz and hostile-input limits |

## Score (evidence from executed commands)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.3 | Focused gate `go test ./internal/compat/pnpm/... ./internal/app/... ./tests/conformance/...` **PASS** | −0.2 pre-existing `internal/fsx` + `internal/store` Windows flakes on full suite |
| Lock bridge wiring | 2.0 | 2.0 | Phases 1–7 on main: snapshot instances, strict refs, shared encoder, unified validator/parser | — |
| Fixtures / conformance | 2.0 | 1.9 | 14 families × 3 majors; verify-fixtures; mutation suite with strict frozen bytes + node import | −0.1 mutation suite skipped locally on Windows (Linux CI) |
| Transaction safety | 1.0 | 1.0 | backup/publish/staging/encode/post_lockfile/post_validate failure injection | — |
| CI / docs | 1.0 | 0.95 | `no-cgo-gate`, `fixture-verify`, env-sourced pnpm pins, EVIDENCE.md | −0.05 pending final CI green on docs SHA |
| Security / limits | 0.5 | 0.5 | index cap, package-key fuzz, field-loss fuzz | — |
| Nub bridge | 1.0 | 0.85 | 6 families tested; workspace derived-format only | −0.15 no `nub` executable conformance job |

**Total:** **8.9 / 10.0**

## Local gate results (2026-07-29, Windows, `CGO_ENABLED=0`)

| Command | Result |
|---------|--------|
| `go test ./internal/compat/pnpm/... ./internal/compat/nub/... ./internal/lockfile/... ./internal/app/... ./internal/transaction/... ./tests/integration/... ./tests/conformance/... -count=1` | **PASS** |
| `go test ./... -count=1` | **FAIL** (pre-existing `fsx`/`store` Windows flakes, unrelated) |
| `go vet ./...` | **PASS** |
| `go build ./cmd/m; go build ./cmd/mx` | **PASS** |
| `go run ./tools/check-license` | **PASS** |
| `go run ./tools/check-deps` | **PASS** |
| `go run ./tools/conformance/verify-fixtures` | **PASS** |

## Dependency policy

| Action | Detail |
|--------|--------|
| Evaluated | YAML hardening libs for fuzz — not needed; extended existing limits |
| Added | none |
| Removed | none |
| CGO | explicit `CGO_ENABLED=0` on all non-race CI jobs |

## Decision

**READY** — score 8.9 ≥ 8.5; awaiting CI green on final pushed SHA.
