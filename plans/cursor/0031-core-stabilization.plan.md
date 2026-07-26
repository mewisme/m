---
name: "0031 Core MVP 22 — Package-Manager Core Stabilization Gate"
overview: "Certify the package-manager core for daily use before beginning runner and runtime parity work."
todos:
  - id: conformance
    content: "Full core conformance run"
    status: pending
  - id: soak
    content: "Extended install soak tests"
    status: pending
  - id: defects
    content: "P0/P1 stabilization fixes"
    status: pending
  - id: doctor
    content: "m doctor health command"
    status: pending
  - id: docs
    content: "core-certification evidence doc"
    status: pending
  - id: freeze
    content: "Public API and m.lock schema freeze"
    status: pending
  - id: unblock
    content: "Runner MVP dependency sign-off"
    status: pending
isProject: false
---

# 0031 - Core MVP 22 — Package-Manager Core Stabilization Gate

## Source contract

Canonical scope lives in [`plans/0031-core-stabilization.md`](../0031-core-stabilization.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- Zero known data-loss or silent-integrity issue.
- Certified read/write matrices are accurate and enforced by tests.
- Transactional recovery succeeds for every injected commit interruption.
- Core commands are documented and machine-readable output is versioned.
- Performance and resource budgets are enforced in CI.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0010, 0011, 0012, 0013, 0014, 0015, 0016, 0017, 0018, 0019, 0020, 0021, 0022, 0023, 0024, 0025, 0026, 0027, 0028, 0029, 0030

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0010, 0011, 0012, 0013, 0014, 0015, 0016, 0017, 0018, 0019, 0020, 0021, 0022, 0023, 0024, 0025, 0026, 0027, 0028, 0029, 0030
2. Read source contract `plans/0031-core-stabilization.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/app, tests/conformance, tests/integration, docs/core-certification.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Full core conformance suite passes on all CI platforms
2. m doctor reports healthy state on clean fixture project
3. No open P0/P1 defects in PM core scope
4. core-certification.md published with test evidence
5. 0040 can depend on install/layout interfaces without breakage

Suggested commands (once code exists):

```powershell
go test ./internal/app/... -count=1
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
