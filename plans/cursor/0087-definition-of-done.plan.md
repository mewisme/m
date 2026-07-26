---
name: "0087 Cross-Cutting — Global Definition of Done"
overview: "Define the non-negotiable completion standard applied to every MVP and the final program."
todos:
  - id: checklist
    content: "Review checklist"
    status: pending
  - id: evidence
    content: "Evidence index"
    status: pending
  - id: waiver
    content: "Expiring waivers"
    status: pending
  - id: signoff
    content: "Owner sign-off rules"
    status: pending
  - id: ci
    content: "Expired waiver fail"
    status: pending
  - id: docs
    content: "Publish DoD"
    status: pending
isProject: false
---

# 0087 - Cross-Cutting — Global Definition of Done

## Source contract

Canonical scope lives in [`plans/0087-definition-of-done.md`](../0087-definition-of-done.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- All planned feature inventory rows are shipped, intentionally omitted, or moved to an approved future backlog.
- All supported compatibility targets pass certification.
- All public formats have tested upgrade, recovery, and rollback paths.
- No open critical security or data-integrity issue.
- Release and installation channels are reproducible and verified.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0009, 0080, 0081, 0082, 0084

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0009, 0080, 0081, 0082, 0084
2. Read source contract `plans/0087-definition-of-done.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docs/definition-of-done.md, docs/waivers/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. DoD checklist exists and is used
2. Waivers expire automatically in policy
3. Format/security changes need owner sign-off

Suggested commands (once code exists):

```powershell
go test ./docs/definition-of-done.md/... -count=1
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
