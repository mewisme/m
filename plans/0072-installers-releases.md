# 0072 — Distribution MVP 1 — Releases, Installers, and Package Channels

## Document Control

| Item | Detail |
|---|---|
| Phase | Distribution / MVP 1 |
| Primary objective | Produce signed, reproducible multi-platform releases and safe install/update paths for direct download and common package channels. |
| Required predecessors | 0031, 0046, 0057, 0062 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Produce signed, reproducible multi-platform releases and safe install/update paths for direct download and common package channels.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0031 before starting this MVP.
- Complete and merge 0046 before starting this MVP.
- Complete and merge 0057 before starting this MVP.
- Complete and merge 0062 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub install scripts, npm package distribution, and release build patterns

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m upgrade
```
```bash
m version --json
```

## In Scope

- Linux, macOS, and Windows binaries for supported architectures.
- Embedded runtime assets and license manifests.
- Checksums, signatures, attestations, and SBOMs.
- POSIX and PowerShell installers.
- Homebrew, Scoop, Winget, npm bootstrap package, and optional Nix packaging.
- Self-update with rollback.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Installer never executes an unverified downloaded binary.
- Self-update uses staged replacement and retains a recoverable previous version.
- Channel manifests are signed and versioned.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Configure GoReleaser or equivalent with reproducible build flags
- [ ] Publish Homebrew, Scoop, Winget, and npm bootstrap package definitions
- [ ] Add clean VM installation tests per platform
- [ ] Publish benchmark baselines alongside release artifacts

### Core logic

- [ ] Build release matrix for Linux, macOS, Windows supported architectures
- [ ] Implement signed versioned channel manifests
- [ ] Test interrupted replacement and locked-file Windows cases
- [ ] Never commit signing keys; use CI secret management

### CLI / UX

- [ ] Embed runtime assets and license manifests in release artifacts
- [ ] Implement m upgrade with staged replacement and rollback
- [ ] Document uninstall instructions per channel
- [ ] Include m.lock compatibility note in release notes template

### Tests & fixtures

- [ ] Generate checksums, signatures, attestations, and SBOMs
- [ ] Retain recoverable previous version after update
- [ ] Coordinate default shim paths with 0062

### Docs & observability

- [ ] Implement POSIX and PowerShell installers with verification step
- [ ] Reject tampered manifest or binary during install/update
- [ ] Require 0046 and 0057 stabilization before stable channel

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Install script verifies checksum/signature before executing m
- [ ] Acceptance: m upgrade rolls back safely on failure
- [ ] Acceptance: Release artifacts include m and mx for all supported platforms
- [ ] Acceptance: Channel manifests are signed and versioned
- [ ] Acceptance: SBOM and provenance published per release
- [ ] Fixture ready: `tests/release/clean-vm — per-OS install smoke`
- [ ] Fixture ready: `tests/release/tamper — rejected signatures/checksums`
- [ ] Fixture ready: `tests/release/upgrade-rollback — self-update recovery`
- [ ] Fixture ready: `tests/release/windows-locked — file replacement edge cases`
- [ ] Fixture ready: `tests/release/channels — manifest parsing matrix`


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
| Multi-platform releases | Nub release builds | signed m/mx binaries | 0072 |
| Install scripts | Nub install.sh/ps1 | POSIX + PowerShell | 0072 |
| Package channels | Nub npm bootstrap | Homebrew/Scoop/Winget/npm | 0072 |
| Self-update | Nub | staged replacement + rollback | 0072 |

## Go Package Map

**Packages / paths:**

- `cmd/m`
- `cmd/mx`
- `.goreleaser/`
- `installers/`
- `docs/install.md`

**Forbidden import edges:**

- internal/resolver
- internal/linker
- internal/runtime (feature work)

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  build[ReproducibleBuild] --> sign[SignAttest]
  sign --> channel[ChannelManifest]
  channel --> install[Installer]
  install --> update[SelfUpdate]
```

## Commands and Flags

| Command | Flags | Notes |
|---|---|---|
| `m upgrade` | channel selector | Self-update with rollback |
| `m version --json` | `--json` | Version + channel metadata |

Installer never executes unverified downloaded binary.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Release archives per OS/arch | m, mx binaries + embedded runtime assets |
| checksums.txt + signatures | Verification material |
| SBOM + provenance attestations | Supply chain metadata |

## Concrete Test Fixtures

- `tests/release/clean-vm — per-OS install smoke`
- `tests/release/tamper — rejected signatures/checksums`
- `tests/release/upgrade-rollback — self-update recovery`
- `tests/release/windows-locked — file replacement edge cases`
- `tests/release/channels — manifest parsing matrix`

## Acceptance Scenarios

1. Install script verifies checksum/signature before executing m
2. m upgrade rolls back safely on failure
3. Release artifacts include m and mx for all supported platforms
4. Channel manifests are signed and versioned
5. SBOM and provenance published per release

## Nub Conformance Targets

- Nub install scripts and npm distribution | parity
- Nub release build patterns | reference

## Open Decisions

- Codesigning requirements for macOS/Windows stable channel
- Whether npm package wraps binaries or bootstrap script only

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
