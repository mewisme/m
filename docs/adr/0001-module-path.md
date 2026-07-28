# ADR 0001: Go module path



> **Status:** Accepted (supersedes short-path deferral)



## Context



MVP 0004 required a finalized Go module path. The repository initially used

`module mew` with imports of the form `mew/internal/...`. A public GitHub path

is required for `go get` consumers and release publishing.



The module path was first published as `github.com/mewisme/m` and later renamed

to `github.com/mewisme/mew` to match the repository name. User data namespaces

under `github.com/mewisme/mew/...` were aligned in the same migration.



## Decision



Use **`module github.com/mewisme/mew`** as the module path. All imports use

`github.com/mewisme/mew/internal/...` (and `github.com/mewisme/mew/cmd/...` for

entrypoints).



## Consequences



### Positive



- Stable vanity path for external consumers and CI.

- Aligns with the `mewisme` GitHub organization and repository name.



### Negative



- One-time mechanical import rewrite across the tree.

- Existing installs with data under `github.com/mewisme/m/...` paths require

  manual relocation or re-install (no automatic migration).



### Neutral



- `go.mod` / `go.sum` remain the identity of the module.



## Alternatives considered



| Alternative | Why rejected |

|---|---|

| `module mew` (short path) | No public `go get` path; deferred too long |

| `github.com/mewjs/mew` (guessed org) | Org not confirmed |

| `github.com/mewisme/m` | Renamed to match repository; see migration commit |



## Compatibility impact



| Axis | Impact |

|---|---|

| CLI | none |

| Lockfile | none |

| Config | store/config path namespace changed (`m` → `mew` segment) |

| Runtime | none |

| Layout | none |



State: shipped (public path)



## Rollback



Rename `module` and all `github.com/mewisme/mew/` imports back to

`github.com/mewisme/m/` in one PR.



## References



- Plan: `plans/0004-repository-bootstrap.md`

- Module: `go.mod`

