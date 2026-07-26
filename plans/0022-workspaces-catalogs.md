# 0022 — Core MVP 13 — Workspaces, Catalogs, and Filtering

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 13 |
| Primary objective | Support monorepo discovery, workspace dependency graphs, catalogs, filters, root checks, and atomic multi-importer installation. |
| Required predecessors | 0011, 0020, 0021 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Support monorepo discovery, workspace dependency graphs, catalogs, filters, root checks, and atomic multi-importer installation.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0011 before starting this MVP.
- Complete and merge 0020 before starting this MVP.
- Complete and merge 0021 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub workspace detection and scripts
- Aube workspace filters and catalogs
- pnpm-compatible recursive and filter grammar where intentionally adopted

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m install -r
```
```bash
m add react --filter web
```
```bash
m list -r
```
```bash
m --filter "...api" install
```

## In Scope

- package.json and compatible workspace declarations.
- Workspace protocol and local package linking.
- Catalog and named-catalog resolution.
- Package selection filters, dependency/dependent traversal, directory selectors, and changed-since selectors.
- Atomic root transaction across all importers.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Create one canonical workspace graph shared by installer and runner.
- Reject duplicate package names and overlapping ambiguous roots.
- Store importer-specific dependency specifiers in lockfiles.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Parse pnpm catalog: and catalog:default in package.json
- [ ] Install all importers atomically in single transaction
- [ ] Add integration tests on workspace-simple and nested fixtures
- [ ] Support negation patterns in filters if pnpm-compatible

### Core logic

- [ ] Resolve catalog references to concrete versions in manifests
- [ ] Validate workspace: protocol targets exist in graph
- [ ] Ensure filter install does not break unrelated members
- [ ] Emit workspace install summary per importer

### CLI / UX

- [ ] Implement --filter pattern matching package names and paths
- [ ] Detect duplicate workspace package names across members
- [ ] Record per-importer sections in m.lock for all members

### Tests & fixtures

- [ ] Support -r recursive install across workspace members
- [ ] Support root package.json as workspace importer
- [ ] Document filter grammar compatibility with pnpm

### Docs & observability

- [ ] Resolve workspace dependency graph with topological ordering
- [ ] Implement m ls -r workspace tree listing
- [ ] Fail on catalog reference to undefined catalog entry

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m install -r installs all workspace members atomically
- [ ] Acceptance: catalog: deps resolve to catalog-defined versions
- [ ] Acceptance: --filter installs only matching packages and deps
- [ ] Acceptance: Broken workspace: reference fails with clear error
- [ ] Acceptance: m.lock contains importer section per workspace package
- [ ] Fixture ready: `fixtures/projects/workspace-simple`
- [ ] Fixture ready: `fixtures/projects/workspace-nested`
- [ ] Fixture ready: `fixtures/projects/workspace-catalog`
- [ ] Fixture ready: `fixtures/projects/workspace-filter`


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
| Workspace filters | pnpm --filter | Selective install/run prep | 0022 |
| Catalogs | pnpm catalog | Shared version definitions | 0022 |
| Multi-importer install | pnpm -r | Atomic workspace install | 0022 |
| Root checks | pnpm | Workspace protocol validation | 0022 |

## Go Package Map

**Packages / paths:**

- `internal/workspace`
- `internal/manifest`
- `internal/resolver`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  ws[workspace graph] --> cat[catalogs] --> filt[filters] --> multi[multi-importer resolve] --> install[atomic install]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | `--filter <pattern>` | Subset install |
| `m install -r` | — | All workspace packages |
| `m ls -r` | `--depth` | Workspace tree listing |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Catalog definitions in pnpm-workspace or package.json | Shared dep versions |
| Workspace filter resolution cache | CLI filter expansion |

## Concrete Test Fixtures

- `fixtures/projects/workspace-simple`
- `fixtures/projects/workspace-nested`
- `fixtures/projects/workspace-catalog`
- `fixtures/projects/workspace-filter`

## Acceptance Scenarios

1. m install -r installs all workspace members atomically
2. catalog: deps resolve to catalog-defined versions
3. --filter installs only matching packages and deps
4. Broken workspace: reference fails with clear error
5. m.lock contains importer section per workspace package

## Nub Conformance Targets

- pnpm workspace filters | parity
- pnpm catalogs | parity
- pnpm recursive install | parity

## Open Decisions

- catalog: storage location (pnpm-workspace.yaml vs package.json)
- Filter grammar subset for v1

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
