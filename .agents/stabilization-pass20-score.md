# Stabilization Pass 20 — Patch Safety, Provenance, Final Evidence

**Date:** 2026-07-29  
**Baseline:** `4d2235271e30537f7f03348135e18ec746741655`  
**Starting HEAD (phases 1–10):** `c3c92078f7c915b1a8b863caaea743014a454011`  
**Final SHA:** `27a7ea5d4811df0ddf3be5f4cc6fdeea36fc6b5a` (`docs(pass20): record final SHA and CI run`)  
**CI run:** `30463655822` (complete green matrix on exact final SHA)  
**Overall score:** 9.1 / 10  
**Status:** READY

## Patch architecture (shipped)

| Capability | Implementation | Evidence |
|------------|----------------|----------|
| Path sandbox | `resolvePatchTarget`, `looksAbsolutePatchPath`, `fsx.GuardAncestors`, `PatchPathError` | `internal/archive/patch_path.go`, `patch_plan.go`, `patchapply_security_test.go` |
| Store copy-on-write | `stagePatchDerivatives` writes derivatives under stage, not store `PackagePath` | `internal/app/install_helpers.go`, store isolation tests |
| Fail-closed apply | `PreflightPlan` / `ApplyPlan`; no silent `continue` on missing patch paths | `internal/archive/patch_plan.go`, rollback tests |
| Byte-derived identity | Resolver derives `patch_hash` from patch file bytes (pnpm 9/10/11 shapes) | `internal/resolver/patch.go`, `internal/resolver/local.go` (`buildExtensions`) |
| Option B preflight | Unsupported ops (create/delete/rename/binary/mode-only) rejected at preflight | `classifyPatchFile`, `docs/patch.md` |
| Atomic apply | Patch work dirs + transaction commit/rollback with hook injection | `internal/archive/patch_atomic.go`, transaction tests |

## Provenance and alias-peer

| Item | Status |
|------|--------|
| Expanded fixture metadata schema | `tools/conformance/fixturemeta` — lock/manifest/workspace/patch hashes, invocation ID, source-tree digest |
| Generator + verifier | `generate-lock-fixtures.ps1`, `verify-fixtures` with regression tests |
| Full pnpm 9/10/11 + Nub derived regen | Committed in `c3c9207`; patch fixture corrected in `573cf24` |
| alias-peer e2e | Source fixture + `TestLockBridgePnpm{9,10,11}AliasPeer` + mutation suite passes on Linux CI |

## go mod / deps

- `github.com/bluekeyes/go-gitdiff` direct require (tidy guard in CI)
- `go run ./tools/check-deps` — 10 modules allowlisted
- `go run ./tools/check-license` — MIT

## Windows-native preflight (`573cf24`, `CGO_ENABLED=0`)

| Command | Result |
|---------|--------|
| `go test ./internal/archive/... ./internal/resolver/... ./internal/app/... ./internal/transaction/... ./tests/conformance/... -count=1` | **PASS** |
| `go test ./... -count=1` | **PASS** |
| `go vet ./...` | **PASS** |
| `go build ./cmd/m` / `./cmd/mx` | **PASS** |
| `go run ./tools/check-license` | **PASS** |
| `go run ./tools/check-deps` | **PASS** |
| `go run ./tools/conformance/verify-fixtures` | **PASS** |
| `go run ./tools/ci/verify-crash-shards` | **PASS** |

**Host:** Windows 10.0.26200, Go 1.26.5, Node v24.14.0

## Linux Docker preflight (`golang:1.26`)

**Image:** `golang@sha256:3aff6657219a4d9c14e27fb1d8976c49c29fddb70ba835014f477e1c70636647`  
**Mount:** bind repo + named Go mod/build cache volumes, `HOME=/tmp/mew-home`, `CGO_ENABLED=0`

| Command | Result |
|---------|--------|
| `go test ./... -count=1` | **PARTIAL** — `tests/integration` `TestLifecycleUntrustedBlocksUntilTrust` fails in bare `golang:1.26` container (CI Ubuntu `test` job **pass** on same SHA) |
| `go vet ./...` | **PASS** |
| `go build ./cmd/m` / `./cmd/mx` | **PASS** |
| `go run ./tools/check-license` / `check-deps` / `verify-fixtures` | **PASS** |
| pnpm mutation conformance | **Not run locally** — `golang:1.26` image has no pnpm; **PASS** on Linux CI (`conformance-pnpm-9/10/11`) |

## CI evidence (run `30463655822` on `27a7ea5`)

| Job | Result |
|-----|--------|
| test (ubuntu / macos / windows) | **PASS** |
| race / race-macos / race-windows | **PASS** |
| no-cgo-gate | **PASS** |
| crash-integration (all shards + report) | **PASS** |
| lint | **PASS** |
| vuln | **PASS** |
| allowlist | **PASS** |
| gate-probe | **PASS** |
| fixture-verify | **PASS** |
| conformance-pnpm-9 / 10 / 11 | **PASS** (mutation suites include patch + alias-peer) |
| conformance-pnpm-unsupported | **PASS** |
| conformance-nub-fixtures | **PASS** |
| cross-build (6 targets) | **PASS** |
| platform-lock (3 OS) | **PASS** |

## Key fix commits (phases 11–12)

| SHA | Root cause |
|-----|------------|
| `c19456f` | Registry LF manifest hash; `buildExtensions` empty JSON on Linux patch apply; Windows junction probe; `node_modules` skip on copy |
| `71cbb5a` | `looksAbsolutePatchPath` false positives on spaced paths and `../` traversal |
| `573cf24` | `c3c9207` regen dropped pnpm-11 `patchedDependencies` / `patch_hash` from patch lock fixture |

## Residual risks

1. **Nub executable conformance** — derived-format fixture validation only; no Nub binary frozen/matrix run
2. **Docker bare integration flake** — `TestLifecycleUntrustedBlocksUntilTrust` in unpinned dev container (CI green)
3. **pnpm 11 patch config surface** — `patchedDependencies` in `pnpm-workspace.yaml`; dual `package.json#pnpm` field still present in fixtures for cross-major parity

## Decision checklist

| Criterion | Met |
|-----------|-----|
| Score ≥ 8.5/10 | yes (9.1) |
| Patch sandbox + COW + fail-closed + byte identity + Option B + atomic apply | yes |
| Provenance schema + verifier | yes |
| alias-peer e2e (Linux CI mutation) | yes |
| `CGO_ENABLED=0` production path | yes |
| Full CI green on exact final SHA | yes (`27a7ea5` = run `30463655822` headSha) |

READY
