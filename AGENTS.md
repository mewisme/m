# AGENTS.md — agent orientation for the Mew repository

This is the canonical entrypoint for AI coding agents working on Mew. Claude Code also reads `CLAUDE.md`, which points back here. Codex must read `CODEX.md` after this file. A dispatched sub-agent must also read `L1_AGENTS.md` or `.github/subagent-guide.md` when its prompt says so.

An optional untracked `AGENTS.local.md` may add maintainer-local tooling, paths, or orchestration preferences. It must not weaken repository rules.

## Product contract (read first)

Before implementing product behavior, read:

- [`docs/charter.md`](docs/charter.md) — identity, compatibility policy, delivery priorities
- [`docs/compatibility-axes.md`](docs/compatibility-axes.md) — parity / divergence / extension / deferred matrix
- [`docs/naming.md`](docs/naming.md) — frozen binaries, lockfiles, config, env, error codes
- [`docs/architecture/README.md`](docs/architecture/README.md) — package map and import boundaries
- [`docs/engineering.md`](docs/engineering.md) — Go floor, quality gates, fixture policy
- [`docs/errors.md`](docs/errors.md) — ERR_M_* codes and exit mapping
- [`docs/reporters.md`](docs/reporters.md) — reporter formats and redaction
- [`docs/config.md`](docs/config.md) — layered configuration
- [`docs/identity.md`](docs/identity.md) — package-manager identity detection
- [`docs/data-model.md`](docs/data-model.md) — canonical graph and shared models
- [`docs/testing.md`](docs/testing.md) — fixtures, clean-home, fuzz, conformance
- [`docs/release-train.md`](docs/release-train.md) — MVP dependency graph and channels
- [`docs/cli.md`](docs/cli.md) — CLI globals, version, completion, reserved stubs
- [`docs/manifest.md`](docs/manifest.md) — package.json discovery and workspaces
- [`docs/registry.md`](docs/registry.md) — registry client and metadata cache
- [`docs/resolver.md`](docs/resolver.md) — semver, transitive graph, decision traces
- [`docs/lockfile.md`](docs/lockfile.md) — native `m.lock` format and frozen validation
- [`docs/adr/README.md`](docs/adr/README.md) — decision records for irreversible choices
- [`docs/features-maintenance.md`](docs/features-maintenance.md) — feature inventory update protocol
- [`features/inventory.json`](features/inventory.json) — machine-readable capability matrix
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — exact local commands

MVP completion must pass [`docs/charter-checklist.md`](docs/charter-checklist.md).

## Non-negotiables

- Keep every change inside the assigned scope and acceptance criteria.
- Never claim compatibility, parity, performance, or security without a fixture, command, benchmark, or source reference.
- Prefer root-cause fixes over warning-only or documentation-only mitigations when the defect is in Mew.
- Preserve existing lockfile identity unless the user explicitly requests migration.
- Never mutate `package.json`, a lockfile, or `node_modules` partially. Package-manager mutations must use the transaction boundary.
- Never weaken integrity verification, archive path validation, lifecycle-script policy, or rollback behavior to make a test pass.
- Do not add an exported Go package merely to avoid an internal design discussion. Default to `internal/`.
- Do not commit secrets, private URLs, local paths, benchmark framing strategy, or user conversation text.
- A resolving PR body must contain `Closes #N`, `Fixes #N`, or `Resolves #N`.
- Do not add automated-agent co-author trailers.

## Product identity

- **Mew** is the project name.
- **`m`** is the primary binary.
- **Mewx** is the package executable runner.
- **`mx`** is the executor binary.
- The native lockfile is **`m.lock`**.
- `nub.lock` is a first-class compatibility target.
- npm, pnpm, Bun, and Yarn projects keep their incumbent identity by default.

Public user-authored configuration should use neutral names when a neutral convention exists. Mew-specific files and commands may use the project name. Do not expose implementation-detail brands from reference projects.

## Architecture positions

### Go control plane

The shipped control plane is Go. Put orchestration, package management, filesystem work, networking, process control, diagnostics, policy, and compatibility adapters in Go.

