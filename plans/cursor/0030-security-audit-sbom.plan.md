---
name: "0030 Core MVP 21 — Audit, SBOM, Provenance, and Supply-Chain Policy"
overview: "Provide comprehensive dependency risk analysis, signed provenance verification, SBOM export, age policies, and enforceable organizational rules."
todos:
  - id: audit
    content: "Vulnerability scanning command"
    status: pending
  - id: sbom
    content: "CycloneDX/SPDX export"
    status: pending
  - id: provenance
    content: "Attestation verification"
    status: pending
  - id: policy
    content: "Org rule enforcement"
    status: pending
  - id: offline
    content: "Cached advisory DB"
    status: pending
  - id: integrate
    content: "Policy in install validate phase"
    status: pending
  - id: tests
    content: "Audit and SBOM fixtures"
    status: pending
isProject: false
---

# 0030 - Core MVP 21 — Audit, SBOM, Provenance, and Supply-Chain Policy

## Source contract

Canonical scope lives in [`plans/0030-security-audit-sbom.md`](../0030-security-audit-sbom.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0012, 0021, 0027, 0029

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0012, 0021, 0027, 0029
2. Read source contract `plans/0030-security-audit-sbom.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/policy, internal/registry, internal/resolver, internal/diagnostics, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m audit reports known CVE on fixture vulnerable package
2. m sbom output validates against CycloneDX schema
3. Policy deny blocks install of blocked package
4. Provenance verify passes on signed fixture package
5. Audit works offline with cached advisory DB

Suggested commands (once code exists):

```powershell
go test ./internal/policy/... -count=1
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
