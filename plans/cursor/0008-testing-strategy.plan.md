---
name: "0008 Testing, Fixtures, Fuzzing, and Conformance Strategy"
overview: "Build the test infrastructure required to port behavior safely and verify package-manager compatibility without depending on public registries."
todos:
  - id: layout
    content: "Freeze fixtures/ and tests/ layout"
    status: pending
  - id: registry
    content: "Local fixture registry helper"
    status: pending
  - id: home
    content: "Clean-home isolation helpers"
    status: pending
  - id: fuzz
    content: "List parser fuzz targets"
    status: pending
  - id: docs
    content: "Testing strategy guide"
    status: pending
  - id: diff
    content: "Differential comparison report schema"
    status: pending
isProject: false
---

# 0008 - Testing, Fixtures, Fuzzing, and Conformance Strategy

## Source contract

Canonical scope lives in [`plans/0008-testing-strategy.md`](../0008-testing-strategy.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0004, 0007

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0004, 0007
2. Read source contract `plans/0008-testing-strategy.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/testkit, tests/conformance, tests/integration, fixtures/registry, fixtures/projects
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Tests never require public registry access
2. Clean-home tests do not touch developer global state
3. Fixture checksums verified on load
4. Differential harness smoke test passes on pinned Nub revision when available

Suggested commands (once code exists):

```powershell
go test ./internal/testkit/... -count=1
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
