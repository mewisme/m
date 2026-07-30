# 0085 — Cross-Cutting — Go Dependency Selection Roadmap

## Document Control

| Item | Detail |
|---|---|
| Phase | Cross-Cutting |
| Primary objective | Choose, evaluate, pin, and periodically review external Go dependencies for CLI, semver, transformation, filesystems, networking, archives, security, and releases. |
| Required predecessors | 0003, 0004 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Choose, evaluate, pin, and periodically review external Go dependencies for CLI, semver, transformation, filesystems, networking, archives, security, and releases.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0003 before starting this MVP.
- Complete and merge 0004 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub dependency choices as behavioral clues rather than direct Go equivalents

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Cobra or alternative CLI framework.
- npm-compatible semver implementation.
- JSONC and order-preserving manifest edits.
- YAML/TOML parsers.
- esbuild or alternative Go transform engine.
- Filesystem clone/reflink helpers.
- Keyring and credential storage.
- SBOM, provenance, signatures, and archive libraries.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Prefer standard library when correctness and maintenance are comparable.
- Every dependency needs ownership, license, maintenance, security, and replacement notes.
- Avoid dependencies that require C toolchains unless a measured capability justifies them.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Prefer stdlib everywhere feasible
- [ ] Evaluate FS notify for watch
- [ ] Periodic review calendar
- [ ] Update 0004 bootstrap accordingly

### Core logic

- [ ] Evaluate CLI library candidates
- [ ] Evaluate transform parser deps
- [ ] Document rejected alternatives
- [ ] Agent must not add deps without ADR when required

### CLI / UX

- [ ] Evaluate semver library candidates
- [ ] Pin versions in go.mod
- [ ] No dependency for YAGNI features
- [ ] Publish dependency roadmap table

### Tests & fixtures

- [ ] Evaluate archive/compress libs vs stdlib
- [ ] License compatibility checks
- [ ] Size/attack-surface notes

### Docs & observability

- [ ] Evaluate HTTP hardening needs
- [ ] govulncheck in CI
- [ ] Windows support required for chosen deps

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Roadmap covers CLI, semver, transform, FS, net, archive, security, release
- [ ] Acceptance: Every non-stdlib dep has rationale
- [ ] Acceptance: Vulncheck wired
- [ ] Fixture ready: `docs/dependencies/roadmap.md`
- [ ] Fixture ready: `docs/adr/`


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
| Go dependency roadmap | Engineering | CLI/semver/FS/net/archive/security | 0085 |

## Go Package Map

**Packages / paths:**

- `docs/dependencies.md`
- `go.mod`

**Forbidden import edges:**

- New deps require ADR when not stdlib

## Data Flow

```mermaid
flowchart LR
  need[Need] --> stdlib[StdlibFirst] --> eval[EvaluateDep] --> pin[Pin] --> review[PeriodicReview]
```

## Commands and Flags

N/A — dependency governance.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Allowlist | Approved deps |
| ADR per heavy dep | Rationale |

## Concrete Test Fixtures

- `docs/dependencies/roadmap.md`
- `docs/adr/`

## Acceptance Scenarios

1. Roadmap covers CLI, semver, transform, FS, net, archive, security, release
2. Every non-stdlib dep has rationale
3. Vulncheck wired

## Nub Conformance Targets

- Dependency discipline | extension

## Open Decisions

- Specific CLI library choice (ties 0010)
- Transform parser library (ties 0051/0089)

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
