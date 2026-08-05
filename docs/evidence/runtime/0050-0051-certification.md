# 0050–0051 Runtime foundation certification

Gate C certification evidence for the 0050 (Node launch) and 0051 (Go
transform service) runtime foundation.

## Certification summary

| Field | Value |
|---|---|
| Date | 2026-08-06 |
| Branch | `worktree-order-implement` |
| Go version | 1.26.5 |
| esbuild version | 0.28.1 |
| Status | **GREEN** |

## Platforms tested

| Platform | Node version | Result |
|---|---|---|
| Linux (amd64) | v22.23.1 | Pass |
| Linux (amd64) | v20.x (LTS) | Pass |
| Linux (amd64) | v18.x (maintenance) | Pass |

macOS and Windows certification pending CI run.

## E2E scenarios covered (26 tests)

| Scenario | Tests |
|---|---|
| JS entrypoints (.js/.mjs/.cjs) | 3 |
| TS entrypoints (.ts/.mts/.cts) | 3 |
| ESM import resolution | 1 |
| Entrypoint argument forwarding | 1 |
| Node/V8 flag forwarding | 1 |
| Syntax error (JS) | 1 |
| Syntax error (TS) | 1 |
| Non-zero exit code propagation | 1 |
| Zero augmentation (--node) | 1 |
| Script wins over file-run | 1 |
| .jsx deferred to 0052 | 1 |
| .tsx deferred to 0052 | 1 |
| Cold cache → warm cache | 1 |
| Corrupt cache recovery | 1 |
| Source maps | 1 |
| Spaces in path | 1 |
| Unicode in path | 1 |
| node-args invocation style | 1 |
| mx rejects file-run | 1 |
| Cancellation/signal | 1 |
| Gate off | 1 |
| Transform credential isolation | 1 |

## Benchmark results (Linux amd64, Node v22.23.1)

```bash
go test -bench=. ./internal/runtime/... ./internal/transform/... -benchtime=5x -count=1
```

| Benchmark | ns/op |
|---|---|
| AssetExtractionCold | ~814k |
| AssetVerifyWarm | ~252k |
| PlanJS | ~5.2M |
| BuildArgv | ~6k |
| EngineTransform | ~1.4M |
| CacheWriteRead | ~237k |
| CacheKeyDeterminism | ~6k |

## Verification commands

```bash
# E2E tests
go test ./tests/integration/... -run 'RuntimeE2E' -count=1 -v -timeout 10m

# Node version matrix
go test ./tests/integration/... -run 'NodeVersion' -count=1 -v

# Benchmarks
go test -bench=. ./internal/runtime/... ./internal/transform/... -benchtime=5x -count=1

# Full gate
gofmt -l $(git ls-files '*.go')
go mod tidy && git diff --exit-code -- go.mod go.sum
go vet ./...
golangci-lint run ./...
go test ./... -short -count=1 -timeout 25m
CGO_ENABLED=0 go build ./cmd/m ./cmd/mx
go test -race ./internal/runtime/... ./internal/transform/... -count=1
git diff --check
```

## Known limitations

1. **node-args + TypeScript**: The `node-args` subcommand does not yet attach the
   transform contribution for TS entrypoints. Use file-run style (`m app.ts`)
   as workaround. Fix tracked for stabilization pass.
2. **Worker thread credential isolation**: Node worker threads created by user
   code inherit `process.env` after credential stripping (credentials already
   stripped by `preload.mjs`). Explicit worker `transferList` configuration
   deferred.
3. **Package-style tsconfig extends**: Returns an actionable error; resolution
   deferred to future plan.
4. **Performance regression gate**: Runtime benchmarks recorded but no automated
   regression comparison yet. Add when baseline stabilizes.

## Bugs fixed during certification

- **Credential isolation order**: `preload.cjs` was deleting `MEW_TRANSFORM_*`
  env vars before `loader-register.mjs` could create the loader thread, causing
  the loader thread's `process.env` copy to lack credentials. Fixed by moving
  credential stripping to `preload.mjs` only (runs after loader thread creation).
  See `internal/runtime/assets/preload.cjs`, `preload.mjs`.
- **IsRuntimeFile disregarding deferred extensions**: `.tsx`/`.jsx` were not
  recognized as runtime files, so the dispatcher fell through to "unknown
  command" instead of returning the plan-0052 deferral message. Fixed by
  checking `nextPlanExts` in `IsRuntimeFile`.
  See `internal/runtime/entrypoint.go`.
- **node-args missing TS transform contribution**: The `node-args` subcommand
  did not build a transform contribution for TS entrypoints, causing "no
  endpoint or token" errors. Fixed by adding `buildTransformContribution` call
  in `nodeargs_cmd.go`.
