---
name: "0056 Runtime MVP 7 — Debugging, Inspection, and Runtime Diagnostics"
overview: "Integrate Node inspector, source maps, transform traces, module traces, and support bundles for production-quality debugging."
todos:
  - id: inspect
    content: "Flag routing to stock Node"
    status: pending
  - id: schema
    content: "Runtime trace event format"
    status: pending
  - id: maps
    content: "Source-map debugger integration"
    status: pending
  - id: bundle
    content: "Redacted support archive"
    status: pending
  - id: cache
    content: "Transpile cache explain command"
    status: pending
  - id: doctor
    content: "Runtime health diagnostics"
    status: pending
  - id: tests
    content: "Inspector and breakpoint fixtures"
    status: pending
isProject: false
---

# 0056 - Runtime MVP 7 — Debugging, Inspection, and Runtime Diagnostics

## Source contract

Canonical scope lives in [`plans/0056-debugging-inspection.md`](../0056-debugging-inspection.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0052, 0053, 0055

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0052, 0053, 0055
2. Read source contract `plans/0056-debugging-inspection.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, internal/transform, cmd/m (doctor, trace)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m --inspect-brk app.ts breaks on first line with mapped sources
2. Stack traces map through transforms to original TypeScript
3. Support bundles contain no secrets or full source by default
4. Trace output validates against published schema
5. Diagnostics do not change execution order materially

Suggested commands (once code exists):

```powershell
go test ./internal/runtime/... -count=1
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
