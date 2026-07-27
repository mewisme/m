# Contributing

## Prerequisites

- Go **1.26.5** or newer (see [`go.mod`](go.mod) and [`docs/engineering.md`](docs/engineering.md))
- Optional: GNU Make (or Git Bash `make` on Windows)
- Optional: pinned `golangci-lint` and `govulncheck` (see [`tools/versions.env`](tools/versions.env); install with [`tools/install.ps1`](tools/install.ps1))

## Exact commands

PowerShell (no Make required):

```powershell
gofmt -w <changed-go-files>
go test ./... -count=1
go vet ./...
golangci-lint run ./...
go build -o bin/m.exe ./cmd/m
go build -o bin/mx.exe ./cmd/mx
go run ./cmd/m development doctor
go run ./tools/check-license
go run ./tools/check-deps
```

With Make:

```text
make test
make vet
make lint
make race
make fuzz-smoke
make vuln
make build
make allowlist
```

Install pinned lint/vuln tools:

```powershell
./tools/install.ps1
```

```sh
./tools/install.sh
```

## Pull requests

- Keep changes inside the assigned MVP scope.
- Run `gofmt -w` on changed Go files, then `go test ./... -count=1`, `go vet ./...`, and `golangci-lint run ./...` before pushing.
- Update [`features/inventory.json`](features/inventory.json) when public behavior ships (see [`docs/features-maintenance.md`](docs/features-maintenance.md)).
- Resolving PRs must include `Closes #N`, `Fixes #N`, or `Resolves #N`.

## Agent orientation

Read [`AGENTS.md`](AGENTS.md) and [`docs/architecture/README.md`](docs/architecture/README.md) before product changes.
