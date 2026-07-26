---
name: "0061 Manager MVP 2 — Package-Manager Meta-Manager"
overview: "Detect, download, cache, pin, invoke, and migrate external package managers as a Corepack replacement and compatibility tool."
todos:
  - id: detect
    content: "PM identity resolver"
    status: pending
  - id: acquire
    content: "Verified PM download/cache"
    status: pending
  - id: invoke
    content: "External PM under selected Node"
    status: pending
  - id: migrate
    content: "Planner, transaction, loss report"
    status: pending
  - id: pin
    content: "Pin/update commands"
    status: pending
  - id: tests
    content: "Migration and rollback corpus"
    status: pending
isProject: false
---

# 0061 - Manager MVP 2 — Package-Manager Meta-Manager

## Source contract

Canonical scope lives in [`plans/0061-pm-manager.md`](../0061-pm-manager.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0023, 0024, 0025, 0060

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0023, 0024, 0025, 0060
2. Read source contract `plans/0061-pm-manager.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/pmmanager, internal/compat, internal/lockfile, cmd/m (pm subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m pm which reports correct manager for incumbent lockfiles
2. Pinned manager version used for invocation
3. Migration produces loss report and rollback snapshot
4. Failed migration restores prior manifest/lock state
5. External PM runs under selected Node from 0060

Suggested commands (once code exists):

```powershell
go test ./internal/pmmanager/... -count=1
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
