# 0025 — Core MVP 16 — Bun and Yarn Lockfile Compatibility

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 16 |
| Primary objective | Implement Bun lock compatibility and a staged Yarn strategy covering classic read support and certified Berry/PnP read/write behavior. |
| Required predecessors | 0023, 0024 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Implement Bun lock compatibility and a staged Yarn strategy covering classic read support and certified Berry/PnP read/write behavior.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0023 before starting this MVP.
- Complete and merge 0024 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub Bun lockfile compatibility
- Nub Yarn read behavior and planned Berry PnP write work
- Aube Bun/Yarn lock parsers

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m lock validate --as bun
```
```bash
m lock validate --as yarn
```
```bash
m lock migrate --from bun --to m
```

## In Scope

- Bun text lock formats in supported releases.
- Yarn classic parsing and explicit write limitation unless separately certified.
- Yarn Berry descriptors, resolutions, checksums, cache ZIPs, PnP metadata, and node-modules mode.
- Per-major compatibility certification.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Do not advertise Yarn write support until generated locks and artifacts pass pinned Yarn immutable/operability tests.
- Keep PnP generation behind a dedicated adapter and runtime integration boundary.
- Report unavoidable churn and losses before migration.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement bun.lock parser adapter to canonical graph
- [ ] Preserve yarn.lock on Yarn classic identity projects
- [ ] Add differential install tests where reference tool available
- [ ] Validate parser against fuzz corpora

### Core logic

- [ ] Implement yarn.lock classic parser
- [ ] Detect yarn berry via .yarnrc.yml and lockfile format
- [ ] Handle yarn resolutions field mapping to overrides
- [ ] Emit migration report for lossy bun/yarn conversions

### CLI / UX

- [ ] Implement Yarn Berry lockfile read for node-modules mode
- [ ] Support migrate lock from bun/yarn to m.lock
- [ ] Support zero-install cache metadata read-only if present

### Tests & fixtures

- [ ] Certify Berry PnP read path or document explicit deferral
- [ ] Document unsupported Berry features with clear errors
- [ ] Never silently convert yarn/bun locks to m.lock

### Docs & observability

- [ ] Preserve bun.lock on Bun-identity projects
- [ ] Add golden fixtures per lock type
- [ ] Integrate identity detection from 0006

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: bun.lock fixture imports to valid install graph
- [ ] Acceptance: yarn.lock classic project installs with preserved lock
- [ ] Acceptance: Berry node-modules fixture installs without PnP
- [ ] Acceptance: Unsupported Berry feature fails with documented error
- [ ] Acceptance: Identity detection selects correct lock adapter
- [ ] Fixture ready: `fixtures/locks/bun/v1-basic`
- [ ] Fixture ready: `fixtures/locks/yarn/classic-v1`
- [ ] Fixture ready: `fixtures/locks/yarn/berry-nm`
- [ ] Fixture ready: `fixtures/locks/yarn/berry-pnp-readonly`


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
| bun.lock read | Bun | Import to canonical graph | 0025 |
| yarn.lock classic read | Yarn 1 | Read support | 0025 |
| Yarn Berry read/write | Yarn 2+ | Certified PnP subset | 0025 |
| Identity preservation | Mew policy | Per-manager lock files | 0025 |

## Go Package Map

**Packages / paths:**

- `internal/lockfile`
- `internal/compat/bun`
- `internal/compat/yarn`
- `internal/resolver`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  bunlock[bun.lock/yarn.lock] --> adapters[bun+yarn adapters] --> graph[CanonicalGraph]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m install` | — | Detect bun/yarn identity |
| `m migrate lock` | `--from yarn`, `--from bun` | Conversion paths |
| Berry | `--mode=node-modules` | When PnP unsupported |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| bun.lock / yarn.lock | Preserved per identity |
| PnP stub hooks | Deferred to runtime MVPs if needed |

## Concrete Test Fixtures

- `fixtures/locks/bun/v1-basic`
- `fixtures/locks/yarn/classic-v1`
- `fixtures/locks/yarn/berry-nm`
- `fixtures/locks/yarn/berry-pnp-readonly`

## Acceptance Scenarios

1. bun.lock fixture imports to valid install graph
2. yarn.lock classic project installs with preserved lock
3. Berry node-modules fixture installs without PnP
4. Unsupported Berry feature fails with documented error
5. Identity detection selects correct lock adapter

## Nub Conformance Targets

- bun.lock format | parity
- yarn.lock classic | parity
- Yarn Berry node-modules | parity
- Yarn Berry PnP | defer

## Open Decisions

- Berry PnP write support timeline vs read-only v1
- bun.lock text vs binary format versions

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
