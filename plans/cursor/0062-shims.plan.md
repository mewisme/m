---
name: "0062 Manager MVP 3 — Node, PM, and Self Shims"
overview: "Install safe cross-platform shims that resolve pinned Node, Mew, and external package-manager versions without unexpectedly augmenting plain Node calls."
todos:
  - id: protocol
    content: "Shim launcher design"
    status: pending
  - id: install
    content: "POSIX and Windows shim paths"
    status: pending
  - id: resolve
    content: "Node/PM/Mew version routing"
    status: pending
  - id: guards
    content: "Recursion prevention and bypass"
    status: pending
  - id: repair
    content: "Status, backup, uninstall flows"
    status: pending
  - id: tests
    content: "PATH and Windows shim matrix"
    status: pending
isProject: false
---

# 0062 - Manager MVP 3 — Node, PM, and Self Shims

## Source contract

Canonical scope lives in [`plans/0062-shims.md`](../0062-shims.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0010, 0060, 0061

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0010, 0060, 0061
2. Read source contract `plans/0062-shims.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/node, internal/pmmanager, cmd/m (shim subcommand)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. node shim runs pinned stock Node without Mew augmentation
2. m/mx shims resolve project-pinned Mew version when configured
3. Recursion guards prevent shim loops
4. m shim remove restores prior PATH state from backup
5. Windows shims handle extensions and quoting correctly

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
