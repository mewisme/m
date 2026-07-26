---
name: "0057 Runtime Stabilization Gate"
overview: "Certify Mew runtime augmentation across supported Node versions, syntax features, module systems, workers, loaders, watch mode, and debugging."
todos:
  - id: corpus
    content: "Syntax and framework certification"
    status: pending
  - id: nodes
    content: "Node version matrix results"
    status: pending
  - id: soak
    content: "Transform service crash/leak soak"
    status: pending
  - id: security
    content: "IPC and embedded asset review"
    status: pending
  - id: matrix
    content: "Publish runtime support matrix"
    status: pending
  - id: protocols
    content: "Freeze runtime schema versions"
    status: pending
isProject: false
---

# 0057 - Runtime Stabilization Gate

## Source contract

Canonical scope lives in [`plans/0057-runtime-stabilization.md`](../0057-runtime-stabilization.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- Supported syntax and Node versions have published certification results.
- No known transform cache corruption or source-map integrity bug.
- Watch and workers do not leak processes, services, or file descriptors.
- Plain Node escape hatch remains behaviorally plain.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0050, 0051, 0052, 0053, 0054, 0055, 0056

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0050, 0051, 0052, 0053, 0054, 0055, 0056
2. Read source contract `plans/0057-runtime-stabilization.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, internal/transform, tests/conformance/runtime, cmd/m
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Supported syntax and Node versions have published certification
2. No known transform cache corruption or source-map integrity bug
3. Watch and workers pass leak soak without orphaned processes
4. Plain Node escape hatch matches stock node within tolerance
5. Runtime conformance passes on Linux, macOS, Windows

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
