---
name: "0071 Product MVP 2 — External Command Plugin Convention"
overview: "Support discoverable external `m-<verb>` commands without loading untrusted code into the Mew process."
todos:
  - id: protocol
    content: "m-<verb> handshake spec"
    status: pending
  - id: discover
    content: "PATH discovery and precedence"
    status: pending
  - id: spawn
    content: "Subprocess dispatch with minimal env"
    status: pending
  - id: doctor
    content: "Trust and compatibility diagnostics"
    status: pending
  - id: sdk
    content: "Reference plugin examples"
    status: pending
  - id: tests
    content: "Shadowing and malicious output fixtures"
    status: pending
isProject: false
---

# 0071 - Product MVP 2 — External Command Plugin Convention

## Source contract

Canonical scope lives in [`plans/0071-plugins.md`](../0071-plugins.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0010, 0043, 0062

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0010, 0043, 0062
2. Read source contract `plans/0071-plugins.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/cli, internal/runner, cmd/m (plugin subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m hello runs m-hello on PATH when not a built-in
2. Built-in commands always win over plugin names
3. Plugin handshake rejects incompatible protocol versions
4. m plugin doctor reports origin and trust metadata
5. No untrusted code loaded into m process address space

Suggested commands (once code exists):

```powershell
go test ./internal/cli/... -count=1
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
