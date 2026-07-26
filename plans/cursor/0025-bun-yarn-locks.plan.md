---
name: "0025 Core MVP 16 — Bun and Yarn Lockfile Compatibility"
overview: "Implement Bun lock compatibility and a staged Yarn strategy covering classic read support and certified Berry/PnP read/write behavior."
todos:
  - id: bun
    content: "bun.lock adapter"
    status: pending
  - id: yarn-classic
    content: "yarn.lock v1 adapter"
    status: pending
  - id: berry-nm
    content: "Yarn Berry node-modules adapter"
    status: pending
  - id: berry-pnp
    content: "PnP read path or defer gate"
    status: pending
  - id: preserve
    content: "bun/yarn lock preservation"
    status: pending
  - id: migrate
    content: "Lock migration to m.lock"
    status: pending
  - id: tests
    content: "bun/yarn lock fixtures"
    status: pending
isProject: false
---

# 0025 - Core MVP 16 — Bun and Yarn Lockfile Compatibility

## Source contract

Canonical scope lives in [`plans/0025-bun-yarn-locks.md`](../0025-bun-yarn-locks.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0023, 0024

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0023, 0024
2. Read source contract `plans/0025-bun-yarn-locks.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/lockfile, internal/compat/bun, internal/compat/yarn, internal/resolver
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. bun.lock fixture imports to valid install graph
2. yarn.lock classic project installs with preserved lock
3. Berry node-modules fixture installs without PnP
4. Unsupported Berry feature fails with documented error
5. Identity detection selects correct lock adapter

Suggested commands (once code exists):

```powershell
go test ./internal/lockfile/... -count=1
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
