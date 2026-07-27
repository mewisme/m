---
name: go-build
description: Build, test, lint, race-check, and cross-compile the Mew Go repository efficiently and correctly, especially inside concurrent Git worktrees. Use for Go build failures, cache questions, CI-gate reproduction, race testing, or cross-platform binary verification.
---

# Go build workflow

Go's build and module caches are content-addressed and safe to share across worktrees. Do not copy Cargo target-directory rules into Mew.

## Fast loop

```sh
gofmt -w <changed-go-files>
go test ./path/to/package -count=1
go test ./path/to/package -run TestName -count=1
```

## Repository gate

```sh
go test ./... -count=1
go vet ./...
golangci-lint run ./...
go test -race ./... -count=1
```

Install pinned tools: `./tools/install.ps1` or `./tools/install.sh`. Pin and config: `tools/versions.env`, `.golangci.yml`.

**errcheck:** `defer func() { _ = f.Close() }()` for `Close`/`RemoveAll`; reporter `fmt.Print*` is excluded in `.golangci.yml`.

## Cross-build

```sh
GOOS=linux GOARCH=amd64 go build ./cmd/m ./cmd/mx
GOOS=darwin GOARCH=arm64 go build ./cmd/m ./cmd/mx
GOOS=windows GOARCH=amd64 go build ./cmd/m ./cmd/mx
```

Cross-building verifies compilation, not behavior. Run Windows-specific tests on Windows.

Use isolated `HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and store paths for integration tests. Keep `GOCACHE` and `GOMODCACHE` shared unless diagnosing cache corruption.
