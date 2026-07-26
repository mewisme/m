---
name: "0004 Repository Bootstrap, Tooling, and Engineering Standards"
overview: "Create a reproducible Go repository with strict quality gates, cross-platform builds, fixture management, and agent-friendly contributor guidance."
todos:
  - id: module
    content: "Init go.mod and package skeleton"
    status: pending
  - id: gates
    content: "Wire test vet lint race vuln"
    status: pending
  - id: ci
    content: "Add OS/arch matrix"
    status: pending
  - id: testkit
    content: "Temp home + fixture helpers"
    status: pending
  - id: docs
    content: "AGENTS.md + CONTRIBUTING"
    status: pending
  - id: doctor
    content: "Stub m development doctor contract"
    status: pending
isProject: false
---

# 0004 - Repository Bootstrap, Tooling, and Engineering Standards

## Source contract

Canonical scope lives in [`plans/0004-repository-bootstrap.md`](../0004-repository-bootstrap.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0003

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0003
2. Read source contract `plans/0004-repository-bootstrap.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: go.mod, cmd/m, cmd/mx, internal/testkit, Makefile, .github/workflows, AGENTS.md, tools/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Fresh clone: go test ./... passes on Linux/macOS/Windows CI
2. Lint and vet wired in CI
3. AGENTS.md present and linked from README
4. cmd/m and cmd/mx build

Suggested commands (once code exists):

```powershell
go test ./go.mod/... -count=1
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
