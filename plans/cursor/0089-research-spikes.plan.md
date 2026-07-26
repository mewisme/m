---
name: "0089 Open Research Spikes and Decision Gates"
overview: "Resolve architecture questions that could invalidate later implementation before those MVPs freeze public contracts."
todos:
  - id: list
    content: "Spike inventory"
    status: pending
  - id: template
    content: "Spike report template"
    status: pending
  - id: gates
    content: "Blocking rules"
    status: pending
  - id: archive
    content: "Evidence archive"
    status: pending
  - id: sync
    content: "Update MVP open decisions"
    status: pending
  - id: docs
    content: "Spike index"
    status: pending
isProject: false
---

# 0089 - Open Research Spikes and Decision Gates

## Source contract

Canonical scope lives in [`plans/0089-research-spikes.md`](../0089-research-spikes.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0003, 0085

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0003, 0085
2. Read source contract `plans/0089-research-spikes.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/spikes/, .agents/threads/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Open spikes listed with owners and due dates
2. Blocking spikes prevent dependent MVP start
3. Resolved spikes recorded with decisions

Suggested commands (once code exists):

```powershell
go test ./docs/spikes//... -count=1
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
