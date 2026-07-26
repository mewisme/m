---
name: "0028 Core MVP 19 — Explainability, Plans, Semantic Diffs, and Time Travel"
overview: "Expose every resolver and installer decision, preview all mutations, compare dependency graphs semantically, and run or restore historical snapshots."
todos:
  - id: explain
    content: "Resolver decision explanation"
    status: pending
  - id: plan
    content: "Install mutation preview"
    status: pending
  - id: diff
    content: "Semantic lock graph diff"
    status: pending
  - id: snapshot
    content: "History restore UX"
    status: pending
  - id: json
    content: "Machine-readable trace output"
    status: pending
  - id: perf
    content: "Large graph explain benchmark"
    status: pending
  - id: tests
    content: "Explain/plan/diff golden fixtures"
    status: pending
isProject: false
---

# 0028 - Core MVP 19 — Explainability, Plans, Semantic Diffs, and Time Travel

## Source contract

Canonical scope lives in [`plans/0028-explain-plan-history.md`](../0028-explain-plan-history.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0017, 0020, 0026

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0017, 0020, 0026
2. Read source contract `plans/0028-explain-plan-history.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/resolver, internal/transaction, internal/diagnostics, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m explain prints version selection path for target package
2. m plan --json matches actual install file changes on dry-run
3. m diff lock detects semver bump between two locks
4. m snapshot restore returns project to recorded state
5. Explain/plan/diff never modify project files

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
