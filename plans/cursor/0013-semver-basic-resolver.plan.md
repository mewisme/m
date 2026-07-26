---
name: "0013 Core MVP 4 — npm Semver and Basic Dependency Resolver"
overview: "Resolve registry dependencies and transitive dependencies using npm-compatible semver and produce a deterministic canonical graph with decision traces."
todos:
  - id: semver
    content: "npm-compatible range parsing"
    status: pending
  - id: resolve
    content: "Direct dependency resolution"
    status: pending
  - id: transitive
    content: "Recursive graph expansion"
    status: pending
  - id: determinism
    content: "Stable graph ordering"
    status: pending
  - id: cycles
    content: "Cycle detection and reporting"
    status: pending
  - id: trace
    content: "Decision trace emission"
    status: pending
  - id: tests
    content: "Semver and graph golden fixtures"
    status: pending
isProject: false
---

# 0013 - Core MVP 4 — npm Semver and Basic Dependency Resolver

## Source contract

Canonical scope lives in [`plans/0013-semver-basic-resolver.md`](../0013-semver-basic-resolver.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0012

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0012
2. Read source contract `plans/0013-semver-basic-resolver.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/resolver, internal/registry, internal/manifest
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Simple ^ range resolves to expected highest compatible version
2. Transitive closure matches fixture registry graph
3. Identical inputs produce byte-identical canonical graph
4. Unsatisfiable range returns stable error with package name
5. Cycle detection reports full cycle path

Suggested commands (once code exists):

```powershell
go test ./internal/resolver/... -count=1
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
