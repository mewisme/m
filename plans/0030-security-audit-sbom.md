# 0030 — Core MVP 21 — Audit, SBOM, Provenance, and Supply-Chain Policy

## Document Control

| Item | Detail |
|---|---|
| Phase | Core / MVP 21 |
| Primary objective | Provide comprehensive dependency risk analysis, signed provenance verification, SBOM export, age policies, and enforceable organizational rules. |
| Required predecessors | 0012, 0021, 0027, 0029 |
| Primary binaries | `m`, `mx` where applicable |
| Implementation language | Go for control plane and package-manager internals; embedded JavaScript only where Node extension APIs require it |
| Status at plan creation | Planned |

## Objective

Provide comprehensive dependency risk analysis, signed provenance verification, SBOM export, age policies, and enforceable organizational rules.

This MVP must be independently reviewable, testable, and releasable behind an explicit experimental gate when it changes public behavior. It must leave stable interfaces for later MVPs and avoid shortcuts that make lockfile compatibility, transactional installation, or cross-platform support impossible.

## Sequence and Dependencies

- Complete and merge 0012 before starting this MVP.
- Complete and merge 0021 before starting this MVP.
- Complete and merge 0027 before starting this MVP.
- Complete and merge 0029 before starting this MVP.

Work inside this MVP should normally follow this order:

1. Freeze behavior and data contracts with fixtures and interface tests.
2. Implement the smallest vertical path that reaches a user-visible result.
3. Add failure injection and cross-platform coverage before broadening features.
4. Measure performance and resource use before declaring the MVP complete.
5. Record intentional differences from Nub and from incumbent package managers.

## Nub Reference Mapping

- Nub/Aube security policy capabilities
- npm provenance and audit concepts
- pnpm supply-chain policy concepts

The Nub implementation is a behavioral reference, not a source-level porting target. Mew must preserve observable semantics where parity is intentional while selecting Go-native architecture, concurrency, storage, and error-handling patterns.

## User-Visible Surface

```bash
m audit
```
```bash
m sbom --format cyclonedx
```
```bash
m policy check
```
```bash
m verify provenance
```

## In Scope

- Advisory database ingestion and caching.
- Reachability-aware reporting where practical.
- CycloneDX and SPDX outputs.
- Registry signature and provenance verification.
- Minimum release age, deprecation, license, registry, source, script, and maintainer-change policies.
- Policy-as-code with explainable denials and waivers.

## Explicit Non-Goals

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Architecture and Interfaces

- Security policy evaluates the canonical graph and install plan before commit.
- Advisory data freshness and confidence are visible.
- Waivers are scoped, expiring, attributable, and never silently global.

Every new package must expose narrow interfaces, accept `context.Context` for cancellable work, avoid global mutable state, and return typed errors carrying stable Mew error codes. Persistent formats must be versioned and deterministic.

## Detailed Implementation Checklist

### Contracts & types

- [ ] Implement m audit against OSV/npm advisory data
- [ ] Implement dependency age policy (minimum release age)
- [ ] Add SBOM golden tests validating schema
- [ ] Stable JSON schema for audit output

### Core logic

- [ ] Support offline audit from cached advisory DB
- [ ] Implement org policy file for deny/warn on licenses and packages
- [ ] Document trust model integration with 0021 lifecycle policy
- [ ] Integrate policy checks into transaction validate phase

### CLI / UX

- [ ] Implement m sbom CycloneDX and SPDX export
- [ ] Fail install when policy severity exceeds threshold
- [ ] Support m audit --fix suggesting safe bumps

### Tests & fixtures

- [ ] Include direct and transitive deps in SBOM
- [ ] Redact internal package names in SBOM if configured
- [ ] Cache advisory DB with signature verification

### Docs & observability

- [ ] Verify package provenance attestations when present
- [ ] Add audit fixtures with known vulnerable versions
- [ ] Never phone home with project source code

## Test Plan

<!-- ENRICHMENT-TESTS -->
- [ ] Acceptance: m audit reports known CVE on fixture vulnerable package
- [ ] Acceptance: m sbom output validates against CycloneDX schema
- [ ] Acceptance: Policy deny blocks install of blocked package
- [ ] Acceptance: Provenance verify passes on signed fixture package
- [ ] Acceptance: Audit works offline with cached advisory DB
- [ ] Fixture ready: `fixtures/audit/vulnerable-transitive`
- [ ] Fixture ready: `fixtures/sbom/medium-graph-cyclonedx-golden.json`
- [ ] Fixture ready: `fixtures/policy/deny-gpl`


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
| m audit | npm audit | Vulnerability report | 0030 |
| SBOM export | CycloneDX/SPDX | Machine-readable bill of materials | 0030 |
| Provenance verify | sigstore/npm | Signed package attestation | 0030 |
| Org policy rules | Nub policy | Enforceable constraints | 0030 |

## Go Package Map

**Packages / paths:**

- `internal/policy`
- `internal/registry`
- `internal/resolver`
- `internal/diagnostics`
- `internal/app`

**Forbidden import edges:**

- None beyond architecture rules in 0003 / AGENTS.md.

## Data Flow

```mermaid
flowchart LR
  lock[installed graph] --> audit[vuln DB] --> sbom[SBOM export] --> prov[provenance] --> policy[org rules]
```

## Commands and Flags

| Surface | Flags | Notes |
|---|---|---|
| `m audit` | `--json`, `--fix` | Vulnerability scan |
| `m sbom` | `--format cyclonedx` | SBOM generation |
| `m trust verify` | — | Provenance checks |
| Policy config | `.github.com/mewisme/mew/policy.toml` | Org rule file |

## Persistent Artifacts

| Artifact | Purpose |
|---|---|
| Audit report JSON | CI gate input |
| SBOM document | cyclonedx.json or spdx.json |
| Policy evaluation log | Deny/warn decisions |

## Concrete Test Fixtures

- `fixtures/audit/vulnerable-transitive`
- `fixtures/sbom/medium-graph-cyclonedx-golden.json`
- `fixtures/policy/deny-gpl`

## Acceptance Scenarios

1. m audit reports known CVE on fixture vulnerable package
2. m sbom output validates against CycloneDX schema
3. Policy deny blocks install of blocked package
4. Provenance verify passes on signed fixture package
5. Audit works offline with cached advisory DB

## Nub Conformance Targets

- npm audit report shape | parity
- CycloneDX SBOM completeness | parity
- Nub security policy model | parity

## Open Decisions

- Default policy severity threshold for install block
- Advisory DB update cadence and mirror strategy

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
