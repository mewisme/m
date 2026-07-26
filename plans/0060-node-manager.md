# 0060 — Manager MVP 1 — Node Version Manager

## Document Control

| Item | Detail |
|---|---|
| Phase | Managers / MVP 1 |
| Primary objective | Install, verify, select, cache, and automatically provision Node versions for projects and commands. |
| Required predecessors | 0031, 0050 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Install, verify, select, cache, and automatically provision Node versions for projects and commands.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0031 before starting this MVP.
- Complete and merge 0050 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub Node provisioning, checksums, extraction, version files, and shims

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m node install 24
```
```bash
m node use 22
```
```bash
m node list
```
```bash
m node remove 20
```
```bash
m node resolve
```

## In Scope

- Node release index and aliases.
- Platform/architecture artifact selection.
- Checksum verification before extraction.
- `.nvmrc`, `.node-version`, package.json engines, devEngines, and Mew config resolution.
- Per-project pin and automatic provisioning.
- Offline cache and pruning.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Node installations are immutable and content verified.
- Version resolution is separate from PATH shim behavior.
- Corporate proxy and custom CA support reuse registry HTTP foundations.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement Node release metadata client and local cache
- [ ] Extract and atomically publish immutable Node installations
- [ ] Implement m node command family with stable error codes
- [ ] Document separation of version resolution vs PATH shims (0062)

### Core logic

- [ ] Implement version range, alias, and LTS resolution
- [ ] Discover pins from .nvmrc, .node-version, package.json engines/devEngines
- [ ] Implement offline cache usage and installation GC/prune
- [ ] Benchmark install and resolve hot paths

### CLI / UX

- [ ] Select platform/architecture artifacts deterministically
- [ ] Define pin precedence across project and Mew config
- [ ] Add platform artifact fixture server tests
- [ ] Never execute unverified downloaded Node binary

### Tests & fixtures

- [ ] Download artifacts with retries and proxy/CA support
- [ ] Implement per-project pin and automatic provisioning hooks
- [ ] Add checksum and extraction attack corpus tests

### Docs & observability

- [ ] Verify checksums before any extraction
- [ ] Integrate resolved Node selection with 0050 runtime launch
- [ ] Test version precedence and offline install scenarios

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m node install 22 installs verified Node for current platform
- [ ] Acceptance: Tampered artifacts rejected before extraction
- [ ] Acceptance: Project pin resolves consistently across commands
- [ ] Acceptance: Runtime launch uses Node manager selection from 0050
- [ ] Acceptance: Offline install works from warm cache
- [ ] Fixture ready: `fixtures/node/releases — metadata and artifact stubs`
- [ ] Fixture ready: `fixtures/node/checksum-attack — tampered tarball rejection`
- [ ] Fixture ready: `fixtures/node/pin-precedence — engines vs .nvmrc matrix`
- [ ] Fixture ready: `fixtures/node/offline — cache-only install`
- [ ] Fixture ready: `fixtures/node/platforms — arch selection matrix`


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
| Node version manager | Nub Node provisioning | m node install/use | 0060 |
| Version pin discovery | Nub | .nvmrc, .node-version, engines | 0060 |
| Checksum verification | Nub | before extraction | 0060 |
| Offline cache | Nub | prune and offline install | 0060 |

## Go Package Map

**Packages / paths:**

- `internal/node`
- `internal/fetch`
- `internal/archive`
- `internal/store`
- `cmd/m (node subcommand)`

**Forbidden import edges:**

- internal/runtime/augment
- internal/transform

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  pin[PinDiscovery] --> resolve[VersionResolve]
  resolve --> fetch[ArtifactFetch]
  fetch --> verify[ChecksumVerify]
  verify --> install[ImmutableInstall]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m node install 24` | version spec | Download and install Node |
| `m node use 22` | alias/range | Select active version |
| `m node list` | | Installed versions |
| `m node remove 20` | | Uninstall version |
| `m node resolve` | | Show resolved version for project |

Installations immutable and content-verified. Reuse registry HTTP foundations.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Node install store | Immutable versioned Node roots |
| Release metadata cache | nodejs.org index with ETag |

## Concrete Test Fixtures

- `fixtures/node/releases — metadata and artifact stubs`
- `fixtures/node/checksum-attack — tampered tarball rejection`
- `fixtures/node/pin-precedence — engines vs .nvmrc matrix`
- `fixtures/node/offline — cache-only install`
- `fixtures/node/platforms — arch selection matrix`

## Acceptance Scenarios

1. m node install 22 installs verified Node for current platform
2. Tampered artifacts rejected before extraction
3. Project pin resolves consistently across commands
4. Runtime launch uses Node manager selection from 0050
5. Offline install works from warm cache

## Nub Conformance Targets

- Nub Node provisioning and checksums | parity
- Nub version file discovery | parity
- Nub offline cache behavior | parity

## Open Decisions

- Custom Node distribution mirror support in v1
- Default auto-provision on missing pin behavior

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
