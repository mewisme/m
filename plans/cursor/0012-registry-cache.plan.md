---
name: "0012 Core MVP 3 — Registry Client and Metadata Cache"
overview: "Fetch npm-compatible package metadata safely, efficiently, and reproducibly with authentication, proxies, retries, and an offline-capable metadata cache."
todos:
  - id: client
    content: "HTTP registry client with retries"
    status: pending
  - id: cache
    content: "Disk metadata cache with ETag"
    status: pending
  - id: auth
    content: "Scoped token resolution"
    status: pending
  - id: proxy
    content: "HTTP/SOCKS proxy wiring"
    status: pending
  - id: offline
    content: "Offline-only mode enforcement"
    status: pending
  - id: tests
    content: "Local fixture registry integration"
    status: pending
  - id: interface
    content: "Registry client contract for resolver"
    status: pending
isProject: false
---

# 0012 - Core MVP 3 — Registry Client and Metadata Cache

## Source contract

Canonical scope lives in [`plans/0012-registry-cache.md`](../0012-registry-cache.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

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

0011

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0011
2. Read source contract `plans/0012-registry-cache.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: internal/registry, internal/config, internal/fetch
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Packument fetch succeeds against local fixture registry
2. ETag cache returns 304 and avoids re-download
3. --offline fails with clear error when metadata absent
4. Auth token never appears in stderr or debug logs
5. Concurrent fetches respect worker pool limit

Suggested commands (once code exists):

```powershell
go test ./internal/registry/... -count=1
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
