---
name: "0070 Product MVP 1 — TypeScript-First Project Initialization"
overview: "Create a fast, opinionated but transparent TypeScript-first project scaffold and a minimal manifest-only mode."
todos:
  - id: templates
    content: "Embedded versioned scaffold assets"
    status: pending
  - id: prompts
    content: "Interactive and flag-driven init"
    status: pending
  - id: generate
    content: "package.json, tsconfig, layout"
    status: pending
  - id: txn
    content: "Install transaction and rollback"
    status: pending
  - id: golden
    content: "Deterministic project snapshots"
    status: pending
  - id: smoke
    content: "Build/run generated projects"
    status: pending
isProject: false
---

# 0070 - Product MVP 1 — TypeScript-First Project Initialization

## Source contract

Canonical scope lives in [`plans/0070-project-init.md`](../0070-project-init.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0011, 0031, 0051

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0011, 0031, 0051
2. Read source contract `plans/0070-project-init.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/app, internal/manifest, internal/transaction, cmd/m (init), templates/init
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m init creates deterministic TS project that runs with m dev
2. Failed init leaves no partial manifest or half-written tree
3. manifest-only mode writes only package.json
4. Nonempty directory policy enforced with clear errors
5. Framework templates directed to mx, not hidden in m init

Suggested commands (once code exists):

```powershell
go test ./internal/app/... -count=1
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
