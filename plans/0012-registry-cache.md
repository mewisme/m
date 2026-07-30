# 0012 — Core MVP 3 — Registry Client and Metadata Cache

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 3 |
| Primary objective | Fetch npm-compatible package metadata safely, efficiently, and reproducibly with authentication, proxies, retries, and an offline-capable metadata cache. |
| Required predecessors | 0011 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Fetch npm-compatible package metadata safely, efficiently, and reproducibly with authentication, proxies, retries, and an offline-capable metadata cache.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0011 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube registry client and npmrc-shaped registry model
- Nub scoped registry and auth compatibility

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m view react
```
```bash
m cache metadata inspect react
```

## In Scope

- Full and abbreviated packuments.
- Scoped registries and bearer/basic authentication.
- Proxy, NO_PROXY, custom CA, redirects, gzip, conditional requests, and rate limits.
- Positive and negative metadata caching with freshness policy.
- Dist-tags, publish times, deprecations, signatures, provenance pointers, and tarball metadata.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Use a shared HTTP transport with bounded connections.
- Cache raw response plus parsed normalized metadata and schema version.
- Key credentials by registry origin and never include them in cache keys.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement npm registry packument fetch with semver version index
- [ ] Implement bounded concurrent metadata fetch worker pool
- [ ] Normalize dist-tags, versions, and dist.integrity fields
- [ ] Implement cache corruption detection and safe eviction

### Core logic

- [ ] Support scoped registries via .npmrc and project config layering
- [ ] Add exponential backoff retry for transient 5xx and network errors
- [ ] Add integration tests against fixtures/registry local server
- [ ] Support custom registry URL per scope prefix

### CLI / UX

- [ ] Resolve auth tokens per registry URL with redaction in diagnostics
- [ ] Respect --offline: fail closed when cache miss
- [ ] Add failure tests: 404, 401, timeout, truncated body

### Tests & fixtures

- [ ] Implement metadata disk cache keyed by registry URL + package name
- [ ] Support HTTP/SOCKS proxy from config and environment
- [ ] Document registry client interface for resolver package

### Docs & observability

- [ ] Honor ETag and If-None-Match for conditional requests
- [ ] Validate packument JSON schema and reject malformed responses
- [ ] Never log auth headers or token values

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Packument fetch succeeds against local fixture registry
- [ ] Acceptance: ETag cache returns 304 and avoids re-download
- [ ] Acceptance: --offline fails with clear error when metadata absent
- [ ] Acceptance: Auth token never appears in stderr or debug logs
- [ ] Acceptance: Concurrent fetches respect worker pool limit
- [ ] Fixture ready: `fixtures/registry/v1/lodash-4.17.21.tgz`
- [ ] Fixture ready: `fixtures/registry/v1/packuments/lodash.json`
- [ ] Fixture ready: `fixtures/registry/v1/packuments/@scope/pkg.json`
- [ ] Fixture ready: `testdata/registry/cache-hit-miss/`


Required test layers:

- Unit tests for parsing, normalization, deterministic ordering, and error classification.
- Golden tests for manifests, lockfiles, command output, and migration reports.
- Integration tests against local fixture registries and isolated temporary homes.
- Failure-injection tests for network interruption, disk exhaustion, permission errors, process termination, and corrupted cache entries.
- Cross-platform tests for Linux, macOS, and Windows, including path length, case sensitivity, junctions, symlinks, and executable shims.
- Conformance tests comparing intentional compatibility surfaces with the corresponding Nub or package-manager behavior.

## Performance Requirements

- Reuse HTTP/TLS connections and compressed packuments.
- Deduplicate concurrent requests for the same package and registry.
- Bound metadata parse memory for very large package histories.

All performance claims must be backed by reproducible benchmark commands, machine metadata, cold/warm cache separation, and multiple samples. Performance regressions on critical paths require an explicit waiver.

## Security and Trust Requirements

- Validate all external input and fail closed on malformed or ambiguous data.
- Use least-privilege filesystem access and redact credentials in diagnostics.
- Maintain integrity verification before extraction or execution.

Secrets must never be written to logs, lockfiles, snapshots, telemetry, crash reports, or plan files. Archive extraction, script execution, registry authentication, and path construction must be treated as hostile-input boundaries.

## Risks and Mitigations

- Compatibility drift: mitigate with fixture-based conformance tests.
- Cross-platform divergence: mitigate with platform-specific CI and filesystem probes.
- Premature abstraction: require at least two concrete callers before generalizing an interface.

## Deliverables

- [ ] Production implementation and public interfaces.
- [ ] Unit, integration, conformance, and failure-injection tests.
- [ ] User documentation and migration notes where behavior is public.
- [ ] Benchmark baseline and diagnostic instrumentation.

## Exit Criteria

- [ ] All required tests pass on supported operating systems.
- [ ] No unresolved correctness, integrity, or data-loss issue remains.
- [ ] Public behavior and intentional deviations are documented.
- [ ] The next dependent MVP can consume stable interfaces without reaching into internals.







<!-- ENRICHMENT:BEGIN -->

## Feature Inventory Links

Rows this MVP owns or primarily advances (from `0002` inventory themes):

| Feature | Nub baseline | Mew target | Primary MVP |
|---|---|---|---|
| Registry metadata fetch | Nub registry client | npm-compatible packument | 0012 |
| Auth token resolution | Nub/npmrc | Scoped registry auth | 0012, 0006 |
| Metadata disk cache | Nub cache | ETag + conditional GET | 0012 |
| Proxy and retry policy | Nub networking | Bounded workers + backoff | 0012 |

## Go Package Map

**Packages / paths:**

- `internal/registry`
- `internal/config`
- `internal/fetch`

**Forbidden import edges:**

- internal/resolver
- internal/linker

## Data Flow

```mermaid
flowchart LR
  cfg[config] --> reg[internal/registry] --> cache[MetadataCache] --> net[HTTPClient] --> pack[Packument]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m cache dir` | — | Show registry cache location (stub ok) |
| Env | `NPM_CONFIG_REGISTRY`, auth tokens | Redacted in logs |
| Offline | `--offline` | Serve from cache only |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Registry metadata cache | ~/.cache/github.com/mewisme/mew/registry or XDG equivalent |
| Packument normalized model | Input to resolver |

## Concrete Test Fixtures

- `fixtures/registry/v1/lodash-4.17.21.tgz`
- `fixtures/registry/v1/packuments/lodash.json`
- `fixtures/registry/v1/packuments/@scope/pkg.json`
- `testdata/registry/cache-hit-miss/`

## Acceptance Scenarios

1. Packument fetch succeeds against local fixture registry
2. ETag cache returns 304 and avoids re-download
3. --offline fails with clear error when metadata absent
4. Auth token never appears in stderr or debug logs
5. Concurrent fetches respect worker pool limit

## Nub Conformance Targets

- npm registry packument format | parity
- Registry auth resolution | parity
- Metadata caching behavior | parity

## Open Decisions

- Default registry cache TTL vs rely on ETag only
- Support for custom CA bundles in corporate proxies

<!-- ENRICHMENT:END -->

## AI-Agent Handoff Contract

- Read 0000, 0003, 0005, 0007, 0008, and the immediate predecessor before changing code.
- Prefer small vertical pull requests over broad mechanical ports.
- Never copy Rust architecture blindly; preserve behavior and invariants using idiomatic Go.
- Update the feature matrix and conformance inventory when behavior changes.

Before submitting work, an agent must provide:

1. A concise behavior summary and the exact compatibility target.
2. A list of files and public interfaces changed.
3. Commands used for tests, benchmarks, and static analysis.
4. Known gaps, deferred cases, and platform limitations.
5. Evidence that generated files and fixtures are deterministic.
6. A rollback note for any persistent-format or migration change.
