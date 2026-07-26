---
name: "0051 Runtime MVP 2 — Go Transform Service and TypeScript Execution"
overview: "Execute TypeScript through stock Node using a Go-native transform pipeline and a small embedded Node loader bridge."
todos:
  - id: benchmark
    content: "Evaluate Go transformers vs OXC corpus"
    status: pending
  - id: protocol
    content: "Freeze transform IPC schema"
    status: pending
  - id: service
    content: "Lifecycle, auth, crash recovery"
    status: pending
  - id: loader
    content: "Node loader bridge for TS formats"
    status: pending
  - id: tsconfig
    content: "Discovery and normalized options"
    status: pending
  - id: cache
    content: "Content-addressed transpile store"
    status: pending
  - id: tests
    content: "IPC failure and syntax corpus"
    status: pending
isProject: false
---

# 0051 - Runtime MVP 2 — Go Transform Service and TypeScript Execution

## Source contract

Canonical scope lives in [`plans/0051-go-transform-service.md`](../0051-go-transform-service.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0050

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0050
2. Read source contract `plans/0051-go-transform-service.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/transform, internal/runtime, runtime/loader-bridge, cmd/m
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m app.ts executes TypeScript through stock Node without separate tsc step
2. Transform service recovers from crash without wedging Node loader
3. Transpile cache hits produce identical output bytes
4. Diagnostics reference original TypeScript sources
5. Unsupported options fail with actionable messages

Suggested commands (once code exists):

```powershell
go test ./internal/transform/... -count=1
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
