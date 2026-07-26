# 0024 — Core MVP 15 — npm Lockfile and Shrinkwrap Compatibility

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 15 |
| Primary objective | Support modern package-lock and npm-shrinkwrap formats while preserving npm project identity and install semantics. |
| Required predecessors | 0023 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Support modern package-lock and npm-shrinkwrap formats while preserving npm project identity and install semantics.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0023 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub npm lockfile compatibility
- Aube npm lock parser and importer

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m install
```
```bash
m lock migrate --from npm --to m
```
```bash
m lock validate --as npm
```

## In Scope

- package-lock lockfile versions in active use.
- npm-shrinkwrap publication semantics.
- Packages map, resolved URLs, integrity, links, workspaces, bundled dependencies, and legacy fields.
- npm config identity and existing-lock preservation.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Treat npm layout metadata separately from the canonical logical graph.
- Do not promise byte-identical formatting; promise supported semantic compatibility.
- Run pinned npm clean-install validation.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement package-lock.json v2 and v3 parsers
- [ ] Import integrity and resolved URL fields from npm lock
- [ ] Add differential tests vs npm install on fixture projects
- [ ] Support workspaces in package-lock v3

### Core logic

- [ ] Map npm lock packages array to canonical graph nodes
- [ ] Support bundledDependencies and packages link fields
- [ ] Document npm-specific fields preserved in adapter
- [ ] Validate lockfilePackages ordering determinism on write

### CLI / UX

- [ ] Preserve npm project identity: write package-lock not m.lock
- [ ] Install produces npm-compatible hoisted layout
- [ ] Implement migrate to m.lock with loss report

### Tests & fixtures

- [ ] Support npm-shrinkwrap.json read and write
- [ ] Detect package-lock drift vs package.json on frozen install
- [ ] Never strip package-lock on npm-identity project install

### Docs & observability

- [ ] Handle lockfileVersion field and forward compatibility
- [ ] Add golden tests for npm lock v2/v3 fixtures
- [ ] Handle absent package-lock: generate on first install

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: npm fixture install matches package-lock dependency tree
- [ ] Acceptance: package-lock.json preserved after m install on npm project
- [ ] Acceptance: Frozen install fails when package.json conflicts with lock
- [ ] Acceptance: npm-shrinkwrap project installs correctly
- [ ] Acceptance: Lock v2 and v3 fixtures parse without error
- [ ] Fixture ready: `fixtures/locks/npm/v2-basic`
- [ ] Fixture ready: `fixtures/locks/npm/v3-workspaces`
- [ ] Fixture ready: `fixtures/projects/npm-app`
- [ ] Fixture ready: `testdata/lockfile/npm-roundtrip/`


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
| package-lock.json read | npm lockfile v2/v3 | Import to graph | 0024 |
| npm-shrinkwrap | npm | Same adapter path | 0024 |
| npm identity preservation | Mew policy | Keep package-lock on npm projects | 0024 |
| npm install semantics | npm | Hoisted layout default | 0024 |

## Go Package Map

**Packages / paths:**

- `internal/lockfile`
- `internal/compat/npm`
- `internal/resolver`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  pkglock[package-lock.json] --> npmAdapter[npm lock adapter] --> graph[CanonicalGraph] --> install[npm-style install]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | — | npm project uses package-lock |
| `m migrate lock --to m.lock` | `--dry-run` | Optional conversion |
| `m config set package-lock true` | — | Force lock generation |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| package-lock.json | Preserved on npm identity projects |
| npm-shrinkwrap.json | Supported read/write |

## Concrete Test Fixtures

- `fixtures/locks/npm/v2-basic`
- `fixtures/locks/npm/v3-workspaces`
- `fixtures/projects/npm-app`
- `testdata/lockfile/npm-roundtrip/`

## Acceptance Scenarios

1. npm fixture install matches package-lock dependency tree
2. package-lock.json preserved after m install on npm project
3. Frozen install fails when package.json conflicts with lock
4. npm-shrinkwrap project installs correctly
5. Lock v2 and v3 fixtures parse without error

## Nub Conformance Targets

- package-lock.json v2/v3 format | parity
- npm install hoisted layout | parity
- npm-shrinkwrap semantics | parity

## Open Decisions

- package-lock v1 read support scope
- Whether m writes lockfileVersion 3 by default

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
