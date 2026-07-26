---
name: "0017 Core MVP 8 — Transactional Install and Instant Rollback"
overview: "Make dependency mutations atomic at the product level and introduce install journals, snapshots, history, recovery, and instant rollback."
todos:
  - id: journal
    content: "Mutation journal with inverse ops"
    status: pending
  - id: rollback
    content: "Failure rollback orchestration"
    status: pending
  - id: snapshot
    content: "Point-in-time snapshot store"
    status: pending
  - id: recover
    content: "Interrupted transaction recovery"
    status: pending
  - id: commit
    content: "Atomic commit gate"
    status: pending
  - id: tests
    content: "Failure injection fixtures"
    status: pending
  - id: docs
    content: "Transaction phase diagram"
    status: pending
isProject: false
---

# 0017 - Core MVP 8 — Transactional Install and Instant Rollback

## Source contract

Canonical scope lives in [`plans/0017-transaction-rollback.md`](../0017-transaction-rollback.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0016

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0016
2. Read source contract `plans/0017-transaction-rollback.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/transaction, internal/app, internal/linker, internal/manifest, internal/lockfile
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Failed install leaves prior node_modules intact and usable
2. Interrupted commit can be recovered or cleanly rolled back
3. Snapshot restore returns project to prior dependency state
4. Journal records sufficient ops for full rollback
5. Commit is atomic: no half-updated lockfile visible

Suggested commands (once code exists):

```powershell
go test ./internal/transaction/... -count=1
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
