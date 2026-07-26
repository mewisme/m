---
name: "0023 Core MVP 14 — Nub and pnpm Lockfile Bridge"
overview: "Read, preserve, write, validate, diff, and explicitly migrate Nub and supported pnpm lockfile generations through the canonical graph."
todos:
  - id: nub-read
    content: "nub.lock import adapter"
    status: pending
  - id: pnpm-read
    content: "pnpm-lock.yaml import adapter"
    status: pending
  - id: preserve
    content: "Incumbent lock preservation policy"
    status: pending
  - id: migrate
    content: "Explicit lock migration command"
    status: pending
  - id: diff
    content: "Canonical graph diff tooling"
    status: pending
  - id: golden
    content: "Per-generation lock fixtures"
    status: pending
  - id: report
    content: "Lossy migration reporting"
    status: pending
isProject: false
---

# 0023 - Core MVP 14 — Nub and pnpm Lockfile Bridge

## Source contract

Canonical scope lives in [`plans/0023-nub-pnpm-lock-bridge.md`](../0023-nub-pnpm-lock-bridge.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0015, 0020, 0022

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0015, 0020, 0022
2. Read source contract `plans/0023-nub-pnpm-lock-bridge.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/lockfile, internal/compat/nub, internal/compat/pnpm, internal/resolver
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Install on nub.lock project preserves nub.lock format
2. pnpm-lock.yaml project installs without silent m.lock conversion
3. m migrate lock --dry-run lists lossy fields
4. Adapter round-trip nub.lock golden matches source
5. Unsupported lock version returns actionable error

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
