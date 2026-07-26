# 0074 — Distribution MVP 3 — Docker Images and Hosted Builder Integration

## Document Control

| Item | Detail |
|---|---|
| Phase | Distribution / MVP 3 |
| Primary objective | Provide minimal container images, cache-efficient Docker patterns, and adapters for hosted build systems. |
| Required predecessors | 0029, 0060, 0072 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Provide minimal container images, cache-efficient Docker patterns, and adapters for hosted build systems.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0029 before starting this MVP.
- Complete and merge 0060 before starting this MVP.
- Complete and merge 0072 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub Docker images and deployment guidance

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
docker run --rm mewjs/m:latest m --version
```

## In Scope

- Slim and full images.
- Multi-architecture manifests.
- Rootless operation and writable cache paths.
- BuildKit cache mounts and capsule workflows.
- Hosted-builder install snippets and lockfile detection.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Images are generated from signed release artifacts.
- No registry credentials in layers.
- Document libc and native-addon implications.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Create slim and full Dockerfiles from signed release artifacts
- [ ] Add hosted-builder install snippets (Railway, Fly, etc.) where applicable
- [ ] Document libc and native-addon implications for musl vs glibc
- [ ] Benchmark image size and cold m install in container

### Core logic

- [ ] Implement multi-architecture build and publish pipeline
- [ ] Detect lockfile type in container entry guidance
- [ ] Never embed registry credentials in image layers
- [ ] Add docker compose examples for monorepo CI

### CLI / UX

- [ ] Run containers as non-root with documented writable cache paths
- [ ] Scan images for vulnerabilities in CI
- [ ] Pin image tags to reproducible release versions
- [ ] Link docs from 0073 GitHub Action for hybrid workflows

### Tests & fixtures

- [ ] Document BuildKit cache mount patterns for store and transform caches
- [ ] Smoke-test rootless and read-only filesystem recipes
- [ ] Coordinate Node provisioning inside images with 0060

### Docs & observability

- [ ] Provide capsule workflow examples for immutable CI dependencies
- [ ] Test multi-arch execution on amd64 and arm64
- [ ] Publish read-only filesystem workaround patterns

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: docker run mewjs/m:latest m --version succeeds on amd64 and arm64
- [ ] Acceptance: Images run non-root by default with working cache dirs
- [ ] Acceptance: Vulnerability scan gate passes on release images
- [ ] Acceptance: BuildKit examples demonstrate cache-efficient m install
- [ ] Acceptance: No credentials appear in image layers or history
- [ ] Fixture ready: `docker/smoke — m --version and m install smoke`
- [ ] Fixture ready: `docker/rootless — non-root cache writable paths`
- [ ] Fixture ready: `docker/read-only — read-only root recipes`
- [ ] Fixture ready: `docker/multi-arch — amd64/arm64 execution`
- [ ] Fixture ready: `docker/buildkit — cache mount examples`


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
| Docker images mewjs/m | Nub Docker images | slim + full variants | 0074 |
| Multi-arch manifests | Nub | linux/amd64, arm64 | 0074 |
| BuildKit cache mounts | Nub deployment guidance | cache-efficient patterns | 0074 |
| Rootless operation | Nub | writable cache paths | 0074 |

## Go Package Map

**Packages / paths:**

- `docker/`
- `docs/ci/docker.md`
- `builders/`

**Forbidden import edges:**

- internal/resolver
- internal/linker

## Data Flow

```mermaid
flowchart LR
  flowchart LR
  release[SignedRelease] --> image[DockerBuild]
  image --> manifest[MultiArchManifest]
  manifest --> run[ContainerRun]
  run --> smoke[SmokeTests]
```

## Commands and Flags

| Surface | Example | Notes |
|---|---|---|
| `docker run mewjs/m:latest m --version` | slim image | Non-root default |
| BuildKit cache mount docs | RUN --mount=type=cache | Store/transform caches |
| Capsule workflows | documented patterns | Immutable CI deps |

Images built from signed 0072 artifacts; no registry creds in layers.

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| docker/Dockerfile.slim, Dockerfile.full | Image definitions |
| Multi-arch manifest lists | Published image indexes |
| BuildKit example Dockerfiles | Cache and capsule patterns |

## Concrete Test Fixtures

- `docker/smoke — m --version and m install smoke`
- `docker/rootless — non-root cache writable paths`
- `docker/read-only — read-only root recipes`
- `docker/multi-arch — amd64/arm64 execution`
- `docker/buildkit — cache mount examples`

## Acceptance Scenarios

1. docker run mewjs/m:latest m --version succeeds on amd64 and arm64
2. Images run non-root by default with working cache dirs
3. Vulnerability scan gate passes on release images
4. BuildKit examples demonstrate cache-efficient m install
5. No credentials appear in image layers or history

## Nub Conformance Targets

- Nub Docker images and deployment guidance | parity

## Open Decisions

- Default base image: Debian vs Alpine vs distroless
- Whether full image bundles Node or relies on m node provision

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
