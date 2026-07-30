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
make core-cert
make core-cert-fast
make core-cert-security
make core-cert-crash
make core-cert-performance
```

Core certification steps are defined in
[`tools/certification/core-manifest.json`](tools/certification/core-manifest.json)
and executed by `tools/certification/run_core_cert.py` (or the Make targets above).
Race tests and the full `-tags crash` integration suite are expensive and remain
separate (`make race`; see [`docs/core-certification.md`](docs/core-certification.md)).

Install pinned lint/vuln tools:

```powershell
./tools/install.ps1
```

```sh
./tools/install.sh
```

## Development installation

The repo ships a **development installer** — not an official release installer. It builds
`m` / `mx` from source, copies them into a user-local directory, updates PATH, and optionally
installs shell completions. No admin or root access is required.

### One-liner

Windows (PowerShell 7+):

```powershell
pwsh -NoProfile -File scripts/install-dev.ps1
```

Linux / macOS:

```sh
./scripts/install-dev.sh
```

With GNU Make (Git Bash on Windows uses the PowerShell script):

```text
make install-dev
make uninstall-dev
```

### Default locations

| Platform | Install directory | Completion root |
|---|---|---|
| Windows | `%LOCALAPPDATA%\MewJS\bin` | `%LOCALAPPDATA%\MewJS\completions` |
| Linux / macOS | `$XDG_DATA_HOME/mewjs/bin` (or `~/.local/share/mewjs/bin`) | sibling `completions/` |

### Binaries and aliases

| Binary | Alias | Windows shim |
|---|---|---|
| `m` | `mew` | `m.cmd`, `mew.cmd` |
| `mx` | `mewx` | `mx.cmd`, `mewx.cmd` |

Unix installs use relative symlinks (`mew -> m`, `mewx -> mx`).

### Supported targets

Host and cross-build targets: `windows`, `linux`, `darwin` × `amd64`, `arm64`.

Cross-compiled builds require `--build-only` / `-BuildOnly` (install is refused on a
non-matching host).

### Flags

| Flag | Effect |
|---|---|
| `--build-only` / `-BuildOnly` | Build into `bin/` only; skip install, PATH, completion, verify |
| `--skip-path` / `-SkipPath` | Skip PATH / profile updates |
| `--skip-completion` / `-SkipCompletion` | Skip completion generation |
| `--skip-verify` / `-SkipVerify` | Skip post-install checks |
| `--install-dir` / `-InstallDir` | Custom install directory |
| `--goos`, `--goarch` / `-GoOS`, `-GoArch` | Cross-compile target |
| `--version` / `-Version` | Override `main.version` ldflag |
| `--force` / `-Force` | Replace conflicting Unix symlinks |

Uninstall: `scripts/uninstall-dev.ps1` or `scripts/uninstall-dev.sh` with optional
`--keep-path` / `--keep-completion`.

### PATH and completions

- **Windows:** user-scope `Path` via `[Environment]::SetEnvironmentVariable` (never `setx`).
- **Unix:** managed block in the shell profile (`# >>> mewjs dev installer >>>`).
- **Completions:** bash, zsh, fish, PowerShell via `m completion <shell>`; profile block
  `# >>> mewjs dev completion >>>`. Alias commands (`mew`, `mewx`) get installer-managed
  wrapper registrations.

Restart terminals after install so PATH and completion changes load.

### Examples

```sh
./scripts/install-dev.sh --build-only
./scripts/install-dev.sh --goos linux --goarch arm64 --build-only
./scripts/install-dev.sh --skip-completion --install-dir "$HOME/.local/mewjs-bin"
./scripts/uninstall-dev.sh
```

Logic tests: `bash scripts/lib/devinstall_test.sh`, `pwsh -NoProfile -File scripts/lib/devinstall_test.ps1`.

## Pull requests

- Keep changes inside the assigned MVP scope.
- Run `gofmt -w` on changed Go files, then `go test ./... -count=1`, `go vet ./...`, and `golangci-lint run ./...` before pushing.
- Update [`features/inventory.json`](features/inventory.json) when public behavior ships (see [`docs/features-maintenance.md`](docs/features-maintenance.md)).
- CLI UX certification entry: `go run ./cmd/m conformance run cli-ux --json` (see [`docs/testing.md`](docs/testing.md) and [`docs/evidence/cli-ux/`](docs/evidence/cli-ux/)).
- Resolving PRs must include `Closes #N`, `Fixes #N`, or `Resolves #N`.

## Agent orientation

Read [`AGENTS.md`](AGENTS.md) and [`docs/architecture/README.md`](docs/architecture/README.md) before product changes.
