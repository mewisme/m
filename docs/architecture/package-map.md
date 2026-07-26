# Package map

Authoritative listing of repository paths and one-line purposes.
Every path named in [`AGENTS.md`](../../AGENTS.md) repository shape must appear here.

Status column: `shipped` means a Go package or documented dir exists today;
`planned` means reserved for a later MVP (directory may be absent).

## Entry and presentation

| Path | Purpose | Status |
|---|---|---|
| `cmd/m/` | Primary CLI entrypoint binary | shipped |
| `cmd/mx/` | Package executor entrypoint binary | shipped |
| `internal/app/` | Process-level orchestration across domains | shipped |
| `internal/cli/` | Parsing, dispatch, help, completions | shipped |
| `internal/config/` | Layered configuration loader | shipped |
| `internal/diagnostics/` | Errors, progress, redaction, reporters | shipped |
| `internal/apperr/` | Typed ERR_M_* errors and exit mapping | shipped |
| `internal/trace/` | Lightweight in-process spans (no OTel) | shipped |
| `internal/charter/` | Charter consistency tests (docs gate) | shipped |
| `internal/archcheck/` | Import-graph and package-map acceptance tests | shipped |
| `internal/bootstrap/` | Clean-clone style repository gate tests | shipped |

## Package-manager domain

| Path | Purpose | Status |
|---|---|---|
| `internal/manifest/` | package.json read/normalize/edit | shipped |
| `internal/project/` | Project root discovery and identity | shipped |
| `internal/workspace/` | Workspace graph, filters, catalogs | shipped |
| `internal/registry/` | Registry clients, auth, metadata cache | shipped |
| `internal/resolver/` | Semver + graph resolution + traces | shipped |
| `internal/semver/` | npm-compatible range satisfaction (Masterminds/v3) | shipped |
| `internal/lockfile/` | Canonical graph + format adapters | shipped |
| `internal/lockfile/mlock/` | Native m.lock codec | shipped |
| `internal/fetch/` | Concurrent tarball download | shipped |
| `internal/archive/` | Safe extraction and path validation | shipped |
| `internal/store/` | Content-addressed global store | shipped |
| `internal/linker/` | Hoisted/isolated layouts + bins | shipped |
| `internal/linker/planner/` | hardlink/reflink/copy/symlink/junction | shipped |
| `internal/transaction/` | Stage, journal, commit, rollback | shipped |
| `internal/lifecycle/` | Dependency lifecycle scripts | shipped |
| `internal/policy/` | Trust and sandbox policy | shipped |
| `internal/graph/` | Shared canonical graph helpers | shipped |
| `internal/plan/` | Mutation plan types | shipped |
| `internal/snapshot/` | Install history snapshots | shipped |
| `internal/journal/` | Crash-recovery journals | planned |

## Runner and runtime

| Path | Purpose | Status |
|---|---|---|
| `internal/runner/` | Scripts, exec, dlx environment builder | shipped |
| `internal/process/` | Signals, shells, child execution | shipped |
| `internal/runtime/` | Node launch orchestration | shipped |
| `internal/runtime/assets/` | Embedded loader/preload JS | shipped |
| `internal/transform/` | Go transform service + IPC | shipped |
| `internal/node/` | Node discovery and provisioning | shipped |
| `internal/pmmanager/` | External PM detect/pin/invoke | shipped |
| `internal/shim/` | Cross-platform shims | planned |
| `runtime/` | Source for go:embed runtime assets | shipped |

## Compatibility, security, distribution

| Path | Purpose | Status |
|---|---|---|
| `internal/compat/` | Nub/npm/pnpm/Yarn/Bun adapters | shipped |
| `internal/audit/` | Advisory normalization | planned |
| `internal/sbom/` | CycloneDX/SPDX export | planned |
| `internal/provenance/` | Signature/provenance verify/emit | planned |
| `internal/capsule/` | Portable dependency capsules (descriptors) | shipped |
| `internal/plugin/` | External m-\<verb\> discovery (no in-process load) | planned |
| `internal/analysis/phantom/` | Optional phantom-dependency analysis | planned |

## Support and fixtures

| Path | Purpose | Status |
|---|---|---|
| `internal/testkit/` | Fixtures, clean-home, local registry | shipped |
| `internal/features/` | Feature inventory schema/runtime | shipped |
| `internal/releasetrain/` | MVP dependency graph validation | shipped |
| `fixtures/registry/` | Local packuments and tarballs | shipped |
| `fixtures/projects/` | Project corpora | shipped |
| `tests/` | Conformance, integration, soak, and benchmark suites | shipped |
| `tests/conformance/` | Differential conformance suites | shipped |
| `tests/integration/` | End-to-end integration suites | shipped |
| `benchmarks/` | Perf baselines | planned |

## Release and docs

| Path | Purpose | Status |
|---|---|---|
| `release/` | Release metadata and notes | planned |
| `install/` | install.sh / install.ps1 sources | planned |
| `.github/actions/` | GitHub Action sources | planned |
| `docker/` | Container images and Dockerfiles | planned |
| `docs/` | User and architecture docs | shipped |
| `docs/adr/` | Architecture decision records | shipped |
| `docs/architecture/` | This package map and boundary docs | shipped |
| `plans/` | Implementation archive | shipped |

## Deferred package decisions

- **No `internal/pm` umbrella.** Flat packages under `internal/` own package-manager
  domains. An umbrella package may be reconsidered only after two concrete callers
  need shared orchestration beyond `internal/app`.
- **`assets/runtime/`** in the 0003 plan listing is synonymous with top-level
  `runtime/` for embed sources; use `runtime/` and `internal/runtime/assets/`.
