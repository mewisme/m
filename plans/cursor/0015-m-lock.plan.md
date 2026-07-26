---
name: "0015 Core MVP 6 — Native `m.lock` Format"
overview: "Design and implement Mew’s deterministic native lockfile with complete graph, importer, policy, integrity, peer-context, and compatibility metadata."
todos:
  - id: schema
    content: "m.lock versioned document format"
    status: pending
  - id: write
    content: "Graph to lockfile serialization"
    status: pending
  - id: read
    content: "Lockfile parser and validator"
    status: pending
  - id: frozen
    content: "Manifest/lock drift detection"
    status: pending
  - id: golden
    content: "Round-trip encoding fixtures"
    status: pending
  - id: docs
    content: "m.lock field reference"
    status: pending
  - id: fuzz
    content: "Parser robustness smoke"
    status: pending
isProject: false
---

# 0015 - Core MVP 6 — Native `m.lock` Format

## Source contract

Canonical scope lives in [`plans/0015-m-lock.md`](../0015-m-lock.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0007, 0013

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0007, 0013
2. Read source contract `plans/0015-m-lock.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/lockfile, internal/lockfile/mlock, internal/resolver
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Resolver graph round-trips through m.lock losslessly
2. Generated m.lock is byte-identical across platforms for same input
3. Frozen lockfile mode fails when manifest changes
4. Corrupt lockfile returns stable parse error code
5. Schema version field present on every document

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
