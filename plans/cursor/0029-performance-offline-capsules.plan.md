---
name: "0029 Core MVP 20 — Performance, Offline Operation, and Portable Capsules"
overview: "Optimize cold and warm installs, make offline behavior first-class, and package reproducible dependency environments for CI and containers."
todos:
  - id: profile
    content: "Install phase profiling"
    status: pending
  - id: optimize
    content: "Hot path optimizations"
    status: pending
  - id: offline
    content: "Offline completeness preflight"
    status: pending
  - id: capsule
    content: "Create/restore dep capsules"
    status: pending
  - id: bench
    content: "Install benchmark harness"
    status: pending
  - id: ci-gate
    content: "Performance regression gate"
    status: pending
  - id: docs
    content: "Offline and tuning guide"
    status: pending
isProject: false
---

# 0029 - Core MVP 20 — Performance, Offline Operation, and Portable Capsules

## Source contract

Canonical scope lives in [`plans/0029-performance-offline-capsules.md`](../0029-performance-offline-capsules.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0018, 0026, 0028

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0018, 0026, 0028
2. Read source contract `plans/0029-performance-offline-capsules.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/fetch, internal/store, internal/registry, internal/archive, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Warm install measurably faster than cold on benchmark fixture
2. Offline install succeeds when capsule/cache complete
3. Capsule round-trip produces identical node_modules hash
4. Benchmark CI gate fails on >10% regression without waiver
5. Phase timing diagnostics available via --debug

Suggested commands (once code exists):

```powershell
go test ./internal/fetch/... -count=1
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
