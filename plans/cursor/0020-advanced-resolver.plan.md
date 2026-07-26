---
name: "0020 Core MVP 11 — Full Dependency Resolver"
overview: "Complete resolver semantics for peer dependencies, optional dependencies, overrides, aliases, platforms, workspace protocols, and deterministic incremental updates."
todos:
  - id: peers
    content: "Peer context resolution"
    status: pending
  - id: optional
    content: "Platform and optional pruning"
    status: pending
  - id: overrides
    content: "Override and alias rewriting"
    status: pending
  - id: workspace
    content: "workspace:* protocol"
    status: pending
  - id: incremental
    content: "Targeted lock reuse"
    status: pending
  - id: explain
    content: "Conflict explanation tree"
    status: pending
  - id: tests
    content: "Advanced resolver conformance fixtures"
    status: pending
isProject: false
---

# 0020 - Core MVP 11 — Full Dependency Resolver

## Source contract

Canonical scope lives in [`plans/0020-advanced-resolver.md`](../0020-advanced-resolver.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0019

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0019
2. Read source contract `plans/0020-advanced-resolver.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/resolver, internal/manifest, internal/workspace, internal/lockfile
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Conflicting peer deps produce explanation tree not silent wrong version
2. workspace:* resolves to correct local package version
3. Optional dep skipped on unsupported platform
4. Override replaces transitive version deterministically
5. Targeted update preserves unrelated lock subgraph

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
