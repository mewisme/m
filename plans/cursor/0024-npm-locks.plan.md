---
name: "0024 Core MVP 15 — npm Lockfile and Shrinkwrap Compatibility"
overview: "Support modern package-lock and npm-shrinkwrap formats while preserving npm project identity and install semantics."
todos:
  - id: parse
    content: "package-lock v2/v3 parser"
    status: pending
  - id: write
    content: "npm lock writer"
    status: pending
  - id: identity
    content: "npm project lock preservation"
    status: pending
  - id: shrinkwrap
    content: "npm-shrinkwrap support"
    status: pending
  - id: workspaces
    content: "npm lock v3 workspaces"
    status: pending
  - id: migrate
    content: "npm lock to m.lock migration"
    status: pending
  - id: tests
    content: "npm lock golden fixtures"
    status: pending
isProject: false
---

# 0024 - Core MVP 15 — npm Lockfile and Shrinkwrap Compatibility

## Source contract

Canonical scope lives in [`plans/0024-npm-locks.md`](../0024-npm-locks.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0023

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0023
2. Read source contract `plans/0024-npm-locks.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/lockfile, internal/compat/npm, internal/resolver
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. npm fixture install matches package-lock dependency tree
2. package-lock.json preserved after m install on npm project
3. Frozen install fails when package.json conflicts with lock
4. npm-shrinkwrap project installs correctly
5. Lock v2 and v3 fixtures parse without error

Suggested commands (once code exists):

```powershell
go test ./internal/lockfile/... -count=1
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
