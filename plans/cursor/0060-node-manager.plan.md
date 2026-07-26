---
name: "0060 Manager MVP 1 — Node Version Manager"
overview: "Install, verify, select, cache, and automatically provision Node versions for projects and commands."
todos:
  - id: metadata
    content: "Node release index client/cache"
    status: pending
  - id: resolve
    content: "Version range and pin precedence"
    status: pending
  - id: install
    content: "Checksum verify and atomic extract"
    status: pending
  - id: integrate
    content: "Wire 0050 Node selection"
    status: pending
  - id: cli
    content: "m node command family"
    status: pending
  - id: tests
    content: "Checksum attack and offline fixtures"
    status: pending
isProject: false
---

# 0060 - Manager MVP 1 — Node Version Manager

## Source contract

Canonical scope lives in [`plans/0060-node-manager.md`](../0060-node-manager.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0031, 0050

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0031, 0050
2. Read source contract `plans/0060-node-manager.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/node, internal/fetch, internal/archive, internal/store, cmd/m (node subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m node install 22 installs verified Node for current platform
2. Tampered artifacts rejected before extraction
3. Project pin resolves consistently across commands
4. Runtime launch uses Node manager selection from 0050
5. Offline install works from warm cache

Suggested commands (once code exists):

```powershell
go test ./internal/node/... -count=1
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
