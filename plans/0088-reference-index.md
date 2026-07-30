# 0088 — Reference Index and Research Sources

## Document Control

| Item | Detail |
|---|---|
| Phase | Cross-Cutting |
| Primary objective | Maintain the authoritative source list for Nub behavior, incumbent package-manager formats, Node APIs, registries, security standards, and Go implementation decisions. |
| Required predecessors | 0002, 0083 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Maintain the authoritative source list for Nub behavior, incumbent package-manager formats, Node APIs, registries, security standards, and Go implementation decisions.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0002 before starting this MVP.
- Complete and merge 0083 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub official repository and documentation
- Node.js official documentation
- npm registry and CLI documentation
- pnpm, Yarn, and Bun official documentation and source fixtures

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

No new command is required in this MVP.

## In Scope

- Pinned Nub commit and release.
- Official docs and relevant source paths.
- Lockfile format fixtures and manager versions.
- Node extension API references.
- Security and SBOM/provenance specifications.
- ADRs and benchmark reports.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Prefer primary sources and pinned versions.
- Record access date and revision for unstable behavior.
- Do not copy large copyrighted documentation sections; summarize behavior and link internally where repository policy permits.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Index Nub behavior sources with commit pins
- [ ] Index Go implementation decision sources
- [ ] No private URLs/secrets
- [ ] Cross-link 0083 migration map

### Core logic

- [ ] Index npm/pnpm/Yarn/Bun lock format docs
- [ ] Keep plans/sources synchronized
- [ ] Process to refresh before major implementation waves
- [ ] Publish docs/references/README

### CLI / UX

- [ ] Index Node loader/preload/inspector docs
- [ ] Record retrieval dates
- [ ] Agent must cite sources for parity claims
- [ ] Validate links periodically

### Tests & fixtures

- [ ] Index registry protocol docs
- [ ] Mark stale sources
- [ ] Separate normative vs informative

### Docs & observability

- [ ] Index security standards (SRI, SBOM, provenance)
- [ ] Link MVPs to references
- [ ] Include license notes

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Authoritative source list exists
- [ ] Acceptance: Nub commit pin recorded
- [ ] Acceptance: Parity claims can cite a source entry
- [ ] Fixture ready: `plans/sources/nub-reference-snapshot.md`
- [ ] Fixture ready: `plans/sources/compatibility-targets.md`
- [ ] Fixture ready: `docs/references/index.md`


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
| Reference index | Research | Authoritative sources | 0088 |

## Go Package Map

**Packages / paths:**

- `plans/sources/`
- `docs/references/`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  topic[Topic] --> source[PinnedSource] --> note[Notes] --> mvp[ConsumingMVP]
```

## Commands and Flags

N/A — bibliography.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Reference index | URLs + commits |
| sources/* | Plan research notes |

## Concrete Test Fixtures

- `plans/sources/nub-reference-snapshot.md`
- `plans/sources/compatibility-targets.md`
- `docs/references/index.md`

## Acceptance Scenarios

1. Authoritative source list exists
2. Nub commit pin recorded
3. Parity claims can cite a source entry

## Nub Conformance Targets

- Research hygiene | extension

## Open Decisions

- Automation for link checking

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
