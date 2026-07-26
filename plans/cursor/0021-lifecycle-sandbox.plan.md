---
name: "0021 Core MVP 12 — Lifecycle Scripts, Trust, and Sandbox Policy"
overview: "Run required package lifecycle scripts under explicit trust policy, capability restrictions, reproducible build caching, and complete audit logs."
todos:
  - id: discover
    content: "Lifecycle script enumeration"
    status: pending
  - id: policy
    content: "Trust and ignore-scripts enforcement"
    status: pending
  - id: sandbox
    content: "Restricted script execution"
    status: pending
  - id: audit
    content: "Lifecycle audit logging"
    status: pending
  - id: cache
    content: "Reproducible build output cache"
    status: pending
  - id: rollback
    content: "Script failure triggers transaction rollback"
    status: pending
  - id: tests
    content: "Lifecycle fixture scripts"
    status: pending
isProject: false
---

# 0021 - Core MVP 12 — Lifecycle Scripts, Trust, and Sandbox Policy

## Source contract

Canonical scope lives in [`plans/0021-lifecycle-sandbox.md`](../0021-lifecycle-sandbox.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0018, 0020

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0018, 0020
2. Read source contract `plans/0021-lifecycle-sandbox.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/lifecycle, internal/policy, internal/app, internal/process
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. postinstall script runs after package materialized
2. Failing lifecycle script triggers full install rollback
3. --ignore-scripts skips all lifecycle execution
4. Untrusted package prompts or blocks per policy
5. Audit log records script executions

Suggested commands (once code exists):

```powershell
go test ./internal/lifecycle/... -count=1
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
