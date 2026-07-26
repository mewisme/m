---
name: "0090 Future Extensions Beyond Nub Parity"
overview: "Capture valuable post-parity ideas without allowing them to expand the ordered implementation critical path."
todos:
  - id: park
    content: "Maintain non-blocking backlog"
    status: pending
  - id: tag
    content: "Value/risk/effort tags"
    status: pending
  - id: promote
    content: "Promotion checklist"
    status: pending
  - id: guard
    content: "Keep off critical path"
    status: pending
  - id: review
    content: "Periodic grooming"
    status: pending
  - id: docs
    content: "Future backlog doc"
    status: pending
isProject: false
---

# 0090 - Future Extensions Beyond Nub Parity

## Source contract

Canonical scope lives in [`plans/0090-future-backlog.md`](../0090-future-backlog.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0087

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0087
2. Read source contract `plans/0090-future-backlog.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/backlog/future.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Backlog explicitly non-blocking
2. No critical-path MVP lists 0090 as required predecessor
3. Promotion requires human decision

Suggested commands (once code exists):

```powershell
go test ./docs/backlog/future.md/... -count=1
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
