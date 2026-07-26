# ADR 0002: Repository task runner

> **Status:** Accepted

## Context

MVP 0004 needs one documented way to run test, vet, lint, race, fuzz-smoke, and
vuln gates. Candidates were Makefile, just, and task.

## Decision

Use a root **`Makefile`** as the canonical task runner. Document equivalent
plain `go` / PowerShell commands in [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
for contributors without Make.

## Consequences

### Positive

- Matches the 0004 enrichment artifact list.
- Familiar to most Go repositories; easy to invoke from CI.

### Negative

- Windows contributors may need Git Bash or install `make`, or use the PowerShell one-liners instead.

### Neutral

- Tool versions remain pinned in `tools/versions.env`, not in Make macros alone.

## Alternatives considered

| Alternative | Why rejected |
|---|---|
| `just` | Extra install; not listed as primary in 0004 artifacts |
| `task` (go-task) | Extra install; weaker default availability |

## Compatibility impact

| Axis | Impact |
|---|---|
| CLI | none |
| Lockfile | none |
| Config | none |
| Runtime | none |
| Layout | none |

State: parity (process)

## Rollback

Add a thin `justfile` or Taskfile that shells out to the same commands; keep
Makefile until CI is switched.

## References

- Plan: `plans/0004-repository-bootstrap.md`
- File: `Makefile`
