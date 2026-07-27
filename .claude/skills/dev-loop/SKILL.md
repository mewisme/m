---
name: dev-loop
description: Run the shortest reliable edit-build-test loop for Mew while preserving deterministic fixtures and avoiding global-state contamination. Use during ordinary Go development, focused package-manager work, runtime loader changes, or repeated integration-test iteration.
---

# Development loop

1. Identify the smallest owned package and focused test.
2. Run the focused test with `-count=1`.
3. Exercise the changed command against a temporary fixture.
4. `gofmt -w` changed Go files; `go vet ./affected/...`; `golangci-lint run ./...`.
5. Expand to dependent packages, then `go test ./...` before push.
6. Use the race detector for concurrent resolver, fetch, store, transaction, runner, or watcher code.

For embedded runtime assets, rebuild the embedding package and run both a direct loader fixture and the normal `m` entrypoint. For global-state behavior, set a temporary HOME and cache root.
