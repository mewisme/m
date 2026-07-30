# Package map

Authoritative listing of repository paths and one-line purposes.
Every path named in [`AGENTS.md`](../../AGENTS.md) repository shape must appear here.

**Path state** — `absent` (no directory), `reserved` (documented placeholder), `exists` (on disk today).

**Capability state** — `scaffolded`, `experimental`, `partial`, `shipped`, `certified`, `planned`, or `deferred`.

## Entry and presentation

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `cmd/m/` | Primary CLI entrypoint binary | exists | shipped |
| `cmd/mx/` | Package executor entrypoint binary | exists | shipped |
| `internal/app/` | Process-level orchestration across domains | exists | shipped |
| `internal/cli/` | Parsing, dispatch, help, completions | exists | shipped |
| `internal/config/` | Layered configuration loader | exists | shipped |
| `internal/diagnostics/` | Errors, progress, redaction, reporters | exists | shipped |
| `internal/apperr/` | Typed ERR_M_* errors and exit mapping | exists | shipped |
| `internal/trace/` | Lightweight in-process spans (no OTel) | exists | shipped |
| `internal/charter/` | Charter consistency tests (docs gate) | exists | shipped |
| `internal/archcheck/` | Import-graph and package-map acceptance tests | exists | shipped |
| `internal/bootstrap/` | Clean-clone style repository gate tests | exists | shipped |

## Package-manager domain

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `internal/manifest/` | package.json read/normalize/edit | exists | shipped |
| `internal/project/` | Project root discovery and identity | exists | shipped |
| `internal/workspace/` | Workspace graph, filters, catalogs | exists | shipped |
| `internal/registry/` | Registry clients, auth, metadata cache | exists | shipped |
| `internal/resolver/` | Semver + graph resolution + traces | exists | shipped |
| `internal/semver/` | npm-compatible range satisfaction (Masterminds/v3) | exists | shipped |
| `internal/lockfile/` | Canonical graph + format adapters | exists | shipped |
| `internal/lockfile/mlock/` | Native m.lock codec | exists | shipped |
| `internal/fetch/` | Concurrent tarball download | exists | shipped |
| `internal/archive/` | Safe extraction and path validation | exists | shipped |
| `internal/store/` | Content-addressed global store | exists | shipped |
| `internal/linker/` | Hoisted/isolated layouts + bins | exists | experimental |
| `internal/linker/planner/` | hardlink/reflink/copy/symlink/junction | exists | shipped |
| `internal/transaction/` | Stage, journal, commit, rollback | exists | shipped |
| `internal/lifecycle/` | Dependency lifecycle scripts | exists | shipped |
| `internal/policy/` | Trust and sandbox policy | exists | shipped |
| `internal/graph/` | Shared canonical graph helpers | exists | shipped |
| `internal/plan/` | Mutation plan types | exists | shipped |
| `internal/snapshot/` | Install history snapshots | exists | shipped |
| `internal/journal/` | Crash-recovery journals | reserved | planned |

## Runner and runtime

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `internal/runner/` | Scripts, exec, dlx environment builder | exists | scaffolded |
| `internal/process/` | Signals, shells, child execution | exists | shipped |
| `internal/runtime/` | Node launch orchestration | exists | scaffolded |
| `internal/runtime/assets/` | Embedded loader/preload JS | exists | scaffolded |
| `internal/transform/` | Go transform service + IPC | exists | scaffolded |
| `internal/node/` | Node discovery and provisioning | exists | scaffolded |
| `internal/pmmanager/` | External PM detect/pin/invoke | exists | shipped |
| `internal/shim/` | Cross-platform shims | reserved | planned |
| `runtime/` | Source for go:embed runtime assets | exists | scaffolded |

## Compatibility, security, distribution

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `internal/compat/` | Nub/npm/pnpm/Yarn/Bun adapters | exists | shipped |
| `internal/audit/` | Advisory normalization | reserved | planned |
| `internal/sbom/` | CycloneDX/SPDX export | exists | certified |
| `internal/provenance/` | Signature/provenance verify/emit | exists | shipped |
| `internal/capsule/` | Portable dependency capsules (descriptors) | exists | shipped |
| `internal/plugin/` | External m-\<verb\> discovery (no in-process load) | reserved | planned |
| `internal/analysis/phantom/` | Optional phantom-dependency analysis | reserved | planned |

Certified SBOM evidence: [`docs/evidence/core/pass32-ci.md`](../evidence/core/pass32-ci.md).

## Support and fixtures

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `internal/testkit/` | Fixtures, clean-home, local registry | exists | shipped |
| `internal/features/` | Feature inventory schema/runtime | exists | shipped |
| `internal/releasetrain/` | MVP dependency graph validation | exists | shipped |
| `fixtures/registry/` | Local packuments and tarballs | exists | shipped |
| `fixtures/projects/` | Project corpora | exists | shipped |
| `tests/` | Conformance, integration, soak, and benchmark suites | exists | shipped |
| `tests/conformance/` | Differential conformance suites | exists | shipped |
| `tests/integration/` | End-to-end integration suites | exists | shipped |
| `benchmarks/` | Perf baselines and waivers | exists | partial |

## Release and docs

| Path | Purpose | Path state | Capability state |
|---|---|---|---|
| `release/` | Release metadata and notes | reserved | planned |
| `install/` | install.sh / install.ps1 sources | reserved | planned |
| `.github/actions/` | GitHub Action sources | reserved | planned |
| `docker/` | Container images and Dockerfiles | reserved | planned |
| `docs/` | User and architecture docs | exists | shipped |
| `docs/adr/` | Architecture decision records | exists | shipped |
| `docs/architecture/` | This package map and boundary docs | exists | shipped |
| `plans/` | Implementation archive | exists | shipped |

## Deferred package decisions

- **No `internal/pm` umbrella.** Flat packages under `internal/` own package-manager
  domains. An umbrella package may be reconsidered only after two concrete callers
  need shared orchestration beyond `internal/app`.
- **`assets/runtime/`** in the 0003 plan listing is synonymous with top-level
  `runtime/` for embed sources; use `runtime/` and `internal/runtime/assets/`.
