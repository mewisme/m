---
name: "0041 Runner MVP 2 — Workspace Script Orchestration"
overview: "Run scripts across selected workspace packages with topology, concurrency control, failure policy, and structured output."
todos:
  - id: graph
    content: "Task graph from workspace dependencies"
    status: pending
  - id: sched
    content: "Scheduler with concurrency and failure policies"
    status: pending
  - id: mux
    content: "Per-package output multiplexing"
    status: pending
  - id: events
    content: "Machine-readable task events"
    status: pending
  - id: tests
    content: "DAG, cycle, and stress fixtures"
    status: pending
  - id: docs
    content: "Workspace runner flags and policies"
    status: pending
isProject: false
---

# 0041 - Runner MVP 2 — Workspace Script Orchestration

## Source contract

Canonical scope lives in [`plans/0041-workspace-runner.md`](../0041-workspace-runner.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0022, 0040

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0022, 0040
2. Read source contract `plans/0041-workspace-runner.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runner, internal/workspace, internal/process, cmd/m (-r run, --filter)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m -r run build executes packages in correct topological order
2. Concurrency limit respected; no unbounded goroutine fan-out
3. Workspace cycles diagnosed without deadlock
4. Per-package output remains attributable under parallel execution
5. Failure policies behave deterministically across platforms

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
