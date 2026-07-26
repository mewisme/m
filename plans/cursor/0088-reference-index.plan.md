---
name: "0088 Reference Index and Research Sources"
overview: "Maintain the authoritative source list for Nub behavior, incumbent package-manager formats, Node APIs, registries, security standards, and Go implementation decisions."
todos:
  - id: nub
    content: "Pin Nub sources"
    status: pending
  - id: pm
    content: "Lock format sources"
    status: pending
  - id: node
    content: "Runtime API sources"
    status: pending
  - id: security
    content: "Standards index"
    status: pending
  - id: refresh
    content: "Update process"
    status: pending
  - id: docs
    content: "Publish reference index"
    status: pending
isProject: false
---

# 0088 - Reference Index and Research Sources

## Source contract

Canonical scope lives in [`plans/0088-reference-index.md`](../0088-reference-index.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0002, 0083

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0002, 0083
2. Read source contract `plans/0088-reference-index.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: plans/sources/, docs/references/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Authoritative source list exists
2. Nub commit pin recorded
3. Parity claims can cite a source entry

Suggested commands (once code exists):

```powershell
go test ./plans/sources//... -count=1
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
