---
name: "0027 Core MVP 18 — Advanced Sources, Patches, Pack, and Publish"
overview: "Support non-registry package sources, package patches, deterministic packing, and authenticated publication with provenance hooks."
todos:
  - id: sources
    content: "git/file/tarball fetchers"
    status: pending
  - id: patch
    content: "Patch commit and apply workflow"
    status: pending
  - id: pack
    content: "Deterministic m pack"
    status: pending
  - id: publish
    content: "Registry publish with auth"
    status: pending
  - id: lock
    content: "Non-registry entries in m.lock"
    status: pending
  - id: policy
    content: "Git fetch sandbox"
    status: pending
  - id: tests
    content: "Source and patch fixtures"
    status: pending
isProject: false
---

# 0027 - Core MVP 18 — Advanced Sources, Patches, Pack, and Publish

## Source contract

Canonical scope lives in [`plans/0027-advanced-sources-publish.md`](../0027-advanced-sources-publish.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0026

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0026
2. Read source contract `plans/0027-advanced-sources-publish.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/fetch, internal/manifest, internal/registry, internal/archive, internal/app
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Git dependency installs at pinned commit
2. Applied patch changes installed file content deterministically
3. m pack tarball matches npm pack file list
4. m publish --dry-run validates without uploading
5. file: dependency resolves relative to manifest

Suggested commands (once code exists):

```powershell
go test ./internal/fetch/... -count=1
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
