# Mew

**Mew** is a Go implementation of the [Nub](https://nubjs.com) product model: a JavaScript toolchain and package manager built around stock Node augmentation.

| Binary | Role |
|---|---|
| **`m`** | Primary CLI — package management, scripts, runtime |
| **`mx`** | Package executable runner (Mewx) |

- New Mew projects use **`m.lock`** ([`docs/lockfile.md`](docs/lockfile.md)).
- Existing projects preserve their incumbent lockfile when Mew has a certified writer.
- **`nub.lock`** is a first-class compatibility target.

## Documentation

- [Program charter](docs/charter.md) — product contract and compatibility policy
- [Compatibility axes](docs/compatibility-axes.md) — parity / divergence / extension matrix
- [Stable naming](docs/naming.md) — binaries, config, env, error codes
- [Migration guide](docs/migration.md) — paths from npm, pnpm, Yarn, Bun, Nub
- [Feature inventory](docs/features-inventory.md) — capability matrix overview
- [Feature inventory maintenance](docs/features-maintenance.md) — how to update `features/inventory.json`
- [Architecture](docs/architecture/README.md) — package map, forbidden imports, boundaries
- [Engineering](docs/engineering.md) — Go floor, gates, fixture policy
- [Error codes](docs/errors.md) — ERR_M_* registry
- [Reporters](docs/reporters.md) — human / NDJSON / JSON diagnostics
- [Configuration](docs/config.md) — layers, keys, `m config`
- [Identity](docs/identity.md) — package-manager detection order
- [Data model](docs/data-model.md) — canonical graph, IDs, plan/snapshot
- [Testing](docs/testing.md) — fixtures, clean-home, fuzz, conformance
- [Release train](docs/release-train.md) — MVP order, channels, stop-the-line
- [CLI](docs/cli.md) — globals, version, completion, reserved stubs, dispatch
- [Manifest](docs/manifest.md) — package.json discovery, normalize, workspaces
- [Registry](docs/registry.md) — packument client, metadata cache, offline
- [Resolver](docs/resolver.md) — semver ranges, transitive graph, `m resolve`
- [Development doctor](docs/development-doctor.md) — `m development doctor` contract
- [Contributing](CONTRIBUTING.md) — exact test and build commands
- [Implementation plans](plans/0000-README.md) — ordered MVP program
- [AGENTS.md](AGENTS.md) — orientation for coding agents

## CLI

Commands are implemented with [Cobra](https://cobra.dev/) in `internal/cli`. Binaries are thin wrappers:

```powershell
go run ./cmd/m --help
go run ./cmd/m version
go run ./cmd/m features --format table
go run ./cmd/m features --format json --module runner
go run ./cmd/m development doctor
go run ./cmd/m config list --sources
go run ./cmd/mx --help
```

## Verify

```powershell
gofmt -w <changed-go-files>
go test ./... -count=1
go vet ./...
golangci-lint run ./...
go build -o bin/m.exe ./cmd/m
go build -o bin/mx.exe ./cmd/mx
```
