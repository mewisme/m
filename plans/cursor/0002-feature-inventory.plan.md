---
name: "0002 Complete Feature Inventory and Parity Matrix"
overview: "Maintain a complete, testable inventory of Nub capabilities and Mew extensions, organized by module and implementation milestone."
todos:
  - id: schema
    content: "Freeze inventory JSON schema"
    status: pending
  - id: extract
    content: "Populate Nub + Mew feature rows"
    status: pending
  - id: assign
    content: "Map features to primary MVPs"
    status: pending
  - id: cli
    content: "Specify m features output"
    status: pending
  - id: ci
    content: "Add inventory consistency gates"
    status: pending
  - id: docs
    content: "Generate human-readable inventory tables"
    status: pending
isProject: false
---

# 0002 - Complete Feature Inventory and Parity Matrix

## Source contract

Canonical scope lives in [`plans/0002-feature-inventory.md`](../0002-feature-inventory.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0001

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0001
2. Read source contract `plans/0002-feature-inventory.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/features, cmd/m, testdata/features
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Schema rejects inventory rows missing primary_mvp
2. Every INDEX MVP owns at least one inventory row
3. Mew extensions marked compatibility_class=extension
4. m features --format json validates against schema

Suggested commands (once code exists):

```powershell
go test ./internal/features/... -count=1
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
