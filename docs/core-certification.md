# Package-manager core certification

MVP **0031**. Evidence index for the PM core shipped in MVPs **0010–0030**.
Runner work (**0040+**) may depend on the install interfaces and schemas
frozen here; see [`schema-freeze.md`](schema-freeze.md).

Related: [`security-pm-core.md`](security-pm-core.md),
[`../testdata/certification/sign-off-checklist.md`](../testdata/certification/sign-off-checklist.md),
[`pm-commands.md`](pm-commands.md).

## Certification entry points

| Command | Purpose |
|---|---|
| `make core-cert` | Full local gate: core conformance + fixture verify + crash-shard verify |
| `go run ./cmd/m conformance run core [--json]` | Execute the core-matrix test suites |
| `go run ./cmd/m conformance list` | List suite ids from `tests/conformance/core-matrix/manifest.json` |
| `go run ./cmd/m doctor [--json] [--strict]` | Project and PM health checks |
| `go run ./cmd/m bench install [--warm\|--cold] [--json]` | Install benchmark (see [`performance.md`](performance.md)) |
| `pwsh tools/soak/install-loop.ps1 -Count <n> -Mode warm` | Repeated install soak (CI: `-Count 10`; manual: `-Count 100`) |
| `pwsh tools/bench/check_regression.ps1 -Mode warm` | Install performance regression gate |

`m benchmark` is the compatibility alias for `m bench` (0031 plan surface).

## Certification matrix

| Gate | Local reproduction | CI job |
|---|---|---|
| Unit + integration tests | `CGO_ENABLED=0 go test ./... -count=1` | `test` (ubuntu, macos, windows) |
| Race detector | `go test -race ./... -count=1` | `race`, `race-macos`, `race-windows` |
| No-CGO production build | `CGO_ENABLED=0 go build ./cmd/m ./cmd/mx` | `no-cgo-gate`, `cross` |
| Vet + lint + vuln + allowlist | `go vet ./...`; `golangci-lint run ./...`; `govulncheck ./...` | `lint`, `vuln`, `allowlist` |
| Platform lockfiles | — | `platform-lock` |
| Fixture provenance | `go run ./tools/conformance/verify-fixtures` | `fixture-verify` |
| Crash-shard assignment | `go run ./tools/ci/verify-crash-shards` | `crash-shard-verify` |
| Transaction / snapshot crash recovery | `go test -tags crash ./tests/integration/... -run Crash -timeout 30m` | `crash-integration` (+ Windows shards + `crash-integration-report`) |
| pnpm 9 lock bridge + mutation | `go test ./tests/conformance/... -run 'Pnpm9' -count=1` | `conformance-pnpm-9` |
| pnpm 10 lock bridge + mutation | `go test ./tests/conformance/... -run 'Pnpm10' -count=1` | `conformance-pnpm-10` |
| pnpm 11 lock bridge + mutation | `go test ./tests/conformance/... -run 'Pnpm11' -count=1` | `conformance-pnpm-11` |
| Unsupported pnpm rejection | `go test ./tests/conformance/... -run UnsupportedLegacy -count=1` | `conformance-pnpm-unsupported` |
| npm lock bridge (read-only) | `go test ./tests/conformance/... -run LockBridgeNpm -count=1` | `conformance-npm` |
| bun lock bridge | `go test ./tests/conformance/... -run LockBridgeBun -count=1` | `conformance-bun` |
| Yarn Classic + Berry | `go test ./tests/conformance/... -run LockBridgeYarn -count=1` | `conformance-yarn` |
| Nub derived fixtures | `go test ./tests/conformance/... -run LockBridgeNub -count=1` | `conformance-nub-fixtures` |
| Core conformance aggregate | `go run ./cmd/m conformance run core --json` | `core-stabilization` (0031) |
| PM health | `go run ./cmd/m doctor --json` | `core-stabilization` |
| Soak (short) | `pwsh tools/soak/install-loop.ps1 -Count 10 -Mode warm` | `core-stabilization` |
| Install bench regression | `pwsh tools/bench/check_regression.ps1 -Mode warm` | `bench-regression` |
| License + dependency allowlist | `go run ./tools/check-license`; `go run ./tools/check-deps` | `gate-probe` |

Pinned pnpm producer versions: `tools/conformance/pnpm-versions.env` (9.15.9 /
10.34.5 / 11.17.0). Inventory: [`tests/conformance/inventory.json`](../tests/conformance/inventory.json).

## Lock-bridge certification scope

Certified (fixture parse, graph conversion, byte-preserving no-op or mutation
where applicable):

- **npm** — `package-lock.json` v2/v3 corpus under `fixtures/locks/npm/` (read-only; semantic mutation rejected with `ERR_M_UNSUPPORTED`)
- **pnpm 9 / 10 / 11** — generated fixtures under `fixtures/locks/generated/pnpm-{9,10,11}/` including mutation families: basic, transitive, optional, peer-context, alias-peer, workspace, alias, patch
- **Yarn Classic** — `fixtures/locks/yarn/classic/`
- **Yarn Berry (node_modules)** — `fixtures/locks/yarn/berry-nm/`
- **Yarn Berry (PnP read-only)** — parse + identity; install rejected with typed error
- **bun** — `fixtures/locks/bun/`
- **Nub** — derived-format fixtures under `fixtures/locks/generated/nub-*` (not live Nub binary runs)

## Crash and transaction evidence

Crash integration covers install interruption, update interruption, snapshot
restore, and workspace snapshot paths. Windows runs in dedicated shards
(snapshot, install/txn, update) per
[`.github/workflows/ci.yml`](../.github/workflows/ci.yml).

See [`transaction.md`](transaction.md) and stabilization pass 20 scorecard
(`.agents/stabilization-pass20-score.md`) for patch-sandbox and provenance
evidence.

## Known limitations (pass 20 residual risks)

1. **Nub executable conformance** — derived-format fixture validation and
   Mew-native graph tests only; no frozen Nub binary differential matrix.
2. **Yarn Berry partial** — PnP install mode is read-only / rejected; full PnP
   linker parity is deferred post-0031.
3. **pnpm 11 patch config** — `patchedDependencies` may live in
   `pnpm-workspace.yaml`; fixtures retain dual `package.json#pnpm` fields for
   cross-major parity.
4. **Differential npm/pnpm CI** — full 0080 runtime conformance program not in
   0031 scope (`m conformance run runtime` deferred).
5. **Live Sigstore** — provenance verification uses fixture attestations; live
   registry Sigstore is deferred.
6. **Advisory feed signing** — `m audit` uses cached OSV bytes with digest
   only; cryptographic feed signature verification deferred.

## Schema freeze (0031)

The following are frozen for runner MVPs (**0040+**). Breaking changes require
an ADR and explicit migration:

- `m.lock` document shape (`lockfileVersion: 3`)
- Shipped PM CLI grammar documented in [`pm-commands.md`](pm-commands.md)
- Machine-readable report schemas listed in [`schema-freeze.md`](schema-freeze.md)

Install orchestration interfaces in `internal/app` (`Install`, `InstallOptions`,
`InstallResult`, transaction journal) are stable contracts for 0040; new fields
may be added in a backward-compatible way only.

## Open decisions

| Decision | Status |
|---|---|
| Core v1 beta channel promotion date | TBD — track in release train |
| Yarn Berry features deferred post-0031 | PnP install/link; Plug'n'Play runtime |
| `m conformance run runtime` scope | Deferred to MVP 0080 |

## Sign-off

Human checklist (0087-aligned, PM-core subset):
[`testdata/certification/sign-off-checklist.md`](../testdata/certification/sign-off-checklist.md).
