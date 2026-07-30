# 0014 — Core MVP 5 — Tarball Fetch, Integrity, and Safe Extraction

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 5 |
| Primary objective | Download package tarballs concurrently, verify integrity before use, and extract archives without path traversal or partial-store corruption. |
| Required predecessors | 0012, 0013 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Download package tarballs concurrently, verify integrity before use, and extract archives without path traversal or partial-store corruption.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0012 before starting this MVP.
- Complete and merge 0013 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube fetch and CAS pipeline
- Nub SHA-512 preference and SHA-1 compatibility fallback

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m fetch --plan-file plan.json
```
```bash
m cache verify
```

## In Scope

- SRI SHA-512 and legacy shasum verification.
- Redirect-aware tarball downloads and resumable retry policy where safe.
- npm `package/` prefix handling.
- Safe tar and gzip parsing, file modes, symlinks, hardlinks, and executable bits.
- Temporary downloads and atomic publication.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Stream hashing while downloading.
- Reject absolute paths, `..`, device nodes, unsafe link targets, and resource-exhaustion archives.
- Keep archive bytes and extracted content separately addressable.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement tarball download from registry dist.tarball URL
- [ ] Reject absolute paths, .. segments, and symlink escapes in archives
- [ ] Store verified blobs in content-addressed staging area
- [ ] Implement download resume only if spec requires (document deferral)

### Core logic

- [ ] Verify dist.integrity sha512 before any extraction
- [ ] Normalize file modes and timestamps for reproducible extraction
- [ ] Add failure injection tests: corrupt tarball, wrong hash, disk full
- [ ] Clean up partial temp files on cancellation

### CLI / UX

- [ ] Support concurrent downloads with bounded worker pool
- [ ] Handle truncated downloads and checksum mismatch with retry
- [ ] Add cross-platform extraction tests on Windows junctions

### Tests & fixtures

- [ ] Write downloads to temp files and atomic rename on success
- [ ] Integrate with registry auth for private package tarballs
- [ ] Document fetch/archive interfaces for installer MVP

### Docs & observability

- [ ] Implement gzip tar extraction with path traversal prevention
- [ ] Support --offline: read from local cache only
- [ ] Redact signed URLs from error messages and logs

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Valid tarball extracts to expected file tree
- [ ] Acceptance: Integrity mismatch aborts before extraction
- [ ] Acceptance: Path traversal archive is rejected without writing files
- [ ] Acceptance: Concurrent downloads respect worker limit
- [ ] Acceptance: Cancelled download removes partial temp files
- [ ] Fixture ready: `fixtures/registry/v1/lodash-4.17.21.tgz`
- [ ] Fixture ready: `fixtures/archives/traversal-attack.tar`
- [ ] Fixture ready: `fixtures/archives/corrupt-hash.tgz`
- [ ] Fixture ready: `testdata/fetch/offline-cache-hit/`


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

- Verify integrity before extraction or execution.
- Enforce archive size, file-count, path-length, and expansion-ratio limits.
- Never trust archive ownership, special files, or absolute link targets.

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
| Tarball download | Nub fetch | Concurrent bounded workers | 0014 |
| Integrity verification | npm integrity | sha512 before extract | 0014 |
| Safe extraction | Nub archive | Path traversal guards | 0014 |
| Partial download recovery | Nub | Atomic temp files | 0014 |

## Go Package Map

**Packages / paths:**

- `internal/fetch`
- `internal/archive`
- `internal/registry`
- `internal/store`

**Forbidden import edges:**

- internal/linker
- internal/transaction

## Data Flow

```mermaid
flowchart LR
  reg[registry] --> fetch[internal/fetch] --> verify[IntegrityCheck] --> arch[internal/archive] --> store[ContentAddressedBlob]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| Internal only | — | No new user commands; used by install path |
| Env | proxy settings | Inherited from registry client |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Download staging temp files | Atomic rename on success |
| Extracted package tree | Input to linker/store |

## Concrete Test Fixtures

- `fixtures/registry/v1/lodash-4.17.21.tgz`
- `fixtures/archives/traversal-attack.tar`
- `fixtures/archives/corrupt-hash.tgz`
- `testdata/fetch/offline-cache-hit/`

## Acceptance Scenarios

1. Valid tarball extracts to expected file tree
2. Integrity mismatch aborts before extraction
3. Path traversal archive is rejected without writing files
4. Concurrent downloads respect worker limit
5. Cancelled download removes partial temp files

## Nub Conformance Targets

- npm dist.integrity verification | parity
- Tarball extraction safety | parity
- Concurrent fetch discipline | parity

## Open Decisions

- Whether to support download resume in v1
- Maximum tarball size limit policy

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
