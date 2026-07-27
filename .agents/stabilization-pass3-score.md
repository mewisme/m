# Stabilization Pass 5 — Quality Scorecard

**Session:** Hard Fix Pass 5 final integration (phases 11–13)  
**Final local verification:** 2026-07-27 (Windows, `F:\Project\package-managers\mew`)  
**Gate:** ≥ 9.0 to unblock MVP 0021 (local evidence met; CI URLs not recorded)

## Score (evidence from executed commands only)

| Category | Max | Awarded | Evidence | Deductions |
|----------|-----|---------|----------|------------|
| Correctness | 2.5 | 2.45 | `go test ./... -count=1` **PASS** (integration ~133s); mutation ordering + install/isolated suites green | — |
| Transaction durability | 1.5 | 1.5 | `txn_crash_test`, `snapshot_crash_test`, `update_crash_test`, `mutation_ordering_test` **PASS** without `-short` | — |
| Store integrity | 1.5 | 1.45 | `internal/contentid` SRI tests **PASS**; `internal/store` import/index proc tests **PASS**; `TestStoreMutationIsolation` **PASS** | — |
| Security | 1.0 | 0.95 | `treemanifest_security_test` **PASS**; path collision reject covered | — |
| Cross-platform | 1.0 | 0.70 | Windows local green; `go test -race ./...` **SKIP** (`-race requires cgo`; no gcc on host); **no CI URLs** | −0.30 pending Actions |
| Test quality | 1.0 | 1.0 | `internal/fsx` ABA proc (`TestTakeoverABAProc*`) **PASS**; full integration matrix **PASS** | — |
| Maintainability | 0.75 | 0.72 | `golangci-lint run ./...` **PASS** after removing ineffectual `idx` assign + unused `cleanupCodeForRelease` | — |
| Docs/status | 0.5 | 0.48 | `transaction.md`, `store.md`, `testing.md`, `errors.md`, `CHECKLIST.md` updated pass 5 | — |
| Performance | 0.25 | 0.25 | no regression signal in this pass | — |

**Estimated total:** **9.50 / 10.0** (local evidence only)

## Automatic blockers (12 hard-fix items)

| # | Area | Local status | Evidence |
|---|------|--------------|----------|
| 1 | Mutation preflight / ordering | **Fixed** | `mutation_ordering_test.go` PASS |
| 2 | ABA lock takeover | **Fixed** | `lockdir_aba_proc_test.go` PASS |
| 3 | Owner-safe release | **Fixed** | `lockdir_release_test.go` (in `go test ./internal/fsx`) PASS |
| 4 | Atomic snapshot restore | **Fixed** | `snapshot_restore_test.go` PASS |
| 5 | Policy parity / drift | **Fixed** | `policy_drift_test.go` PASS |
| 6 | Incremental merge identity | **Fixed** | `incremental_merge_test.go` PASS |
| 7 | Store index cross-process | **Fixed** | `index_proc_test.go` PASS |
| 8 | contentid / npm SRI keys | **Fixed** | `contentid_test.go` + store helper tests PASS |
| 9 | Journal phase sub-states | **Fixed** | crash matrix through publish/commit PASS |
| 10 | Windows directory sync | **Fixed** | `durability_windows.go` access-denied no-op; install commit PASS |
| 11 | macOS darwin lock (`x/sys/unix`) | **Compile-only locally** | `go build ./...` PASS; CI macOS job not run here |
| 12 | Phase 4 validate hook | **Assumed fixed** | `validate` phase in integration output; full suite PASS |

## Local gate results (2026-07-27, Windows)

| Command | Result |
|---------|--------|
| `gofmt -w` (changed files) | **PASS** |
| `go test ./... -count=1` | **PASS** (~135s; integration ~133s) |
| `go test ./tests/integration/... -count=1` | **PASS** (no `-short`) |
| `go test ./tests/integration/... -run Crash -count=1` | **PASS** (subset of full integration) |
| `go test ./internal/fsx/... ./internal/contentid/... -count=1 -run 'ABA\|SRI\|contentid\|Proc'` | **PASS** |
| `go test -race ./... -count=1` | **SKIP** — `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` (no gcc on host) |
| `go vet ./...` | **PASS** |
| `golangci-lint run ./...` | **PASS** (0 issues after lint fixes) |
| `govulncheck ./...` | **PASS** (no vulnerabilities) |

## CI jobs (phase 12)

Defined in `.github/workflows/ci.yml`:

| Job | Platform | Command |
|-----|----------|---------|
| `test` | ubuntu / macos / windows matrix | `go test ./... -count=1` |
| `race` | ubuntu-latest | `go test -race ./... -count=1` |
| `race-windows` | windows-latest | `go test -race ./internal/transaction/... ./internal/store/... ./internal/fsx/...` |
| `race-macos` | macos-latest | `go test -race ./internal/transaction/... ./internal/store/... ./internal/resolver/...` |
| `crash-integration` | ubuntu-latest | `go test ./tests/integration/... -count=1 -run Crash` |
| `platform-lock` | OS matrix | `go test ... -run 'Proc\|Identity\|Takeover\|Import\|ABA'` |

**CI URLs:** **not recorded** — requires push/PR to GitHub Actions. No green run verified in this session.

## Phase status (pass 5 scope)

- [x] Phase 11 — integration tests verified (mutation ordering, crash matrix, contentid/SRI, isolated, ABA proc)
- [x] Phase 12 — CI workflow reviewed; macOS darwin lock compiles; URLs pending push
- [x] Phase 13 — docs + scorecard updated with executed-test evidence

## Decision

**BLOCKED** for MVP 0021.

Local gate is green and score **9.50** meets the ≥ 9.0 bar, but the plan requires green CI URLs. Push to origin and confirm `test`, `crash-integration`, `race-windows`, `race-macos`, and `platform-lock` jobs before unblocking 0021.
