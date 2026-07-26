---
name: "0083 Cross-Cutting — Nub Rust to Mew Go Migration Map"
overview: "Map each Nub component and behavior to a Mew Go package, a compatibility test, a replacement design, or an intentional omission."
todos:
  - id: inventory
    content: "Nub+Aube crate list"
    status: pending
  - id: map
    content: "Go package targets"
    status: pending
  - id: omit
    content: "Intentional omissions"
    status: pending
  - id: tests
    content: "Attach compat test IDs"
    status: pending
  - id: sync
    content: "Keep with 0003"
    status: pending
  - id: docs
    content: "Migration guide outline"
    status: pending
isProject: false
---

# 0083 - Cross-Cutting — Nub Rust to Mew Go Migration Map

## Source contract

Canonical scope lives in [`plans/0083-rust-go-migration-map.md`](../0083-rust-go-migration-map.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0002, 0003

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0002, 0003
2. Read source contract `plans/0083-rust-go-migration-map.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/migration/nub-to-mew.md, plans/0083-rust-go-migration-map.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Every Nub crate has map/omit row
2. Every mapped row has owner MVP
3. Intentional omissions documented

Suggested commands (once code exists):

```powershell
go test ./docs/migration/nub-to-mew.md/... -count=1
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
