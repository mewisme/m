# 0020 — Core MVP 11 — Full Dependency Resolver

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 11 |
| Primary objective | Complete resolver semantics for peer dependencies, optional dependencies, overrides, aliases, platforms, workspace protocols, and deterministic incremental updates. |
| Required predecessors | 0019 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Complete resolver semantics for peer dependencies, optional dependencies, overrides, aliases, platforms, workspace protocols, and deterministic incremental updates.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0019 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube resolver peer contexts and auto-installed peers
- Nub optional, peer, override, and compatibility behavior

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m update
```
```bash
m explain peer react
```
```bash
m resolve --trace
```

## In Scope

- Peer dependency contexts and optional peers.
- Auto-installed peer policy with strict modes.
- Optional dependencies and platform/CPU/libc filters.
- Overrides, resolutions, aliases, npm protocol, workspace protocol, file/link/portal placeholders.
- Incremental reuse of a previous lock graph.
- Conflict explanations and actionable diagnostics.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Model peer context in package identity.
- Keep policy choices explicit and recorded in lockfile settings.
- Preserve unaffected graph regions when updating targeted packages.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement peer dependency constraint collection per importer
- [ ] Support npm: alias protocol and package aliases
- [ ] Emit conflict explanation tree for unsatisfiable peers
- [ ] Fail with actionable errors for missing workspace targets

### Core logic

- [ ] Generate peer contexts as part of package identity
- [ ] Resolve workspace:* and workspace:^ protocol to local packages
- [ ] Record resolver policy choices in m.lock settings
- [ ] Benchmark resolver on large monorepo fixture

### CLI / UX

- [ ] Implement auto-install peers policy with strict and loose modes
- [ ] Support file:, link:, and portal: source placeholders
- [ ] Add conformance fixtures for peer, optional, override cases

### Tests & fixtures

- [ ] Prune optional dependencies failing os/cpu/libc filters
- [ ] Implement incremental lock reuse preserving unaffected subgraph
- [ ] Add workspace protocol resolution tests

### Docs & observability

- [ ] Apply overrides and resolutions rewriting dependency edges
- [ ] Minimize graph churn on targeted m update
- [ ] Document peer context ID format

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Conflicting peer deps produce explanation tree not silent wrong version
- [ ] Acceptance: workspace:* resolves to correct local package version
- [ ] Acceptance: Optional dep skipped on unsupported platform
- [ ] Acceptance: Override replaces transitive version deterministically
- [ ] Acceptance: Targeted update preserves unrelated lock subgraph
- [ ] Fixture ready: `fixtures/resolver/peers/react-ecosystem`
- [ ] Fixture ready: `fixtures/resolver/optional-platform`
- [ ] Fixture ready: `fixtures/resolver/overrides-nested`
- [ ] Fixture ready: `fixtures/projects/workspace-protocol`


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
| Peer dependencies | npm/Aube | Context-aware resolution | 0020 |
| Optional + platform filters | npm optionalDependencies | OS/CPU/libc pruning | 0020 |
| Overrides and aliases | npm overrides | Graph rewriting | 0020 |
| Workspace protocol | pnpm workspace:* | Monorepo local deps | 0020, 0022 |

## Go Package Map

**Packages / paths:**

- `internal/resolver`
- `internal/manifest`
- `internal/workspace`
- `internal/lockfile`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  manifest[declarations] --> peer[peer contexts] --> opt[optional/platform] --> ovr[overrides] --> ws[workspace:*] --> graph[CanonicalGraph]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m update` | `--latest`, `--interactive` | Targeted graph updates |
| `m resolve` | `--trace` | Full resolver trace |
| `m explain peer <pkg>` | — | Peer conflict diagnostics |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Extended m.lock peer contexts | Peer resolution record |
| Resolver policy settings | Auto-install peers strictness |

## Concrete Test Fixtures

- `fixtures/resolver/peers/react-ecosystem`
- `fixtures/resolver/optional-platform`
- `fixtures/resolver/overrides-nested`
- `fixtures/projects/workspace-protocol`

## Acceptance Scenarios

1. Conflicting peer deps produce explanation tree not silent wrong version
2. workspace:* resolves to correct local package version
3. Optional dep skipped on unsupported platform
4. Override replaces transitive version deterministically
5. Targeted update preserves unrelated lock subgraph

## Nub Conformance Targets

- npm peer dependency semantics | parity
- pnpm workspace protocol | parity
- npm overrides | parity
- optionalDependencies platform gating | parity

## Open Decisions

- Default auto-install-peers policy strictness
- portal: vs link: semantics alignment with pnpm

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
