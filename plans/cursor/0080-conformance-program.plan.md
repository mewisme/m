---
name: "0080 Cross-Cutting — Compatibility and Conformance Program"
overview: "Continuously certify Mew against Nub and incumbent package managers by behavior, file semantics, and runtime outcomes."
todos:
  - id: matrix
    content: "Target matrix per PM major"
    status: pending
  - id: harness
    content: "Differential runner"
    status: pending
  - id: ci
    content: "Certified suite gates"
    status: pending
  - id: report
    content: "Machine-readable reports"
    status: pending
  - id: docs
    content: "Add-target guide"
    status: pending
  - id: waivers
    content: "Expiring waiver process"
    status: pending
isProject: false
---

# 0080 - Cross-Cutting — Compatibility and Conformance Program

## Source contract

Canonical scope lives in [`plans/0080-conformance-program.md`](../0080-conformance-program.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0002, 0008

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0002, 0008
2. Read source contract `plans/0080-conformance-program.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: tests/conformance, internal/testkit/conformance, docs/conformance
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Certified suite green on supported OS matrix
2. Every compatibility target has pinned version + fixtures
3. Regressions block release train gates

Suggested commands (once code exists):

```powershell
go test ./tests/conformance/... -count=1
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
