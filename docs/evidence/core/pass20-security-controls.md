# Pass 20 security controls (evidence)

Controls shipped during stabilization pass 20. This file lists **controls and
implementation references only** — not historical scores.

## Patch apply

- Fail-closed path validation and sandboxed extraction
- Implementation: `internal/archive/patch_path.go`, `internal/archive/patch_plan.go`
- Tests: `internal/archive/patchapply_security_test.go`

## Store integrity

- Verified put/get/exists with corrupt quarantine
- Implementation: `internal/store/verified.go`
- Tests: `internal/store/verified_test.go`

## Provenance

- Explicit `TrustConfiguredKey` in production; exact package binding
- Implementation: `internal/provenance/trust.go`, `internal/app/provenance.go`
- `m publish --provenance` fails closed before upload without provider

## SBOM export

- Graph `dependencies` / `bom-ref` / SPDX `DEPENDS_ON`
- Implementation: `internal/sbom/sbom.go`
- Golden fixtures under `testdata/sbom/`

## Capsule archives

- Atomic verified create; quarantined restore
- Implementation: `internal/capsule/archive.go`

## OSV advisory ranges

- Multi-interval semver state machine; `m audit --fail-on`
- Implementation: `internal/advisory/range.go`
- Tests: `internal/advisory/range_test.go`

## Pack sandbox

- Root containment; symlink/reparse rejection; size limits
- Implementation: `internal/pack/sandbox.go`

See also [`docs/security-pm-core.md`](../../security-pm-core.md) and
[`docs/core-certification.md`](../../core-certification.md).
