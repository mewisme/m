---
name: "0050 Runtime MVP 1 — Node Launch and Compatibility Boundary"
overview: "Launch the user-selected stock Node process from Go with predictable argument handling, preload injection, compatibility escape hatches, and embedded runtime assets."
todos:
  - id: launch
    content: "File-run dispatch and Node selection"
    status: pending
  - id: assets
    content: "Embed, extract, hash runtime assets"
    status: pending
  - id: preload
    content: "CJS/ESM injection hooks"
    status: pending
  - id: optout
    content: "--node zero-augmentation path"
    status: pending
  - id: tests
    content: "Node matrix and opt-out parity"
    status: pending
  - id: docs
    content: "Stock Node augmentation boundary"
    status: pending
isProject: false
---

# 0050 - Runtime MVP 1 — Node Launch and Compatibility Boundary

## Source contract

Canonical scope lives in [`plans/0050-node-launch-compat.md`](../0050-node-launch-compat.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0046

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0046
2. Read source contract `plans/0050-node-launch-compat.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, internal/node, internal/process, runtime/, cmd/m
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m app.js launches stock Node with embedded preloads when augmentation enabled
2. m --node app.js matches plain node behavior within documented tolerance
3. Corrupted runtime assets rejected and re-extracted safely
4. Signals and exit codes propagate correctly
5. No Node source patching or private libnode embedding

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
