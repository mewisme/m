# Mew benchmarks

Reproducible package-manager benchmarks for transaction lock, resolver, linker,
and store paths (stabilization pass 2).

## Environment

Record these values with every published result:

| Field | Command |
|---|---|
| OS / arch | `go env GOOS GOARCH` |
| Go version | `go version` |
| CPU model | optional but helpful on laptops |
| CGO | `go env CGO_ENABLED` (required for `-race`) |
| Cold vs warm | see below |

Template:

```text
date: 2026-07-27
goos/goarch: windows/amd64
go: go1.24.x
cgo: 0
notes: warm registry fixture cache
```

## Full gate (copy into CI notes)

```powershell
go test ./... -count=1
go vet ./...
golangci-lint run ./...
govulncheck ./...
```

Race (Linux/macOS or Windows with CGO):

```powershell
$env:CGO_ENABLED = "1"
go test -race ./internal/transaction/... ./internal/store/... ./internal/resolver/... -count=1
```

## Run all package-manager benches

From the repository root:

```powershell
go test -bench=. -benchmem -count=5 `
  ./internal/transaction/... `
  ./internal/resolver/... `
  ./internal/linker/hoisted/... `
  ./internal/store/... `
  | Tee-Object -FilePath bench.out
```

Race-sensitive packages are excluded from bench runs; use unit and cross-process
tests for concurrency.

## Focused suites

### Transaction (0017)

```powershell
go test -bench=BenchmarkProjectLockContention -benchmem -count=5 ./internal/transaction/...
go test -bench=BenchmarkTransaction -benchmem -count=5 ./internal/transaction/...
```

### Resolver (0020)

```powershell
go test -bench=BenchmarkResolveTransitive -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkPeerContextResolution -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkTargetedIncrementalUpdate -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkLargeGraphResolve -benchmem -count=5 ./internal/resolver/...
```

### Store (0018)

```powershell
go test -bench=BenchmarkStoreImport -benchmem -count=1 ./internal/store/...
go test -bench=BenchmarkStoreImportContention -benchmem -count=5 ./internal/store/...
go test -bench=BenchmarkStoreVerifyWarm -benchmem -count=5 ./internal/store/...
go test -bench=BenchmarkStoreFullTreeVerify -benchmem -count=5 ./internal/store/...
```

### Linker (0016/0019)

```powershell
go test -bench=BenchmarkHoistedPlan -benchmem -count=5 ./internal/linker/hoisted/...
```

## Cold vs warm

**Cold** — first import / first resolve (no store reuse):

```powershell
go test -bench=BenchmarkResolveTransitive -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkStoreImport -benchmem -count=1 ./internal/store/...
```

**Warm** — cached registry + verified store:

```powershell
go test -bench=BenchmarkResolveWorkspaceProtocol -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkStoreVerifyWarm -benchmem -count=5 ./internal/store/...
go test -bench=BenchmarkHoistedPlan -benchmem -count=5 ./internal/linker/hoisted/...
```

## Interpreting results

- `ns/op` — wall time per iteration
- `B/op` — bytes allocated per iteration
- `allocs/op` — heap allocations per iteration

Compare multiple `-count` samples; report median. `BenchmarkLargeGraphResolve`
uses an in-memory synthetic graph; a 1k+ workspace fixture corpus is deferred.

## Sample baseline (windows/amd64, local run 2026-07-27)

| Benchmark | Median ns/op | allocs/op | Notes |
|---|---|---|---|
| BenchmarkProjectLockContention | run locally | — | uncontended acquire/release |
| BenchmarkTransactionCommit | run locally | — | journal + staged publish |
| BenchmarkResolveTransitive | run locally | — | registry fixture pkg-a chain |
| BenchmarkPeerContextResolution | run locally | — | react ecosystem fixture |
| BenchmarkTargetedIncrementalUpdate | run locally | — | lodash unchanged subgraph |
| BenchmarkStoreVerifyWarm | run locally | — | lodash tree manifest walk |
| BenchmarkStoreFullTreeVerify | run locally | — | bidirectional manifest verify |
| BenchmarkHoistedPlan | run locally | — | three-package hoisted graph |

Fill medians after `go test -bench=... -count=5` on your machine; do not commit
machine-specific numbers to golden files.

## End-to-end install bench (`m bench install`)

MVP **0029** adds a CLI harness that runs a full install against
`fixtures/bench/medium-graph` (or `--fixture`) with isolated cache home:

```powershell
go run ./cmd/m bench install --cold --json
go run ./cmd/m bench install --warm --json
```

Published medians: [`benchmarks/install-baseline.json`](../../benchmarks/install-baseline.json).

Regression gate (10% over median fails unless `BENCH_WAIVER=1`):

```text
python tools/bench/check_regression.py --mode warm
```

Soak loop: `python tools/soak/install_loop.py --count 10 --mode cold`.

See [`docs/performance.md`](../../docs/performance.md) for phase timing,
worker defaults, and hot-path profiling notes.
