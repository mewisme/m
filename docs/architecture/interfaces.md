# Core interfaces

These interfaces are frozen as contracts for later MVPs. Canonical value types
live in [`docs/data-model.md`](../data-model.md) (MVP 0007).

All methods accept `context.Context`, avoid global mutable state, and must
return typed errors carrying stable Mew error codes (MVP 0005).

## Interface owners

| Interface | Package | Methods (sketch) |
|---|---|---|
| `Registry` | `internal/registry` | `Metadata(ctx, name, version)` → `PackageMetadata`; also `Client.Packument` |
| `Resolver` | `internal/resolver` | `Resolve(ctx, root, opts) (*Resolution, error)` — `Resolution` holds `*graph.Graph` + decisions; `ResolveOptions.Hints` is an optional prior `*graph.Graph` for pin reuse |
| `Store` | `internal/store` | `Get(ctx, key)`, `Put(ctx, key, content)` |
| `Linker` | `internal/linker` | `Plan(ctx, *graph.Graph)`, `Apply(ctx, *linker.Plan)` |
| `LockfileAdapter` | `internal/lockfile` | `Read(ctx, path)`, `Write(ctx, path, *graph.Graph)` |
| `Transaction` | `internal/transaction` | `Begin`, `Stage`, `Commit`, `Rollback` |
| `ScriptRunner` | `internal/runner` | `Run(ctx, script, env)` |
| `ProcessSupervisor` | `internal/process` | `Start(ctx, spec)`, `Wait(ctx)` |

Compile-time fakes in each package's `*_test.go` prove interfaces are
independently mockable.

## Plan naming

| Type | Package | Role |
|---|---|---|
| `linker.Plan` | `internal/linker` | Filesystem link operations |
| `plan.Plan` | `internal/plan` | Install mutation plan (desired / ops / commits) |

## Immutability and copy-on-write

| Boundary | Rule |
|---|---|
| Resolution result | Immutable after `Resolve` returns; consumers must not mutate shared graphs |
| Lockfile graph | Adapters produce canonical values; edits are copy-on-write |
| Install plan | Desired state is fixed before stage; stage applies a separate operation list |
| Store entries | Content-addressed; never overwrite an existing key with different bytes |
| Transaction | Staged trees are private until commit; commit swaps roots atomically |

## Extension points (no public plugin ABI)

- External `m-<verb>` executables discovered on PATH (`internal/plugin`, planned).
- Lockfile and PM adapters under `internal/compat` and `internal/lockfile/*`.
- Config layers in `internal/config`.
- Adapter-owned `lockfile.Extensions` and `LossReport` for format fidelity.

Do not load untrusted Go plugins into the Mew process. Extension is out-of-process
or adapter registration owned by Mew packages.

## Decisions that block later MVPs

1. Flat packages (no `internal/pm` umbrella) — 0004+ skeleton must match this map.
2. Resolve-complete-before-mutate — 0013–0017 must not stream mutate during resolve.
3. Stock Node only — 0050–0057 must not introduce libnode.
4. Transaction owns commit — 0016–0017 must not bypass `internal/transaction`.
5. Shared `graph.Graph` — lockfile, linker, and resolver must not redefine graph types.
