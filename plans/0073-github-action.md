# 0073 — Distribution MVP 2 — GitHub Action and CI Integration

## Document Control

| Item | Detail |
|---|---|
| Phase | Distribution / MVP 2 |
| Primary objective | Provide a maintained GitHub Action that installs Mew, restores verified caches, selects Node, and exposes reproducible CI modes. |
| Required predecessors | 0029, 0060, 0072 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Provide a maintained GitHub Action that installs Mew, restores verified caches, selects Node, and exposes reproducible CI modes.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0029 before starting this MVP.
- Complete and merge 0060 before starting this MVP.
- Complete and merge 0072 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub GitHub Action and CI cache guidance

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
uses: mewjs/setup-m@v1
```

## In Scope

- Version/channel selection.
- Checksum verification.
- Node provisioning.
- Store, metadata, transform, and execution cache keys.
- Frozen install and capsule restore helpers.
- Problem matchers and summaries.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Cache keys include format versions and lock semantic hashes.
- Never cache credentials or unsafe mutable project state.
- Action outputs remain stable within a major release.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement action metadata and TypeScript/JavaScript bundle
- [ ] Restore/save store, metadata, transform, and execution caches
- [ ] Test fork PR credential safety (no secret leakage)
- [ ] Pin action release to signed 0072 artifacts

### Core logic

- [ ] Download release artifacts from 0072 with checksum verification
- [ ] Implement frozen install and capsule restore helpers for CI
- [ ] Test cache poisoning and stale cache invalidation
- [ ] Benchmark CI install time cold vs warm cache

### CLI / UX

- [ ] Implement version/channel input resolution
- [ ] Add GitHub job summaries and problem matchers
- [ ] Document example workflows for monorepos and m.lock projects
- [ ] Publish versioning policy for setup-m major bumps

### Tests & fixtures

- [ ] Integrate Node provisioning via m node from 0060
- [ ] Keep action outputs stable within major release
- [ ] Never cache credentials or unsafe mutable project state

### Docs & observability

- [ ] Implement cache key computation with format version salts
- [ ] Test GitHub-hosted Linux/macOS/Windows matrix
- [ ] Expose diagnostics for cache hit/miss reasons

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: setup-m installs verified m on GitHub-hosted runners
- [ ] Acceptance: Cache restore speeds repeat CI runs without correctness loss
- [ ] Acceptance: Fork PRs do not expose repository secrets via action
- [ ] Acceptance: Node version inputs provision correct runtime
- [ ] Acceptance: Action outputs remain stable for v1 consumers
- [ ] Fixture ready: `fixtures/ci/workflows — example GitHub Actions`
- [ ] Fixture ready: `tests/ci/matrix — hosted runner smoke`
- [ ] Fixture ready: `tests/ci/fork-pr — credential safety`
- [ ] Fixture ready: `tests/ci/cache-poison — stale key rejection`
- [ ] Fixture ready: `tests/ci/frozen-install — reproducible CI mode`


Required test layers:

- Unit tests for parsing, normalization, deterministic ordering, and error classification.
- Golden tests for manifests, lockfiles, command output, and migration reports.
- Integration tests against local fixture registries and isolated temporary homes.
- Failure-injection tests for network interruption, disk exhaustion, permission errors, process termination, and corrupted cache entries.
- Cross-platform tests for Linux, macOS, and Windows, including path length, case sensitivity, junctions, symlinks, and executable shims.
- Conformance tests comparing intentional compatibility surfaces with the corresponding Nub or package-manager behavior.

## Performance Requirements

- Add benchmarks for every newly introduced hot path.
- Avoid unbounded goroutines, file descriptors, memory growth, or registry requests.
- Publish baseline measurements in repository benchmark artifacts.

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
| setup-m GitHub Action | Nub GH Action | install + cache + Node | 0073 |
| Cache keys | Nub CI guidance | store/metadata/transform hashes | 0073 |
| Frozen install mode | Nub | reproducible CI installs | 0073 |
| Problem matchers | Nub | CI-friendly diagnostics | 0073 |

## Go Package Map

**Packages / paths:**

- `actions/setup-m/`
- `docs/ci/github-actions.md`

**Forbidden import edges:**

- internal/resolver
- internal/linker

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  action[setup-m Action] --> dl[ReleaseDownload]
  dl --> verify[ChecksumVerify]
  verify --> cache[CacheRestore]
  cache --> node[NodeProvision]
  node --> summary[JobSummary]
```

## Commands and Flags

| Surface | Input | Notes |
|---|---|---|
| `uses: mewjs/setup-m@v1` | `version`, `channel` | Install m/mx |
| | `node-version` | Via 0060 integration |
| | `cache` | Store/metadata/transform keys |

Cache keys include format versions; never cache credentials.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| actions/setup-m action bundle | Compiled JS action release |
| Example workflow templates | CI integration docs |

## Concrete Test Fixtures

- `fixtures/ci/workflows — example GitHub Actions`
- `tests/ci/matrix — hosted runner smoke`
- `tests/ci/fork-pr — credential safety`
- `tests/ci/cache-poison — stale key rejection`
- `tests/ci/frozen-install — reproducible CI mode`

## Acceptance Scenarios

1. setup-m installs verified m on GitHub-hosted runners
2. Cache restore speeds repeat CI runs without correctness loss
3. Fork PRs do not expose repository secrets via action
4. Node version inputs provision correct runtime
5. Action outputs remain stable for v1 consumers

## Nub Conformance Targets

- Nub GitHub Action and CI cache guidance | parity

## Open Decisions

- Whether action publishes post-install job annotations by default
- Cache scope: global vs per-lockfile hash

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
