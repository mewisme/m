---
name: "0042 Runner MVP 3 — Direct `m <script>` Shortcuts"
overview: "Allow exact package.json script names such as `m dev` and `m start` while preserving deterministic built-in command precedence."
todos:
  - id: dispatch
    content: "Two-pass built-in then script resolution"
    status: pending
  - id: args
    content: "Direct forwarding rules without `--`"
    status: pending
  - id: collision
    content: "Reserved-name diagnostics and m run escape"
    status: pending
  - id: suggest
    content: "Typo suggestions without fuzzy execution"
    status: pending
  - id: tests
    content: "Exhaustive collision matrix"
    status: pending
  - id: docs
    content: "Document Mew extension vs Nub"
    status: pending
  - id: inventory
    content: "Mark direct shortcuts as extension"
    status: pending
isProject: false
---

# 0042 - Runner MVP 3 — Direct `m <script>` Shortcuts

## Source contract

Canonical scope lives in [`plans/0042-direct-script-shortcuts.md`](../0042-direct-script-shortcuts.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0010, 0040

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0010, 0040
2. Read source contract `plans/0042-direct-script-shortcuts.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/cli, internal/runner, internal/manifest, cmd/m (dispatch)
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m dev runs dev script when not a built-in command
2. m add runs built-in add; m run add runs add script if present
3. Misspelled commands show suggestions without executing
4. Dispatch precedence matches documented charter order
5. No fuzzy script execution occurs

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
