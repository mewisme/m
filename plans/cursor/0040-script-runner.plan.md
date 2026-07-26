---
name: "0040 Runner MVP 1 — Package Script Runner"
overview: "Implement `m run` with npm-compatible environment construction, lifecycle hooks, argument forwarding, signal propagation, and deterministic output."
todos:
  - id: runner
    content: "Implement ScriptRunner and script lookup"
    status: pending
  - id: env
    content: "Build npm-compatible environment and PATH injection"
    status: pending
  - id: supervisor
    content: "Process groups, signals, exit codes"
    status: pending
  - id: reporters
    content: "Human/JSON/NDJSON output modes"
    status: pending
  - id: hooks
    content: "pre/post lifecycle expansion"
    status: pending
  - id: tests
    content: "Shell quoting and signal fixtures"
    status: pending
  - id: docs
    content: "m run forwarding and escape hatch"
    status: pending
isProject: false
---

# 0040 - Runner MVP 1 — Package Script Runner

## Source contract

Canonical scope lives in [`plans/0040-script-runner.md`](../0040-script-runner.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0031

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0031
2. Read source contract `plans/0040-script-runner.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runner, internal/process, internal/manifest, internal/project, internal/lifecycle, cmd/m (run subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m run dev executes script with npm-compatible environment on Linux/macOS/Windows
2. pre/post hooks run in documented order with correct failure propagation
3. Signals forwarded to child; exit code matches child process
4. m run remains explicit path when script name collides with built-in
5. Reporter modes produce deterministic structured output in CI

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
