---
name: "0011 Core MVP 2 — Manifest Parsing and Project Discovery"
overview: "Reliably discover projects and workspaces, read and update `package.json` without destructive reformatting, and expose normalized dependency declarations."
todos:
  - id: discover
    content: "Walk-up project root detection"
    status: pending
  - id: parse
    content: "Lossless package.json read"
    status: pending
  - id: write
    content: "Safe manifest field updates"
    status: pending
  - id: workspace
    content: "Glob expansion and member index"
    status: pending
  - id: normalize
    content: "Dependency declaration model"
    status: pending
  - id: validate
    content: "Name/version/bin validation"
    status: pending
  - id: tests
    content: "Manifest golden fixtures"
    status: pending
isProject: false
---

# 0011 - Core MVP 2 — Manifest Parsing and Project Discovery

## Source contract

Canonical scope lives in [`plans/0011-manifest-project-discovery.md`](../0011-manifest-project-discovery.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0010

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0010
2. Read source contract `plans/0011-manifest-project-discovery.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/manifest, internal/project, internal/workspace, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. package.json round-trips without unintended whitespace or key loss
2. Workspace globs resolve to stable sorted member list
3. Project discovery stops at first valid root from cwd
4. Invalid manifest fields produce stable machine-readable error codes
5. Normalized dependency map matches npm semantics for scoped packages

Suggested commands (once code exists):

```powershell
go test ./internal/manifest/... -count=1
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
