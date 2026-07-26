---
name: "0019 Core MVP 10 — Isolated Virtual Store and Node Modules Layout"
overview: "Implement a pnpm/Nub-style isolated dependency layout that prevents phantom dependencies while retaining a compatibility hoisted mode."
todos:
  - id: isolated
    content: "Virtual store linker implementation"
    status: pending
  - id: phantom
    content: "Undeclared dep access prevention"
    status: pending
  - id: mode
    content: "Hoisted vs isolated config switch"
    status: pending
  - id: bins
    content: "Isolated .bin shim paths"
    status: pending
  - id: settings
    content: "m.lock linker persistence"
    status: pending
  - id: tests
    content: "pnpm layout comparison fixtures"
    status: pending
  - id: windows
    content: "Junction and long-path handling"
    status: pending
isProject: false
---

# 0019 - Core MVP 10 — Isolated Virtual Store and Node Modules Layout

## Source contract

Canonical scope lives in [`plans/0019-isolated-linker.md`](../0019-isolated-linker.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0018

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0018
2. Read source contract `plans/0019-isolated-linker.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/linker/isolated, internal/linker, internal/store, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Isolated install blocks requiring undeclared dependencies
2. pnpm-simple fixture layout matches expected structure
3. Hoisted mode still works via --linker=hoisted
4. Isolated .bin shims execute correctly on Windows
5. Linker mode persists in m.lock settings

Suggested commands (once code exists):

```powershell
go test ./internal/linker/isolated/... -count=1
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
