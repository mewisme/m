---
name: "0073 Distribution MVP 2 — GitHub Action and CI Integration"
overview: "Provide a maintained GitHub Action that installs Mew, restores verified caches, selects Node, and exposes reproducible CI modes."
todos:
  - id: action
    content: "setup-m bundle and inputs"
    status: pending
  - id: verify
    content: "Release download checksum gate"
    status: pending
  - id: cache
    content: "Key computation and restore/save"
    status: pending
  - id: node
    content: "Node provisioning integration"
    status: pending
  - id: ci
    content: "Hosted runner matrix tests"
    status: pending
  - id: docs
    content: "Example workflow templates"
    status: pending
isProject: false
---

# 0073 - Distribution MVP 2 — GitHub Action and CI Integration

## Source contract

Canonical scope lives in [`plans/0073-github-action.md`](../0073-github-action.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0029, 0060, 0072

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0029, 0060, 0072
2. Read source contract `plans/0073-github-action.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: actions/setup-m/, docs/ci/github-actions.md
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. setup-m installs verified m on GitHub-hosted runners
2. Cache restore speeds repeat CI runs without correctness loss
3. Fork PRs do not expose repository secrets via action
4. Node version inputs provision correct runtime
5. Action outputs remain stable for v1 consumers

Suggested commands (once code exists):

```powershell
go test ./actions/setup-m//... -count=1
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
