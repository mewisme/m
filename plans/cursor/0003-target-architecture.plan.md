---
name: "0003 Target Architecture and Rust-to-Go Boundaries"
overview: "Define the final Go architecture, module boundaries, dependency direction, and the small embedded JavaScript surface required by Node extension APIs."
todos:
  - id: map
    content: "Finalize package map and forbidden edges"
    status: pending
  - id: ifaces
    content: "Freeze core interfaces"
    status: pending
  - id: boundary
    content: "Document Node augmentation + JS embed rules"
    status: pending
  - id: archcheck
    content: "Specify import-graph tests"
    status: pending
  - id: docs
    content: "Publish architecture package listing"
    status: pending
  - id: nubmap
    content: "Map Nub crates to Mew packages"
    status: pending
isProject: false
---

# 0003 - Target Architecture and Rust-to-Go Boundaries

## Source contract

Canonical scope lives in [`plans/0003-target-architecture.md`](../0003-target-architecture.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0001, 0002

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0001, 0002
2. Read source contract `plans/0003-target-architecture.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: cmd/m, cmd/mx, internal/app, internal/cli, internal/config, internal/manifest, internal/project, internal/workspace
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Every AGENTS.md package appears in the map
2. No cyclic dependency in the documented graph
3. JS surface limited to Node extension APIs
4. Transaction boundary documented for all install-family mutations

Suggested commands (once code exists):

```powershell
go test ./cmd/m/... -count=1
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
