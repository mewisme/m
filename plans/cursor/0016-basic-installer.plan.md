---
name: "0016 Core MVP 7 — Basic End-to-End Installer"
overview: "Deliver the first usable `m install`, `m add`, and `m remove` path using `m.lock`, a project-local staging area, and a conservative hoisted layout."
todos:
  - id: install
    content: "End-to-end install orchestration"
    status: pending
  - id: hoist
    content: "Conservative hoisted linker"
    status: pending
  - id: add-remove
    content: "Manifest and lock mutations"
    status: pending
  - id: bins
    content: ".bin shim generation"
    status: pending
  - id: staging
    content: "Stage-then-publish layout"
    status: pending
  - id: dry-run
    content: "Plan-only install mode"
    status: pending
  - id: tests
    content: "Install integration fixtures"
    status: pending
isProject: false
---

# 0016 - Core MVP 7 — Basic End-to-End Installer

## Source contract

Canonical scope lives in [`plans/0016-basic-installer.md`](../0016-basic-installer.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0011, 0013, 0014, 0015

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0011, 0013, 0014, 0015
2. Read source contract `plans/0016-basic-installer.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/app, internal/resolver, internal/fetch, internal/lockfile, internal/linker, internal/manifest
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. m install on greenfield project produces working node_modules
2. m add lodash updates m.lock and node_modules
3. m remove prunes unused packages from hoisted tree
4. Failed install does not leave corrupt partial node_modules
5. m install --frozen-lockfile fails when package.json changed

Suggested commands (once code exists):

```powershell
go test ./internal/app/... -count=1
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
