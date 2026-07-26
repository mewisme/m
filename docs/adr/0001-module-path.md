# ADR 0001: Go module path

> **Status:** Accepted (supersedes short-path deferral)

## Context

MVP 0004 required a finalized Go module path. The repository initially used
`module mew` with imports of the form `mew/internal/...`. A public GitHub path
is required for `go get` consumers and release publishing.

## Decision

Use **`module github.com/mewisme/m`** as the module path. All imports use
`github.com/mewisme/m/internal/...` (and `github.com/mewisme/m/cmd/...` for
entrypoints).

## Consequences

### Positive

- Stable vanity path for external consumers and CI.
- Aligns with the `mewisme` GitHub organization.

### Negative

- One-time mechanical import rewrite across the tree.

### Neutral

- `go.mod` / `go.sum` remain the identity of the module.

## Alternatives considered

| Alternative | Why rejected |
|---|---|
| `module mew` (short path) | No public `go get` path; deferred too long |
| `github.com/mewjs/mew` (guessed org) | Org not confirmed |

## Compatibility impact

| Axis | Impact |
|---|---|
| CLI | none |
| Lockfile | none |
| Config | none |
| Runtime | none |
| Layout | none |

State: shipped (public path)

## Rollback

Rename `module` and all `github.com/mewisme/m/` imports back to `mew/` in one PR.

## References

- Plan: `plans/0004-repository-bootstrap.md`
- Module: `go.mod`
