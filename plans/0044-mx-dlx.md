# 0044 — Runner MVP 5 — `mx` Remote Fetch and Execution

## Document Control

| Item | Detail |
|---|---|
| Phase | Runner / MVP 5 |
| Primary objective | Implement secure temporary package execution with local-first behavior, consent, version pinning, execution cache, and offline support. |
| Required predecessors | 0021, 0029, 0043 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement secure temporary package execution with local-first behavior, consent, version pinning, execution cache, and offline support.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0021 before starting this MVP.
- Complete and merge 0029 before starting this MVP.
- Complete and merge 0043 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nubx local-first DLX fallback, consent flow, package flags, and cache

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
mx create-vite
```
```bash
mx vite@latest
```
```bash
mx -p typescript tsc --version
```
```bash
mx --offline prettier .
```

## In Scope

- Package spec parsing and bin inference.
- Local binary preference unless explicit packages force remote context.
- TTY consent on first implicit fetch; fail closed in non-TTY without `--yes`.
- Versioned execution environments and cache retention.
- Multiple packages and shell mode.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Reuse resolver, store, linker, lifecycle policy, and process supervisor.
- Execution environments are isolated from the project unless explicitly requested.
- Consent record keys include normalized package spec and trust-relevant policy.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement mx argument parser and top-level dispatch
- [ ] Implement versioned execution-cache identity and atomic transaction
- [ ] Implement cache retention, cleanup, and prune commands
- [ ] Document mx vs m exec security boundary

### Core logic

- [ ] Implement local-first bin lookup reusing 0043 resolver
- [ ] Implement TTY consent on first implicit fetch
- [ ] Add local-hit/no-network integration tests
- [ ] Benchmark cold vs warm mx execution cache hits

### CLI / UX

- [ ] Implement package spec parsing and bin inference
- [ ] Fail closed in non-TTY without explicit --yes
- [ ] Add consent and non-TTY matrix tests
- [ ] Ensure execution environments isolated from project unless requested

### Tests & fixtures

- [ ] Build ephemeral importer and minimal lock graph for remote packages
- [ ] Support multiple packages and shell mode execution
- [ ] Test concurrent same-spec cache construction

### Docs & observability

- [ ] Reuse resolver, store, linker, lifecycle policy, and supervisor
- [ ] Implement bin ambiguity errors with actionable diagnostics
- [ ] Add malicious lifecycle package fixtures with policy enforcement

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: mx vite@latest runs after explicit consent or --yes in CI
- [ ] Acceptance: Local bin preferred without fetch when available
- [ ] Acceptance: Non-TTY implicit fetch fails without --yes
- [ ] Acceptance: Concurrent mx invocations share safe cache construction
- [ ] Acceptance: Malicious lifecycle scripts blocked by policy
- [ ] Fixture ready: `fixtures/mx/local-hit — no network when bin present`
- [ ] Fixture ready: `fixtures/mx/consent — TTY and --yes matrices`
- [ ] Fixture ready: `fixtures/mx/concurrent-cache — parallel same-spec builds`
- [ ] Fixture ready: `fixtures/mx/malicious-lifecycle — policy enforcement`
- [ ] Fixture ready: `fixtures/mx/offline — cache-only execution`


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
| mx remote execution | Nubx DLX | Temporary package execution | 0044 |
| Local-first lookup | Nubx | Prefer local bin unless -p forces remote | 0044 |
| Consent flow | Nubx | TTY consent; --yes for CI | 0044 |
| Execution cache | Nubx | Versioned ephemeral environments | 0044 |

## Go Package Map

**Packages / paths:**

- `cmd/mx`
- `internal/runner`
- `internal/resolver`
- `internal/store`
- `internal/linker`
- `internal/transaction`
- `internal/policy`

**Forbidden import edges:**

- internal/runtime
- internal/transform

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  mx[mx argv] --> local[LocalFirstLookup]
  local -->|miss| consent[ConsentGate]
  consent --> resolve[Resolver]
  resolve --> stage[EphemeralLink]
  stage --> exec[ProcessSupervisor]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `mx create-vite` | package spec inference | Bin name from package |
| `mx vite@latest` | version pin | Explicit version |
| `mx -p typescript tsc --version` | `-p` / `--package` | Force package context |
| `mx --offline prettier .` | `--offline` | Cache-only execution |

Non-TTY without --yes fails closed on implicit fetch.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Execution cache directory | Content-addressed ephemeral roots |
| Consent record store | Trust policy keyed by normalized spec |

## Concrete Test Fixtures

- `fixtures/mx/local-hit — no network when bin present`
- `fixtures/mx/consent — TTY and --yes matrices`
- `fixtures/mx/concurrent-cache — parallel same-spec builds`
- `fixtures/mx/malicious-lifecycle — policy enforcement`
- `fixtures/mx/offline — cache-only execution`

## Acceptance Scenarios

1. mx vite@latest runs after explicit consent or --yes in CI
2. Local bin preferred without fetch when available
3. Non-TTY implicit fetch fails without --yes
4. Concurrent mx invocations share safe cache construction
5. Malicious lifecycle scripts blocked by policy

## Nub Conformance Targets

- Nubx local-first DLX | parity
- Nubx consent and cache behavior | parity
- Nubx package flags | parity

## Open Decisions

- Default execution-cache retention TTL
- Whether mx shares consent store with m pm operations

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
