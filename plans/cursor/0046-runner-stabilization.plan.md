---
name: "0046 Runner Stabilization Gate"
overview: "Certify script and executable execution for interactive development, CI, workspaces, PnP, and cross-platform signal behavior."
todos:
  - id: corpus
    content: "Real-world script conformance corpus"
    status: pending
  - id: soak
    content: "Process leak and cancellation soak"
    status: pending
  - id: matrix
    content: "Publish compatibility/divergence matrix"
    status: pending
  - id: schema
    content: "Freeze runner event schema"
    status: pending
  - id: mx
    content: "Security and consent certification"
    status: pending
  - id: ci
    content: "Stop-the-line runner conformance gates"
    status: pending
isProject: false
---

# 0046 - Runner Stabilization Gate

## Source contract

Canonical scope lives in [`plans/0046-runner-stabilization.md`](../0046-runner-stabilization.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- No known signal, exit-code, stdin, or output corruption bug.
- Direct script shortcut collision behavior is fully tested.
- `mx` never fetches implicitly in non-TTY without explicit consent.
- Workspace scheduler is deterministic and resource bounded.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0040, 0041, 0042, 0043, 0044, 0045

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0040, 0041, 0042, 0043, 0044, 0045
2. Read source contract `plans/0046-runner-stabilization.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runner, internal/process, cmd/m, cmd/mx, tests/conformance/runner
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. No known signal, exit-code, stdin, or output corruption bug
2. Direct script shortcut collisions fully tested and documented
3. mx never fetches implicitly in non-TTY without explicit consent
4. Workspace scheduler deterministic and resource bounded
5. Runner conformance suite passes on Linux, macOS, Windows

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
