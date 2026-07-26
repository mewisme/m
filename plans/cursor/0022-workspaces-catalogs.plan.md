---
name: "0022 Core MVP 13 — Workspaces, Catalogs, and Filtering"
overview: "Support monorepo discovery, workspace dependency graphs, catalogs, filters, root checks, and atomic multi-importer installation."
todos:
  - id: catalog
    content: "Catalog parse and resolution"
    status: pending
  - id: filter
    content: "--filter pattern expansion"
    status: pending
  - id: recursive
    content: "-r multi-importer install"
    status: pending
  - id: graph
    content: "Workspace dependency ordering"
    status: pending
  - id: lock
    content: "Per-importer m.lock sections"
    status: pending
  - id: validate
    content: "workspace: target checks"
    status: pending
  - id: tests
    content: "Workspace monorepo fixtures"
    status: pending
isProject: false
---

# 0022 - Core MVP 13 — Workspaces, Catalogs, and Filtering

## Source contract

Canonical scope lives in [`plans/0022-workspaces-catalogs.md`](../0022-workspaces-catalogs.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0011, 0020, 0021

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0011, 0020, 0021
2. Read source contract `plans/0022-workspaces-catalogs.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/workspace, internal/manifest, internal/resolver, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m install -r installs all workspace members atomically
2. catalog: deps resolve to catalog-defined versions
3. --filter installs only matching packages and deps
4. Broken workspace: reference fails with clear error
5. m.lock contains importer section per workspace package

Suggested commands (once code exists):

```powershell
go test ./internal/workspace/... -count=1
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
