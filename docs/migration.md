# Migration Narrative Outline

How users move between incumbent package managers and Mew. Full commands ship in later MVPs; this document defines the **policy** those commands must follow.

## Principles

1. **No silent migration** — lockfile or identity changes require an explicit user action.
2. **Preserve round-trip** — when Mew writes an incumbent format, the result must pass that manager's frozen install where certified.
3. **Graph equality** — no-churn installs leave a semantically equal lockfile untouched.
4. **One lockfile authority** — multiple lockfiles without declaration is an error (parity with Nub ambiguity rules).

## npm users

| Starting state | Mew behavior | Explicit migration |
|---|---|---|
| `package-lock.json` only | Operate as npm identity; preserve `package-lock.json` | `m pm use m` → convert to `m.lock` |
| `packageManager: npm@…` | npm identity from manifest | `m lock migrate --from npm --to m` |
| Greenfield | Mew identity; create `m.lock` on first install | — |

## pnpm users

| Starting state | Mew behavior | Explicit migration |
|---|---|---|
| `pnpm-lock.yaml` only | Operate as pnpm identity; preserve lockfile | `m pm use m` or `m lock migrate` |
| pnpm workspace config | Read via compatibility adapter | Neutral `package.json` fields on Nub-style switch |
| `packageManager: pnpm@…` | pnpm identity from manifest | — |

## Yarn users

| Starting state | Mew behavior | Explicit migration |
|---|---|---|
| `yarn.lock` only | Yarn identity; read support first (Berry/PnP staged in 0025) | Explicit migrate when write certified |
| Berry / PnP | Read path in 0025; write requires conformance gate | `m lock migrate` when available |

## Bun users

| Starting state | Mew behavior | Explicit migration |
|---|---|---|
| `bun.lock` (text) | Bun identity; preserve on install | `m lock migrate` |
| `bun.lockb` (binary) | Rejected — convert to text `bun.lock` first | User converts with Bun, then Mew |

## Nub users

| Starting state | Mew behavior | Explicit migration |
|---|---|---|
| `nub.lock` only | Nub identity; `nub.lock` is pnpm v9–compatible bytes | Preserve format on install (0023) |
| `packageManager: nub@…` | Nub identity from manifest | — |
| Full Nub parity target | Behavioral reference for CLI, lock, runtime | See [`compatibility-axes.md`](compatibility-axes.md) |

`nub.lock` beside a foreign lockfile without `packageManager` declaration is ambiguous — Mew errors with `ERR_M_LOCKFILE_AMBIGUOUS` (parity intent).

## Greenfield Mew projects

Empty directory or `m init` (0070):

- No lockfile → first `m install` creates **`m.lock`**
- `packageManager` may be set to `m@<version>` on explicit `m pm use m`

See fixture `fixtures/charter/empty` for greenfield wording tests.

## Fixture references

| Fixture | Purpose |
|---|---|
| `fixtures/charter/npm-app` | npm lockfile preservation wording |
| `fixtures/charter/pnpm-app` | pnpm lockfile preservation wording |
| `fixtures/charter/nub-app` | nub.lock preservation wording |
| `fixtures/charter/empty` | greenfield `m.lock` default |
