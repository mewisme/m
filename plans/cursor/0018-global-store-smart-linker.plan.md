---
name: "0018 Core MVP 9 — Global Content Store and Smart Filesystem Planner"
overview: "Introduce an immutable global content-addressable store and automatically choose safe hardlink, reflink, copy, symlink, or junction strategies per filesystem."
todos:
  - id: store
    content: "Content-addressed global store"
    status: pending
  - id: planner
    content: "Filesystem-aware link planner"
    status: pending
  - id: import
    content: "Verified blob import"
    status: pending
  - id: prune
    content: "Store garbage collection"
    status: pending
  - id: probe
    content: "Platform link capability detection"
    status: pending
  - id: integrate
    content: "Hoisted linker store backend"
    status: pending
  - id: tests
    content: "Cross-platform link fixtures"
    status: pending
isProject: false
---

# 0018 - Core MVP 9 — Global Content Store and Smart Filesystem Planner

## Source contract

Canonical scope lives in [`plans/0018-global-store-smart-linker.md`](../0018-global-store-smart-linker.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0014, 0017

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0014, 0017
2. Read source contract `plans/0018-global-store-smart-linker.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/store, internal/linker, internal/linker/planner, internal/fetch
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Identical package imported twice shares one store blob
2. Link planner selects copy on cross-device install
3. Store prune removes only unreferenced blobs
4. Corrupt store entry is detected and re-fetched
5. m store path reports configured location

Suggested commands (once code exists):

```powershell
go test ./internal/store/... -count=1
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
