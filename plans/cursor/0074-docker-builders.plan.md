---
name: "0074 Distribution MVP 3 — Docker Images and Hosted Builder Integration"
overview: "Provide minimal container images, cache-efficient Docker patterns, and adapters for hosted build systems."
todos:
  - id: dockerfiles
    content: "Slim and full image definitions"
    status: pending
  - id: multiarch
    content: "Build and publish pipeline"
    status: pending
  - id: rootless
    content: "Non-root cache path recipes"
    status: pending
  - id: buildkit
    content: "Cache mount documentation"
    status: pending
  - id: smoke
    content: "Container smoke and vuln scan gates"
    status: pending
  - id: docs
    content: "Hosted builder snippets"
    status: pending
isProject: false
---

# 0074 - Distribution MVP 3 — Docker Images and Hosted Builder Integration

## Source contract

Canonical scope lives in [`plans/0074-docker-builders.md`](../0074-docker-builders.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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
2. Read source contract `plans/0074-docker-builders.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: docker/, docs/ci/docker.md, builders/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. docker run mewjs/m:latest m --version succeeds on amd64 and arm64
2. Images run non-root by default with working cache dirs
3. Vulnerability scan gate passes on release images
4. BuildKit examples demonstrate cache-efficient m install
5. No credentials appear in image layers or history

Suggested commands (once code exists):

```powershell
go test ./docker//... -count=1
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