CLI parsing and command dispatch use [Cobra](https://cobra.dev/) in `internal/cli`. Keep `cmd/m` and `cmd/mx` as thin entrypoints that call `cli.ExecuteM` / `cli.ExecuteMX`.

Use embedded JavaScript only where Node's loader, preload, worker, or runtime APIs require JavaScript. Keep the bridge narrow and versioned.

### Runtime augmentation, not a Node fork

Mew runs the user's selected or installed Node binary and augments it through supported extension surfaces such as preload modules, loader hooks, environment overlays, and optional native addons. Do not patch Node source or embed a private libnode.

The compatibility test is:

> Could the same behavior be implemented with stock Node and a supported loader, preload, addon, environment, or command-line surface?

If not, surface the architecture decision before implementation.

### Package-manager ownership

Mew owns its package-manager engine. Nub and Aube are behavioral references, not libraries to embed and not source trees to transliterate.

The core dependency direction is:

```text
cmd/m, cmd/mx
    -> internal/app and internal/cli
    -> manifest, project, workspace, config
    -> resolver and lockfile adapters
    -> registry, fetch, archive, store
    -> linker, lifecycle, policy
    -> transaction and diagnostics
```

Keep resolver decisions independent from disk mutation. Resolve a complete immutable graph before fetch/link/commit.

### Atomic mutation

Every install-family mutation follows:

```text
inspect -> resolve -> plan -> fetch -> verify -> stage -> validate -> commit
                                                        \-> rollback on failure
```

The old manifest, lockfile, and `node_modules` remain usable until commit. Journal enough information to recover from interruption.

### Lockfile neutrality

CLI grammar and lockfile compatibility are separate concerns. Detect identity in this order unless a documented command overrides it:

1. `packageManager`
2. `devEngines.packageManager`
3. existing recognized lockfile
4. Mew native identity

Adapters normalize into a shared graph but preserve format-specific information required for safe round trips. A lossy conversion must be explicit and reported.

### Direct script shortcuts

Mew intentionally supports package-script shortcuts:

```sh
m dev
m start
m build -- --mode production
```

Dispatch priority is:

1. built-in command
2. built-in alias
3. exact `package.json` script
4. optional local executable lookup only when the command contract enables it
5. suggestion and error

Use `m run <script>` to force script execution when a name collides with a built-in.

## Repository shape

Authoritative package map, forbidden imports, and boundary docs:
[`docs/architecture/`](docs/architecture/README.md).

Expected high-level layout:

```text
cmd/m/                     primary CLI (thin; calls internal/cli)
cmd/mx/                    package executor (thin; calls internal/cli)
internal/app/              process-level orchestration
internal/cli/              Cobra parsing, dispatch, help, completions
internal/config/           global and project configuration
internal/manifest/         package.json read/write
internal/project/          project discovery and identity
internal/workspace/        workspace graph and filters
internal/registry/         registry clients and auth
internal/semver/           npm-compatible range satisfaction
internal/resolver/         dependency and peer resolution
internal/lockfile/         normalized graph and adapters
internal/fetch/            downloads and retries
internal/archive/          safe extraction
internal/store/            content-addressed store
internal/linker/           isolated/hoisted linking
internal/transaction/      staging, journal, commit, rollback
internal/lifecycle/        dependency scripts
internal/policy/           trust and sandbox policy
internal/runner/           scripts, exec, dlx
internal/process/          signals, shells, child execution
internal/runtime/          Node augmentation orchestration
internal/transform/        TS/JSX transform service
internal/node/             Node discovery and provisioning
internal/pmmanager/        external PM provisioning
internal/compat/           Nub/npm/pnpm/Yarn/Bun compatibility
internal/testkit/          fixtures and integration helpers
internal/archcheck/        import-graph and package-map acceptance tests
internal/features/         feature inventory schema/runtime
runtime/                   embedded Node loader/preload assets
tests/                     conformance, integration, soak, benchmarks
```

## Engineering discipline

### Evidence before design

For compatibility work, create a minimal differential fixture and run Mew plus the claimed reference tool on identical inputs. Record versions and exact commands. Source reading supports the experiment; it does not replace it.

Treat each significant package-manager major as a separate compatibility target when behavior materially differs. Do not assume the latest version represents older projects.

### Quality over velocity

- Name what is actually implemented. A scaffold is not parity.
- Do not mark a task complete without end-to-end verification.
- Add a regression test for every durable bug fix when practical.
- Update user documentation in the same change as user-visible behavior.
- Avoid speculative abstractions. Add extension points after two concrete consumers or a committed near-term requirement.
- Preserve deterministic ordering in lockfiles, plans, diagnostics, and tests.

### Concurrency

Use contexts for cancellation and bounded worker pools for network or filesystem fan-out. Avoid unbounded goroutines. Define ownership for channels and shutdown. Run race tests on concurrency-sensitive packages.

### Errors

Use stable machine-readable error codes at the CLI boundary. Wrap errors with operation and subject. Do not leak credentials, auth headers, or signed URLs. Preserve exit-code contracts in tests.

## Verification

Run the smallest focused checks during iteration, then the complete applicable gate before pushing.

```sh
gofmt -w <changed-go-files>
go test ./path/to/touched/package -count=1
go test ./... -count=1
go vet ./...
go test -race ./... -count=1       # concurrency-sensitive changes
staticcheck ./...                   # when installed/configured
golangci-lint run                   # when configured
```

Also run the actual behavior in a temporary fixture. Package-manager changes require at least one clean-home test and one incumbent-lockfile test. Runtime changes require a Node-version matrix covering the supported floor and current stable. Windows behavior must be verified on Windows, not inferred from Wine or a Linux container.

Use Docker for clean Linux environments, libc variants, proxy/cache isolation, and Node-version probes. Keep containers ephemeral.

## Git and GitHub

- Work in an isolated worktree for substantive code changes.
- Stage only files owned by the current task.
- Keep commits small, factual, and neutral.
- Verify locally before pushing. CI is the backstop, not the development loop.
- Do not force-push the default branch.
- Do not merge a substantive PR without explicit authorization.
- Initial acknowledgement of an external issue is exactly `Investigating.`
- Keep substantive comments terse and factual; follow `PROSE.md`.

## Agent threads

Use `.agents/threads/<slug>.md` for durable task state when an effort spans turns or agents. Valid statuses are:

```text
planning
planned
enqueued
active
blocked
done
dismissed
```

A thread is current truth, not a chronological diary. Keep `Goal`, `Status`, `Decisions`, `Open questions`, `Steps`, and `Next step` current. The agent explicitly assigned the thread owns it. Other agents return findings instead of editing the same file.

## Human-owned decisions

Stop and surface a clear recommendation before committing a decision about:

- default behavior
- security or trust posture
- public API, config, environment, or file format
- compatibility promises
- irreversible migration
- release policy
- major architecture expansion

Mechanical implementation and clear bug fixes may proceed inside approved scope.
