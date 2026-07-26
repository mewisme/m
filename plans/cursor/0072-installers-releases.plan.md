---
name: "0072 Distribution MVP 1 — Releases, Installers, and Package Channels"
overview: "Produce signed, reproducible multi-platform releases and safe install/update paths for direct download and common package channels."
todos:
  - id: goreleaser
    content: "Reproducible release matrix"
    status: pending
  - id: sign
    content: "Checksums, signatures, SBOM, provenance"
    status: pending
  - id: installers
    content: "POSIX and PowerShell scripts"
    status: pending
  - id: channels
    content: "Homebrew/Scoop/Winget/npm defs"
    status: pending
  - id: upgrade
    content: "Self-update with rollback"
    status: pending
  - id: tests
    content: "Clean VM and tamper rejection"
    status: pending
isProject: false
---

# 0072 - Distribution MVP 1 — Releases, Installers, and Package Channels

## Source contract

Canonical scope lives in [`plans/0072-installers-releases.md`](../0072-installers-releases.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- All required tests pass on supported operating systems.
- No unresolved correctness, integrity, or data-loss issue remains.
- Public behavior and intentional deviations are documented.
- The next dependent MVP can consume stable interfaces without reaching into internals.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0031, 0046, 0057, 0062

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0031, 0046, 0057, 0062
2. Read source contract `plans/0072-installers-releases.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: cmd/m, cmd/mx, .goreleaser/, installers/, docs/install.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Install script verifies checksum/signature before executing m
2. m upgrade rolls back safely on failure
3. Release artifacts include m and mx for all supported platforms
4. Channel manifests are signed and versioned
5. SBOM and provenance published per release

Suggested commands (once code exists):

```powershell
go test ./cmd/m/... -count=1
```

Adjust package paths to those actually created. Always include a clean-home fixture test for install-family work.

## Handoff

Before submitting work provide:

1. Behavior summary and compatibility target
2. Files and public interfaces changed
3. Test/benchmark/static-analysis commands
4. Known gaps and platform limits
5. Determinism evidence for generated files
6. Rollback note for persistent-format changes
