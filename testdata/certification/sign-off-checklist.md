# PM core sign-off checklist (0031)

0087-aligned definition-of-done checklist for the package-manager core
(MVPs **0010–0030**). Global program DoD (**0087**) remains separate; this
checklist covers only PM-core stabilization.

Evidence index: [`docs/core-certification.md`](../../docs/core-certification.md).

## Behavior and correctness

- [x] Full `go test ./... -count=1` passes on Linux, macOS, and Windows (`test` CI job)
- [x] No open P0/P1 defects in PM core scope (integrity, crash recovery, lock adapters)
- [x] Lock-bridge inventory cases marked `certified` in `tests/conformance/inventory.json`
- [x] Fixture provenance verifier green (`fixture-verify` / `verify-fixtures`)
- [x] Unsupported pnpm majors fail before graph conversion (`conformance-pnpm-unsupported`)

## Crash safety and transactions

- [x] All crash-integration shards green (`crash-integration`, Windows shards)
- [x] Crash-shard verifier passes (`crash-shard-verify`)
- [x] Interrupted install leaves incumbent bytes restorable (`tests/integration` crash tags)
- [x] Patch apply is fail-closed and sandboxed ([`docs/evidence/core/pass20-security-controls.md`](../../docs/evidence/core/pass20-security-controls.md))

## Conformance and compatibility

- [x] pnpm 9 / 10 / 11 mutation suites green (`conformance-pnpm-9/10/11`)
- [x] npm, bun, Yarn Classic + Berry fixture bridges green (`conformance-npm/bun/yarn`)
- [x] Nub derived fixtures validate (`conformance-nub-fixtures`)
- [x] `m conformance run core` passes locally and in `core-stabilization` CI
- [x] Known limitations documented in `docs/core-certification.md` (Nub binary, Yarn Berry PnP)

## Health, soak, and performance

- [x] `m doctor` reports healthy on clean-home fixture (`tests/integration/doctor_test.go`)
- [x] Soak loop passes at CI count (`install_loop.py --count 10`)
- [x] Manual soak documented at 100+ iterations on `fixtures/soak/representative-projects/`
- [x] Install bench regression gate green (`bench-regression` / `check_regression.py`)

## Security (PM-core subset of 0082)

- [x] Archive extraction bounded and path-safe (`internal/archive`, evil-tarball tests)
- [x] Registry auth redacted in diagnostics (`docs/security-pm-core.md`)
- [x] Patch sandbox and store copy-on-write (`docs/patch.md`, pass 20)
- [x] Policy gate blocks denied packages/licenses (`m policy check`, install hook)
- [x] Provenance fixture verification (`m verify provenance`; no live Sigstore claim)

## Pass 32 hardening (evidence-backed)

- [x] Content-addressed blob store verifies on read, write, and existence (`internal/store/verified.go`)
- [x] Core certification fail-closed: zero-match, forced-skip, and missing-tool skips fail (`cert-negative-probes` CI)
- [x] npm incumbent semantic mutation rejected (`TestLockBridgeNpmMutationRejected`)
- [x] `m pack` root containment; symlinks and escape paths rejected (`internal/pack/sandbox.go`)
- [x] OSV multi-interval range matching; `m audit --fail-on` exit policy (`internal/advisory/range.go`)
- [x] Provenance explicit trust policy in production (`TrustConfiguredKey`; fixture DSSE ≠ Sigstore)
- [x] `m publish --provenance` fails before upload without configured provider
- [x] Capsule atomic verified create and quarantined restore
- [x] SBOM graph `dependencies` / `bom-ref` / SPDX `DEPENDS_ON` edges
- [x] Bench multi-sample median/p95 regression (`benchmarks/install-baseline.json` schema v2)

## Documentation and contracts

- [x] `docs/core-certification.md` published with CI job mapping
- [x] `docs/schema-freeze.md` published (m.lock + JSON report schemas)
- [x] `docs/security-pm-core.md` published (PM threat subset)
- [x] `docs/pm-commands.md` indexes shipped PM commands (0026–0030)
- [x] `features/inventory.json` — `foundation.core-stabilization` → `shipped`

## Release interfaces for 0040

- [x] Install/layout interfaces documented as stable for runner MVPs
- [x] No breaking change to `m.lock` or install result JSON without ADR
- [x] Schema freeze notice acknowledged in `docs/core-certification.md`

## Waivers and exceptions

Record any active waiver here (must include owner, expiry, and link to issue):

| Waiver | Owner | Expires | Issue |
|---|---|---|---|
| _(none)_ | — | — | — |
