# Forbidden import edges

These rules keep presentation, resolution, and mutation layers separate.
Enforced by [`internal/archcheck`](../../internal/archcheck/) tests.

## Rules

| From | Must not import | Rationale |
|---|---|---|
| `cmd/m`, `cmd/mx` | any `github.com/mewisme/mew/internal/...` except `app` and `cli` | Thin entrypoints only |
| `internal/cli` | `linker`, `store`, `fetch` | CLI must not own mutation; may call registry/resolver |
| `internal/resolver` | `linker`, `transaction`, `runner`, `fetch`, `store` | Resolve completes before mutate |
| `internal/apperr`, `internal/diagnostics`, `internal/trace` | `registry`, `fetch`, `linker` | Diagnostics stay free of PM engine |
| `internal/app`, `internal/runner`, `internal/transaction`, `internal/resolver`, `internal/linker`, `internal/store`, `internal/lifecycle` | `internal/presentation`, `charm.land/*`, `github.com/charmbracelet/*` | Domain stays free of presentation and Charm |
| `internal/config`, `internal/project` | `resolver`, `linker`, `fetch` | Config/identity stay free of mutate path |
| `internal/graph`, `plan`, `snapshot`, `manifest`, `policy`, `capsule` | `fetch`, `linker`, `registry` | Canonical models stay free of network/mutate |

## Allowed cmd imports

`cmd/m` and `cmd/mx` may import only:

- `github.com/mewisme/mew/internal/app`
- `github.com/mewisme/mew/internal/cli`
- Go standard library
- Module root packages that are not under `internal/` (none today)

## Adapter rule

Packages under `internal/compat` and `internal/lockfile/*` convert to and from
canonical types. They must not own filesystem mutation. Mutation commits only
through `internal/transaction`.

## Resolve-complete-before-mutate

`internal/resolver` produces an immutable resolution result. Downstream packages
may consume that result but the resolver must not reach into fetch, store, or
linker packages. See [transaction-boundary.md](transaction-boundary.md).
