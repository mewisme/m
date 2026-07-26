# 0027 — Core MVP 18 — Advanced Sources, Patches, Pack, and Publish

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 18 |
| Primary objective | Support non-registry package sources, package patches, deterministic packing, and authenticated publication with provenance hooks. |
| Required predecessors | 0026 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Support non-registry package sources, package patches, deterministic packing, and authenticated publication with provenance hooks.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0026 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Aube source resolvers and publish feature
- Nub publish command and workspace filters

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m add github:user/repo
```
```bash
m patch package
```
```bash
m pack
```
```bash
m publish --provenance
```

## In Scope

- Git URLs and hosted Git shortcuts.
- Local directories, files, tarballs, link/portal/workspace sources.
- Package alias and patch protocols.
- Deterministic package selection and tarball creation.
- Registry publication, tags, access, OTP, dry run, provenance, and workspace publish selection.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Normalize each source into a content identity and recorded resolution.
- Never execute Git hooks or user shell configuration implicitly.
- Packing must use documented inclusion rules and deterministic metadata.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Support git+https, git+ssh, and github: dependency sources
- [ ] Implement m pack producing npm-compatible tarball
- [ ] Sandbox git fetch network access per policy
- [ ] Document supported source protocols matrix

### Core logic

- [ ] Support file: and tarball: local dependency paths
- [ ] Validate package files field and .npmignore on pack
- [ ] Add provenance attestation hook points (optional)
- [ ] Never execute arbitrary scripts from git deps without policy

### CLI / UX

- [ ] Fetch git sources at resolved commit/tag with submodule policy
- [ ] Implement m publish with registry auth and OTP support
- [ ] Add tests for git dep, file dep, patch, pack fixtures

### Tests & fixtures

- [ ] Implement pnpm-style patch commit workflow (m patch)
- [ ] Record non-registry sources in m.lock with integrity
- [ ] Redact credentials in publish error output

### Docs & observability

- [ ] Apply patches deterministically during install
- [ ] Validate git URL and ref before fetch
- [ ] Support --dry-run on publish

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: Git dependency installs at pinned commit
- [ ] Acceptance: Applied patch changes installed file content deterministically
- [ ] Acceptance: m pack tarball matches npm pack file list
- [ ] Acceptance: m publish --dry-run validates without uploading
- [ ] Acceptance: file: dependency resolves relative to manifest
- [ ] Fixture ready: `fixtures/sources/git-dep`
- [ ] Fixture ready: `fixtures/sources/file-dep`
- [ ] Fixture ready: `fixtures/sources/patch-left-pad`
- [ ] Fixture ready: `fixtures/pack/minimal-package`


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
| git/tarball/file sources | npm protocols | Non-registry deps | 0027 |
| Package patches | pnpm patch | Deterministic patched installs | 0027 |
| m pack | npm pack | Deterministic tarball creation | 0027 |
| m publish | npm publish | Authenticated registry publish | 0027 |

## Go Package Map

**Packages / paths:**

- `internal/fetch`
- `internal/manifest`
- `internal/registry`
- `internal/archive`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  src[git|file|tarball] --> fetch[fetch] --> patch[patch apply] --> pack[m pack] --> pub[m publish]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m add <git url>` | `--save-git` | Git dependency sources |
| `m patch <pkg>` | `--edit-dir` | Patch workflow |
| `m pack` | `--pack-destination` | Create tarball |
| `m publish` | `--tag`, `--access` | Registry publish |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| `patches/` directory | Committed package patches |
| Packed .tgz | publish input |
| Provenance attestation hooks | Optional sigstore |

## Concrete Test Fixtures

- `fixtures/sources/git-dep`
- `fixtures/sources/file-dep`
- `fixtures/sources/patch-left-pad`
- `fixtures/pack/minimal-package`

## Acceptance Scenarios

1. Git dependency installs at pinned commit
2. Applied patch changes installed file content deterministically
3. m pack tarball matches npm pack file list
4. m publish --dry-run validates without uploading
5. file: dependency resolves relative to manifest

## Nub Conformance Targets

- npm git/file/tarball protocols | parity
- pnpm patch workflow | parity
- npm pack file selection | parity
- npm publish auth | parity

## Open Decisions

- Git submodule inclusion policy
- Provenance signing on by default or opt-in

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
