---
name: "0053 Runtime MVP 4 — Module Resolution, Path Aliases, and Custom Loaders"
overview: "Match Node resolution while adding TypeScript path aliases, extension mapping, custom loader chaining, and package-manager layout awareness."
todos:
  - id: paths
    content: "tsconfig baseUrl/paths matcher"
    status: pending
  - id: hooks
    content: "CJS register and ESM loader chain"
    status: pending
  - id: pnp
    content: "PnP resolution adapter"
    status: pending
  - id: policy
    content: "Extension mapping and conditions"
    status: pending
  - id: trace
    content: "Module resolution diagnostics"
    status: pending
  - id: tests
    content: "Exports corpus and loader composition"
    status: pending
isProject: false
---

# 0053 - Runtime MVP 4 — Module Resolution, Path Aliases, and Custom Loaders

## Source contract

Canonical scope lives in [`plans/0053-module-resolution-loaders.md`](../0053-module-resolution-loaders.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0019, 0025, 0052

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0019, 0025, 0052
2. Read source contract `plans/0053-module-resolution-loaders.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, runtime/, internal/compat, cmd/m
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. tsconfig paths resolve consistently in monorepos
2. Custom loaders run in documented order with user args preserved
3. PnP projects resolve modules through adapter
4. Resolution errors include Node context and Mew guidance
5. Plain Node opt-out bypasses Mew resolution hooks

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
