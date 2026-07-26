---
name: "0010 Core MVP 1 — CLI Foundation and Command Dispatch"
overview: "Ship a stable command shell for `m` and `mx`, global flags, help, version output, command dispatch, and reserved-name policy."
todos:
  - id: entry
    content: "cmd/m and cmd/mx bootstrap"
    status: pending
  - id: cli
    content: "Cobra root command and global flags"
    status: pending
  - id: dispatch
    content: "Reserved-name policy and precedence"
    status: pending
  - id: exit
    content: "Error-code to exit-code mapping"
    status: pending
  - id: completion
    content: "Shell completion generation"
    status: pending
  - id: tests
    content: "Help/version golden fixtures"
    status: pending
  - id: docs
    content: "CLI foundation user surface"
    status: pending
isProject: false
---

# 0010 - Core MVP 1 — CLI Foundation and Command Dispatch

## Source contract

Canonical scope lives in [`plans/0010-cli-foundation.md`](../0010-cli-foundation.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0004, 0005, 0006, 0007

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0004, 0005, 0006, 0007
2. Read source contract `plans/0010-cli-foundation.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: cmd/m, cmd/mx, internal/app, internal/cli, internal/config, internal/diagnostics
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m --help and mx --help render stable usage without panic
2. m version prints name, semver, commit, and build date
3. Global --cwd changes effective project root for downstream services
4. SIGINT returns non-zero exit and cancels in-flight context
5. Reserved built-in names cannot be shadowed by future script shortcuts

Suggested commands (once code exists):

```powershell
go test ./cmd/m/... -count=1
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
