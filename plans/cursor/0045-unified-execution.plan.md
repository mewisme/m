---
name: "0045 Runner MVP 6 — Unified Execution and Snapshot Environments"
overview: "Unify `m exec`, `mx`, historical snapshots, and capsules behind one environment builder and executable resolver."
todos:
  - id: model
    content: "ExecutionRequest and PreparedEnvironment types"
    status: pending
  - id: refactor
    content: "Unify exec and mx onto shared services"
    status: pending
  - id: providers
    content: "Snapshot and capsule environment sources"
    status: pending
  - id: inspect
    content: "Environment provenance command"
    status: pending
  - id: cleanup
    content: "Ephemeral root leases"
    status: pending
  - id: tests
    content: "Equivalence and isolation fixtures"
    status: pending
isProject: false
---

# 0045 - Runner MVP 6 — Unified Execution and Snapshot Environments

## Source contract

Canonical scope lives in [`plans/0045-unified-execution.md`](../0045-unified-execution.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0028, 0029, 0043, 0044

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0028, 0029, 0043, 0044
2. Read source contract `plans/0045-unified-execution.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runner, internal/lockfile, internal/transaction, cmd/m, cmd/mx
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m exec and mx produce equivalent supervision through shared layer
2. Snapshot and capsule providers verify integrity before execution
3. Incompatible graphs never merge without explicit user action
4. Environment inspect shows identity, provenance, and cache state
5. Ephemeral roots cleaned up on success and failure

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
