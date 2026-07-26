---
name: "0001 Program Charter and Product Contract"
overview: "Define Mew as a Go implementation of the Nub product model, with `m` as the primary toolchain binary and `mx` as the executable runner, while explicitly documenting Mew-specific improvements."
todos:
  - id: contracts
    content: "Freeze charter and compatibility axes"
    status: pending
  - id: naming
    content: "Freeze binary/config/env/error naming"
    status: pending
  - id: policy
    content: "Lockfile preservation and experimental gates"
    status: pending
  - id: docs
    content: "Publish charter + ADR template"
    status: pending
  - id: verify
    content: "Map INDEX MVPs to charter objectives"
    status: pending
  - id: migration
    content: "Draft incumbent PM migration narrative"
    status: pending
isProject: false
---

# 0001 - Program Charter and Product Contract

## Source contract

Canonical scope lives in [`plans/0001-program-charter.md`](../0001-program-charter.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

None

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: None
2. Read source contract `plans/0001-program-charter.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/charter.md, docs/compatibility-axes.md, AGENTS.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Charter names m, mx, m.lock, and Nub as behavioral reference without source-port language
2. Compatibility axes table covers CLI, lockfile, config, runtime, and layout
3. Every INDEX MVP maps to at least one charter objective
4. Direct script shortcuts listed as intentional Mew extension
5. ADR process documented before any persistent format is designed

Suggested commands (once code exists):

```powershell
go test ./docs/charter.md/... -count=1
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
