---
name: "0085 Cross-Cutting — Go Dependency Selection Roadmap"
overview: "Choose, evaluate, pin, and periodically review external Go dependencies for CLI, semver, transformation, filesystems, networking, archives, security, and releases."
todos:
  - id: inventory
    content: "Domains needing deps"
    status: pending
  - id: eval
    content: "Candidate comparisons"
    status: pending
  - id: pin
    content: "go.mod policy"
    status: pending
  - id: license
    content: "License gate"
    status: pending
  - id: vuln
    content: "govulncheck"
    status: pending
  - id: docs
    content: "Roadmap + ADRs"
    status: pending
isProject: false
---

# 0085 - Cross-Cutting — Go Dependency Selection Roadmap

## Source contract

Canonical scope lives in [`plans/0085-dependency-roadmap.md`](../0085-dependency-roadmap.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0003, 0004

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0003, 0004
2. Read source contract `plans/0085-dependency-roadmap.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/dependencies.md, go.mod
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Roadmap covers CLI, semver, transform, FS, net, archive, security, release
2. Every non-stdlib dep has rationale
3. Vulncheck wired

Suggested commands (once code exists):

```powershell
go test ./docs/dependencies.md/... -count=1
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
