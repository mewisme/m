---
name: "0055 Runtime MVP 6 — Dependency-Aware Watch Mode"
overview: "Restart applications safely when relevant source, configuration, environment, or package dependencies change while preserving terminal and signal behavior."
todos:
  - id: watcher
    content: "Platform backends and polling fallback"
    status: pending
  - id: deps
    content: "Collect files from runtime hooks"
    status: pending
  - id: restart
    content: "State machine and signal escalation"
    status: pending
  - id: debounce
    content: "Coalescing and clear-screen policy"
    status: pending
  - id: env
    content: "Reload env/tsconfig on change"
    status: pending
  - id: tests
    content: "Atomic save and leak soak fixtures"
    status: pending
isProject: false
---

# 0055 - Runtime MVP 6 — Dependency-Aware Watch Mode

## Source contract

Canonical scope lives in [`plans/0055-watch-mode.md`](../0055-watch-mode.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0040, 0053, 0054

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0040, 0053, 0054
2. Read source contract `plans/0055-watch-mode.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, internal/process, cmd/m (watch)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m watch restarts app when relevant source or config changes
2. Supervisor survives env/tsconfig reloads
3. Debouncing prevents restart storms on rapid saves
4. No process or file descriptor leaks in soak tests
5. TTY and signal behavior preserved across restarts

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
