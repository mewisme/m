---
name: "0044 Runner MVP 5 — `mx` Remote Fetch and Execution"
overview: "Implement secure temporary package execution with local-first behavior, consent, version pinning, execution cache, and offline support."
todos:
  - id: parser
    content: "mx argument parser and dispatch"
    status: pending
  - id: cache
    content: "Execution-cache identity and transactions"
    status: pending
  - id: consent
    content: "TTY and noninteractive consent rules"
    status: pending
  - id: ephemeral
    content: "Ephemeral resolve/link pipeline"
    status: pending
  - id: tests
    content: "Local-hit, consent, and malicious fixtures"
    status: pending
  - id: policy
    content: "Lifecycle trust enforcement"
    status: pending
  - id: docs
    content: "mx security and offline modes"
    status: pending
isProject: false
---

# 0044 - Runner MVP 5 — `mx` Remote Fetch and Execution

## Source contract

Canonical scope lives in [`plans/0044-mx-dlx.md`](../0044-mx-dlx.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0021, 0029, 0043

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0021, 0029, 0043
2. Read source contract `plans/0044-mx-dlx.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: cmd/mx, internal/runner, internal/resolver, internal/store, internal/linker, internal/transaction, internal/policy
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. mx vite@latest runs after explicit consent or --yes in CI
2. Local bin preferred without fetch when available
3. Non-TTY implicit fetch fails without --yes
4. Concurrent mx invocations share safe cache construction
5. Malicious lifecycle scripts blocked by policy

Suggested commands (once code exists):

```powershell
go test ./cmd/mx/... -count=1
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
