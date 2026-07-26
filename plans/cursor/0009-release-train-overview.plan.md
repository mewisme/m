---
name: "0009 Release Train and MVP Dependency Graph"
overview: "Define the ordered delivery train from package-manager core through complete Nub parity and Mew extensions."
todos:
  - id: graph
    content: "Publish MVP dependency graph"
    status: pending
  - id: channels
    content: "Define alpha/beta/rc/stable"
    status: pending
  - id: gates
    content: "Experimental + stop-the-line rules"
    status: pending
  - id: map
    content: "Inventory features to milestones"
    status: pending
  - id: docs
    content: "Release-train overview"
    status: pending
  - id: sync
    content: "Keep INDEX.md and release-train aligned"
    status: pending
isProject: false
---

# 0009 - Release Train and MVP Dependency Graph

## Source contract

Canonical scope lives in [`plans/0009-release-train-overview.md`](../0009-release-train-overview.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0001, 0002, 0003, 0008

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0001, 0002, 0003, 0008
2. Read source contract `plans/0009-release-train-overview.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/release-train.md, plans/INDEX.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Every inventory feature has a milestone
2. Stabilization gates 0031/0046/0057 cannot start early
3. Stop-the-line criteria include corruption and integrity failures
4. Milestone graph has no cycles and matches INDEX.md ordering

Suggested commands (once code exists):

```powershell
go test ./docs/release-train.md/... -count=1
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
