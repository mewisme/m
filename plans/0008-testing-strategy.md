# 0008 — Testing, Fixtures, Fuzzing, and Conformance Strategy

## Document Control

| Item | Detail |
|---|---|
| Phase | Foundation |
| Primary objective | Build the test infrastructure required to port behavior safely and verify package-manager compatibility without depending on public registries. |
| Required predecessors | 0004, 0007 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Build the test infrastructure required to port behavior safely and verify package-manager compatibility without depending on public registries.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0004 before starting this MVP.
- Complete and merge 0007 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub integration and conformance inventories
- Nub soak scripts and compatibility fixtures

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
go test ./...
```
```bash
go test -race ./...
```
```bash
go test -fuzz=Fuzz -run=^$ ./...
```

## In Scope

- Hermetic local npm registry fixture.
- Golden repositories for each lockfile format and manager major version.
- Filesystem capability matrix.
- Fault-injection proxy and process killer.
- Behavioral differential harness invoking Nub and incumbent managers where licensed and available.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Never make normal CI dependent on the public npm registry.
- Keep large ecosystem corpus tests in scheduled jobs.
- Normalize nondeterministic output before comparison.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Define fixture manifest format and checksums
- [ ] Implement reference PM invocation wrappers (optional when tools present)
- [ ] Failure injection helpers: network cut, disk full simulation
- [ ] Keep large ecosystem corpus tests in scheduled jobs

### Core logic

- [ ] Define clean-home test contract
- [ ] Define fuzz targets list for parsers
- [ ] Cross-platform path/symlink/junction probes
- [ ] Normalize nondeterministic output before comparison

### CLI / UX

- [ ] Define differential comparison report schema
- [ ] Document how to add a fixture
- [ ] Testing strategy doc with layout diagram
- [ ] Add known-bad corpus verifying parsers fail safely

### Tests & fixtures

- [ ] Implement local fixture registry server helper
- [ ] Document required metadata: OS, tool versions
- [ ] Conformance inventory stub for 0080

### Docs & observability

- [ ] Implement isolated HOME/XDG/cache redirection
- [ ] Smoke: install from fixture registry
- [ ] Never make normal CI dependent on the public npm registry

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Tests never require public registry access
- [ ] Acceptance: Clean-home tests do not touch developer global state
- [ ] Acceptance: Fixture checksums verified on load
- [ ] Acceptance: Differential harness smoke test passes on pinned Nub revision when available
- [ ] Fixture ready: `fixtures/registry/v1/lodash-4.17.21.tgz`
- [ ] Fixture ready: `fixtures/projects/basic-cjs`
- [ ] Fixture ready: `fixtures/projects/basic-esm`
- [ ] Fixture ready: `fixtures/projects/typescript-app`
- [ ] Fixture ready: `fixtures/projects/workspace-simple`


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
| Fixture registry | Nub/Aube tests | Local registry | 0008 |
| Conformance harness | Nub | Differential fixtures | 0008, 0080 |
| Clean-home isolation | Nub soak scripts | Hermetic test homes | 0008 |

## Go Package Map

**Packages / paths:**

- `internal/testkit`
- `tests/conformance`
- `tests/integration`
- `fixtures/registry`
- `fixtures/projects`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  fix[Fixtures] --> reg[LocalRegistry] --> mew[MewUnderTest] --> cmp[Compare] --> ref[ReferencePM]
```

## Commands and Flags

| Harness command | Purpose |
|---|---|
| `go test ./tests/...` | Suites |
| `make conformance` | Differential runs |
| `make fuzz-smoke` | Short fuzz |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| `fixtures/registry/v1/` | Packaged tarballs + metadata |
| `fixtures/projects/*` | Project corpora |
| Golden outputs | Lockfiles, trees, stdout |

## Concrete Test Fixtures

- `fixtures/registry/v1/lodash-4.17.21.tgz`
- `fixtures/projects/basic-cjs`
- `fixtures/projects/basic-esm`
- `fixtures/projects/typescript-app`
- `fixtures/projects/workspace-simple`

## Acceptance Scenarios

1. Tests never require public registry access
2. Clean-home tests do not touch developer global state
3. Fixture checksums verified on load
4. Differential harness smoke test passes on pinned Nub revision when available

## Nub Conformance Targets

- Local-registry testing discipline | parity

## Open Decisions

- Which reference PM versions to pin in CI images

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

## Fixture Registry and Harness Layout

```text
fixtures/
  registry/v1/
    packuments/           # npm-compatible packument JSON
    tarballs/             # *.tgz with recorded integrity
    manifest.json         # checksum index for all blobs
  projects/
    basic-cjs/
    basic-esm/
    typescript-app/
    workspace-simple/
    optional-deps/
    scoped-deps/
  identity/               # lockfile identity detection cases
  security/
    evil-archives/        # path traversal and link bombs (never extracted in prod paths)
tests/
  conformance/            # differential Mew vs reference PM
  integration/            # clean-home end-to-end
  fuzz/                   # parser fuzz smoke targets
internal/testkit/
  home.go                 # isolated HOME/XDG/cache
  registry.go             # local fixture registry server
  compare.go              # differential report helpers
```

## Initial Fuzz Targets

| Target | Package | Input |
|---|---|---|
| Manifest JSON | `internal/manifest` | malformed package.json |
| Lock codecs | `internal/lockfile/*` | truncated/garbage locks |
| Archive paths | `internal/archive` | crafted tar members |
| Semver ranges | `internal/semver` | random range strings |
| Config merge | `internal/config` | hostile config files |
