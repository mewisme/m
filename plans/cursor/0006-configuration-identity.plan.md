---
name: "0006 Configuration and Project Identity Model"
overview: "Implement layered configuration and package-manager identity detection without reading branded configuration from an unrelated incumbent manager."
todos:
  - id: layers
    content: "Freeze config precedence"
    status: pending
  - id: identity
    content: "Implement detection order"
    status: pending
  - id: keys
    content: "Publish owned config key list"
    status: pending
  - id: tests
    content: "Identity + merge fixtures"
    status: pending
  - id: docs
    content: "Config reference"
    status: pending
  - id: creds
    content: "Separate credential references from config"
    status: pending
isProject: false
---

# 0006 - Configuration and Project Identity Model

## Source contract

Canonical scope lives in [`plans/0006-configuration-identity.md`](../0006-configuration-identity.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0004, 0005

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0004, 0005
2. Read source contract `plans/0006-configuration-identity.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/config, internal/project
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Detection order matches AGENTS.md
2. Conflicting signals produce explicit errors, not silent picks
3. Env overrides project overrides user as documented
4. pnpm-specific files are not read for Mew-identity projects unless importing

Suggested commands (once code exists):

```powershell
go test ./internal/config/... -count=1
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
