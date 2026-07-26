---
name: "0043 Runner MVP 4 — Local Package Binary Execution"
overview: "Implement network-free local binary execution through `m exec`, workspace-aware `.bin` discovery, and robust platform shims."
todos:
  - id: lookup
    content: "Bin index and workspace discovery"
    status: pending
  - id: pnp
    content: "PnP bin resolution adapter"
    status: pending
  - id: spawn
    content: "Direct spawn and Windows shims"
    status: pending
  - id: policy
    content: "No registry fallback on exec miss"
    status: pending
  - id: tests
    content: "Collision, PnP, and shim fixtures"
    status: pending
  - id: docs
    content: "m exec vs mx boundary"
    status: pending
isProject: false
---

# 0043 - Runner MVP 4 — Local Package Binary Execution

## Source contract

Canonical scope lives in [`plans/0043-local-exec.md`](../0043-local-exec.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0019, 0040

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0019, 0040
2. Read source contract `plans/0043-local-exec.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runner, internal/linker, internal/compat, cmd/m (exec subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m exec eslint runs local bin without network access
2. Ambiguous package bins produce clear selection errors
3. PnP projects resolve bins through adapter
4. Windows shims execute with correct quoting
5. Local miss suggests install; never silently fetches

Suggested commands (once code exists):

```powershell
go test ./internal/runner/... -count=1
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
