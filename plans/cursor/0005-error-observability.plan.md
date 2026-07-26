---
name: "0005 Stable Error Model, Diagnostics, and Observability"
overview: "Establish stable error codes, structured diagnostics, progress events, tracing, and redaction before implementing networked or destructive operations."
todos:
  - id: codes
    content: "Freeze error code registry"
    status: pending
  - id: types
    content: "Implement typed errors + wrapping"
    status: pending
  - id: reporter
    content: "Human + NDJSON progress"
    status: pending
  - id: redact
    content: "Credential/URL redaction"
    status: pending
  - id: tests
    content: "Golden diagnostics fixtures"
    status: pending
  - id: panic
    content: "Command-boundary panic recovery"
    status: pending
isProject: false
---

# 0005 - Stable Error Model, Diagnostics, and Observability

## Source contract

Canonical scope lives in [`plans/0005-error-observability.md`](../0005-error-observability.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0004

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0004
2. Read source contract `plans/0005-error-observability.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/diagnostics, internal/apperr, internal/trace
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Every public failure path yields a stable code
2. Tokens in registry URLs are redacted in logs
3. JSON reporter validates against schema
4. NDJSON progress events are line-atomic under concurrency

Suggested commands (once code exists):

```powershell
go test ./internal/diagnostics/... -count=1
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
