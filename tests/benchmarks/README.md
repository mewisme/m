# Mew benchmarks

Reproducible package-manager benchmarks for resolver, linker, and store paths.

## Environment

Record these values with every published result:

- OS and version (`go env GOOS GOARCH`)
- Go version (`go version`)
- CPU model (optional but helpful on laptops)
- Cold vs warm (see below)

## Run

From the repository root:

```powershell
go test -bench=. -benchmem -count=5 ./internal/resolver/... ./internal/linker/hoisted/... ./internal/store/... | Tee-Object -FilePath bench.out
```

Race-sensitive packages are excluded from bench runs; use unit tests for concurrency.

### Cold (first import / first resolve)

```powershell
go test -bench=BenchmarkResolveTransitive -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkStoreImport -benchmem -count=1 ./internal/store/...
```

### Warm (cached registry + verified store)

```powershell
go test -bench=BenchmarkResolveWorkspaceProtocol -benchmem -count=5 ./internal/resolver/...
go test -bench=BenchmarkStoreVerifyWarm -benchmem -count=5 ./internal/store/...
go test -bench=BenchmarkHoistedPlan -benchmem -count=5 ./internal/linker/hoisted/...
```

## Interpreting results

- `ns/op` — wall time per iteration
- `B/op` — bytes allocated per iteration
- `allocs/op` — heap allocations per iteration

Compare multiple `-count` samples; report median. Large fixture corpora land in a follow-up when `fixtures/projects` gains a 1k+ workspace graph.

## Sample baseline (fill after local run)

| Benchmark | OS | Go | Median ns/op | Notes |
|-----------|----|----|--------------|-------|
| BenchmarkResolveTransitive | windows/amd64 | 1.24.x | _run locally_ | registry fixture pkg-a chain |
| BenchmarkStoreVerifyWarm | windows/amd64 | 1.24.x | _run locally_ | lodash tree manifest walk |
| BenchmarkHoistedPlan | windows/amd64 | 1.24.x | _run locally_ | three-package hoisted graph |
