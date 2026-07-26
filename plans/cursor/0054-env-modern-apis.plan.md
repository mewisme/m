---
name: "0054 Runtime MVP 5 — Environment Loading, Workers, Storage, and Modern APIs"
overview: "Provide Nub-style environment-file loading and selected browser-compatible APIs without violating plain Node semantics or worker boundaries."
todos:
  - id: parser
    content: ".env parse and expansion"
    status: pending
  - id: discovery
    content: "Mode-aware .env* loading"
    status: pending
  - id: precedence
    content: "Shell/file/flag merge rules"
    status: pending
  - id: workers
    content: "Runtime hook propagation"
    status: pending
  - id: storage
    content: "Web Storage compatibility layer"
    status: pending
  - id: trace
    content: "Redacted env diagnostics"
    status: pending
  - id: tests
    content: "Precedence and worker fixtures"
    status: pending
isProject: false
---

# 0054 - Runtime MVP 5 — Environment Loading, Workers, Storage, and Modern APIs

## Source contract

Canonical scope lives in [`plans/0054-env-modern-apis.md`](../0054-env-modern-apis.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0050, 0053

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0050, 0053
2. Read source contract `plans/0054-env-modern-apis.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/runtime, internal/config, cmd/m
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. --env-file overrides auto-discovery per documented policy
2. Child processes receive explicit env overlays; parent env not raced
3. Workers inherit transform/runtime hooks without recursive services
4. Env trace redacts secrets by default
5. Web Storage APIs behave per documented persistence policy

Suggested commands (once code exists):

```powershell
go test ./internal/runtime/... -count=1
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
