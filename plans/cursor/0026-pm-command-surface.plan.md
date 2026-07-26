---
name: "0026 Core MVP 17 — Complete Package-Manager Command Surface"
overview: "Complete the package-manager command family with a coherent Mew grammar, documented pnpm-compatible areas, and safe transaction-backed mutations."
todos:
  - id: ci
    content: "m ci clean install"
    status: pending
  - id: outdated
    content: "Version drift reporting"
    status: pending
  - id: dedupe
    content: "Lock deduplication command"
    status: pending
  - id: prune
    content: "Extraneous package removal"
    status: pending
  - id: grammar
    content: "Unified PM flag naming"
    status: pending
  - id: txn
    content: "Transaction wrap all mutations"
    status: pending
  - id: help
    content: "Complete subcommand documentation"
    status: pending
isProject: false
---

# 0026 - Core MVP 17 — Complete Package-Manager Command Surface

## Source contract

Canonical scope lives in [`plans/0026-pm-command-surface.md`](../0026-pm-command-surface.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0021, 0022, 0023, 0024, 0025

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0021, 0022, 0023, 0024, 0025
2. Read source contract `plans/0026-pm-command-surface.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/cli, internal/app, internal/transaction, internal/manifest, internal/resolver
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m ci fails when lockfile out of sync with manifest
2. m outdated reports available updates as JSON
3. m dedupe reduces duplicate packages in lock
4. All mutating commands rollback on failure
5. Help text complete for every PM subcommand

Suggested commands (once code exists):

```powershell
go test ./internal/cli/... -count=1
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
