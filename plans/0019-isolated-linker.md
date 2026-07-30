# 0019 — Core MVP 10 — Isolated Virtual Store and Node Modules Layout

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 10 |
| Primary objective | Implement a pnpm/Nub-style isolated dependency layout that prevents phantom dependencies while retaining a compatibility hoisted mode. |
| Required predecessors | 0018 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement a pnpm/Nub-style isolated dependency layout that prevents phantom dependencies while retaining a compatibility hoisted mode.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0018 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube linker and Nub default isolated node_modules model
- Nub compatibility modes for npm-like projects

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m install --node-linker isolated
```
```bash
m install --node-linker hoisted
```
```bash
m why lodash
```

## In Scope

- Project virtual store under node_modules.
- Direct dependency links at importer roots.
- Package-private dependency links.
- Peer-context-specific package instances.
- Hoisted compatibility mode and configurable public-hoist patterns.
- `.modules.yaml`-equivalent Mew metadata or a new documented format.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Layout generation consumes the canonical graph and emits an operation plan.
- Do not encode resolver decisions in linker traversal.
- Preserve Node resolution semantics on all supported platforms.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement isolated linker creating per-package node_modules trees
- [ ] Integrate with global store from 0018 for file linking
- [ ] Add Windows junction tests for long paths
- [ ] Deterministic ordering of virtual store directory names

### Core logic

- [ ] Layout packages under node_modules/.pnpm/<id>/node_modules/
- [ ] Update m.lock settings block with linker mode
- [ ] Document isolated vs hoisted trade-offs
- [ ] Emit layout summary in install output

### CLI / UX

- [ ] Symlink or junction top-level aliases to isolated paths
- [ ] Handle peer dependency symlink targets in isolated layout
- [ ] Ensure transaction rollback works with isolated tree

### Tests & fixtures

- [ ] Prevent access to undeclared dependencies (phantom dep test)
- [ ] Create .bin shims resolving through isolated paths
- [ ] Validate staged isolated tree before commit

### Docs & observability

- [ ] Support hoisted mode as compatibility fallback via config
- [ ] Add integration tests comparing pnpm fixture layouts
- [ ] Support scoped packages in virtual store paths

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Isolated install blocks requiring undeclared dependencies
- [ ] Acceptance: pnpm-simple fixture layout matches expected structure
- [ ] Acceptance: Hoisted mode still works via --linker=hoisted
- [ ] Acceptance: Isolated .bin shims execute correctly on Windows
- [ ] Acceptance: Linker mode persists in m.lock settings
- [ ] Fixture ready: `fixtures/projects/isolated-basic`
- [ ] Fixture ready: `fixtures/projects/isolated-peers`
- [ ] Fixture ready: `fixtures/projects/phantom-dep-negative`
- [ ] Fixture ready: `testdata/linker/isolated-layout-golden/`


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
| Isolated node_modules | pnpm layout | Per-package virtual store | 0019 |
| Phantom dep prevention | pnpm | Strict dependency boundaries | 0019 |
| Hoisted compatibility mode | npm | Configurable linker setting | 0019, 0016 |
| .pnpm virtual store | pnpm | Symlink/junction structure | 0019 |

## Go Package Map

**Packages / paths:**

- `internal/linker/isolated`
- `internal/linker`
- `internal/store`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  store[global store] --> iso[linker/isolated] --> vstore[.pnpm virtual] --> nm[node_modules/.pnpm]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | `--linker=isolated`, `--linker=hoisted` | Linker mode selection |
| Config | linker setting in m.lock | Persisted preference |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| `node_modules/.pnpm/` | Virtual store layout |
| Linker mode in m.lock settings | Per-project policy |

## Concrete Test Fixtures

- `fixtures/projects/isolated-basic`
- `fixtures/projects/isolated-peers`
- `fixtures/projects/phantom-dep-negative`
- `testdata/linker/isolated-layout-golden/`

## Acceptance Scenarios

1. Isolated install blocks requiring undeclared dependencies
2. pnpm-simple fixture layout matches expected structure
3. Hoisted mode still works via --linker=hoisted
4. Isolated .bin shims execute correctly on Windows
5. Linker mode persists in m.lock settings

## Nub Conformance Targets

- pnpm isolated node_modules layout | parity
- Phantom dependency prevention | parity
- Hoisted compatibility mode | parity

## Open Decisions

- Default linker mode for new m.lock projects
- Peer symlink strategy in isolated layout

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
