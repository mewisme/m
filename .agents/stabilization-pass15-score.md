# Stabilization Pass 15 — Quality Scorecard

**Session:** Stabilization Pass 15 (MVP 0023 lock bridge Phases 8–14)  
**Baseline:** `e79cebf0c8be7e3e3e1734dbb181fb29e5e8e40e`  
**Final SHA:** `3b5c6df` (pending CI verification after push)  
**Docs tip SHA:** `3b5c6df`  
**Branch:** `main`  
**Gate:** ≥ 8.5

## Commits (pass 15 phases 8–14)

| SHA | Message |
|-----|---------|
| `bd823b1` | app: txn-only incumbent writes and migration fixes |
| `3d17b67` | compat/nub: policy-based adapter from incumbent detection |
| `4017388` | compat/pnpm: input limits, duplicate-key rejection, and fuzz target |
| `82f0097` | fixtures: generated lock bridge conformance corpus |
| `2973730` | ci: pnpm 9/10/11 and nub lock bridge conformance jobs |
| `b1da56b` | cli: lock detection output and migration reports |
| `3b5c6df` | docs: pass15 lock bridge support matrix and conformance notes |

(Phases 1–7 landed in parallel as `1d6c5c2` and earlier on `main`.)

## Pinned pnpm versions

From `tools/conformance/pnpm-versions.env`:

| Major | Version |
|-------|---------|
| 9 | 9.15.9 |
| 10 | 10.14.0 |
| 11 | 11.0.2 |

## Defect closure (baseline e79cebf)

| ID | Status | Evidence |
|----|--------|----------|
| D7 Live incumbent write bypass | **closed** | `WriteLock` rejects nub/pnpm; `writeStagedExtLock` + txn tests |
| D8 Weak fixtures / no CI conformance | **closed** | `fixtures/locks/generated/`, 4 CI jobs, `lock_bridge_pnpm_test.go` |
| D4 False-certain detection | **closed** (phase 5) | `detect.go` evidence order; v10/v11 structural markers in fixtures |
| D1–D3, D5–D6 | **closed** (phases 1–7) | `1d6c5c2` + compat/pnpm tests |

## Score

| Category | Max | Awarded | Evidence |
|----------|-----|---------|----------|
| Correctness | 2.5 | 2.35 | txn-only writes, migration fail-closed, detection wiring |
| Transaction durability | 1.5 | 1.45 | commit-interrupt + encode-failure injection tests |
| Conformance / fixtures | 1.5 | 1.4 | generated corpus + metadata; pnpm 9/10/11 + nub CI |
| Security | 1.0 | 0.9 | limits, duplicate keys, FuzzDecodePnpmLock |
| CLI / reporting | 0.75 | 0.7 | detection output, migration report JSON |
| Test quality | 1.0 | 0.9 | app txn, conformance, integration lock_bridge |
| Docs / inventory | 0.5 | 0.5 | lockfile matrix, testing, CHECKLIST note |
| Cross-platform | 0.25 | 0.2 | Windows local pass; CI pending on push |

**Total:** **8.6 / 10.0**

## Local gate results (Windows, 2026-07-28)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./internal/app/... ./internal/compat/pnpm/... ./internal/lockfile/... -count=1` | **PASS** |
| `go test ./tests/conformance/... -run LockBridge -count=1` | **PASS** |
| `go test ./tests/integration/... -run LockBridge -count=1` | **PASS** |
| `go test ./... -count=1` | **PARTIAL** (pre-existing Windows: `internal/store` symlink privilege) |
| `go test -race ./... -count=1` | **SKIP** (Windows: CGO disabled) |
| `go vet ./...` | **PASS** (scoped packages) |
| `golangci-lint run ./...` | not re-run this session |
| `govulncheck ./...` | not re-run this session |

## CI

**Pending:** push `3b5c6df` → inspect all 25 jobs (21 existing + 4 conformance).

## Verdict

**BLOCKED** until final SHA CI green on all required jobs.
