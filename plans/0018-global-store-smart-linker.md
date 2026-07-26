# 0018 — Core MVP 9 — Global Content Store and Smart Filesystem Planner

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 9 |
| Primary objective | Introduce an immutable global content-addressable store and automatically choose safe hardlink, reflink, copy, symlink, or junction strategies per filesystem. |
| Required predecessors | 0014, 0017 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Introduce an immutable global content-addressable store and automatically choose safe hardlink, reflink, copy, symlink, or junction strategies per filesystem.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0014 before starting this MVP.
- Complete and merge 0017 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube global virtual store and linker
- pnpm-style content reuse as behavioral reference

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m store path
```
```bash
m store status
```
```bash
m doctor filesystem
```

## In Scope

- Content-addressed archive and unpacked package objects.
- Store locks, leases, references, validation, repair, and garbage collection.
- Filesystem capability probe and persisted strategy decisions.
- Cross-device and Windows fallback behavior.
- Package mutation detection.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Store entries are immutable after publication.
- Physical package identity includes content, source, patch, build state, platform, and relevant policy.
- Linker planner returns an explainable operation plan.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement content-addressed global store keyed by integrity hash
- [ ] Use Windows junctions/symlinks per platform policy
- [ ] Add cross-platform tests: Linux, macOS, Windows linking
- [ ] Support MEW_STORE_DIR override with validation

### Core logic

- [ ] Import verified tarballs into store without duplication
- [ ] Integrate store with hoisted linker from 0016
- [ ] Add tests for cross-filesystem copy fallback
- [ ] Emit link strategy summary in install diagnostics

### CLI / UX

- [ ] Probe filesystem for hardlink, reflink, symlink, junction support
- [ ] Track store reference counts from project link manifests
- [ ] Document store layout and garbage collection rules

### Tests & fixtures

- [ ] Implement link planner choosing safest fastest strategy per path
- [ ] Implement m store path and m store prune commands
- [ ] Never mutate store blobs in place after import

### Docs & observability

- [ ] Fall back to copy when hardlink/reflink unavailable or cross-device
- [ ] Prune unreferenced blobs with dry-run preview
- [ ] Verify integrity on store read before linking

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Identical package imported twice shares one store blob
- [ ] Acceptance: Link planner selects copy on cross-device install
- [ ] Acceptance: Store prune removes only unreferenced blobs
- [ ] Acceptance: Corrupt store entry is detected and re-fetched
- [ ] Acceptance: m store path reports configured location
- [ ] Fixture ready: `testdata/store/layout-golden/`
- [ ] Fixture ready: `fixtures/store/cross-device-copy/`
- [ ] Fixture ready: `fixtures/store/reflink-probe/`


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
| Global content store | pnpm store | Immutable CAS blobs | 0018 |
| Smart link planner | Nub linker | hardlink/reflink/copy/symlink | 0018 |
| Filesystem probing | Nub | Per-OS capability detection | 0018 |
| Dedup across projects | pnpm | Shared store entries | 0018 |

## Go Package Map

**Packages / paths:**

- `internal/store`
- `internal/linker`
- `internal/linker/planner`
- `internal/fetch`

**Forbidden import edges:**

- internal/linker/isolated

## Data Flow

```mermaid
flowchart LR
  fetch[verified blob] --> store[internal/store] --> planner[linker/planner] --> link[hardlink|reflink|copy] --> nm[node_modules]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m store path` | — | Show global store location |
| `m store prune` | `--dry-run` | Remove unreferenced blobs |
| Env | `MEW_STORE_DIR` | Override store root |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Global content-addressed store | ~/.local/share/github.com/mewisme/m/store |
| Link plan records | Per-install strategy audit |

## Concrete Test Fixtures

- `testdata/store/layout-golden/`
- `fixtures/store/cross-device-copy/`
- `fixtures/store/reflink-probe/`

## Acceptance Scenarios

1. Identical package imported twice shares one store blob
2. Link planner selects copy on cross-device install
3. Store prune removes only unreferenced blobs
4. Corrupt store entry is detected and re-fetched
5. m store path reports configured location

## Nub Conformance Targets

- pnpm global store layout | parity
- Smart linking strategy selection | parity
- Store prune semantics | parity

## Open Decisions

- Default store location per OS (XDG vs Library/Caches)
- Reflink support detection on APFS/btrfs

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
