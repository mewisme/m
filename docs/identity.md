# Package-manager identity

Detection order matches [`AGENTS.md`](../AGENTS.md). Conflicting signals error;
Mew never silently picks a winner.

## Order

1. `package.json` → `packageManager` (parse `name@version`)
2. Else `devEngines.packageManager`
3. Else lockfile on disk:
   - `nub.lock` → `nub`
   - `m.lock` → `mew`
   - `pnpm-lock.yaml` → `pnpm`
   - `package-lock.json` / `npm-shrinkwrap.json` → `npm`
   - `yarn.lock` → `yarn`
   - `bun.lock` / `bun.lockb` → `bun`
4. Else **`mew`** (greenfield native)

## Conflicts

If a field identity (steps 1–2) disagrees with the sole recognized lockfile
identity (step 3), DetectIdentity returns `ERR_M_CONFIG` with subject `identity`.

Example: `packageManager: "npm@10"` with only `pnpm-lock.yaml` present → error.

When the field matches the lockfile, the field wins (same identity).

## Examples

| Signals | Result |
|---|---|
| `packageManager: pnpm@9` + `pnpm-lock.yaml` | `pnpm` |
| only `package-lock.json` | `npm` |
| only `nub.lock` | `nub` |
| `m.lock` / empty greenfield | `mew` |
| `packageManager: npm@10` + only `pnpm-lock.yaml` | error |

## Branded config authority

| Identity | May use branded config as authority |
|---|---|
| `mew` | No (`.npmrc` etc. ignored by `config.Load`) |
| `npm` / `pnpm` / `yarn` / `bun` / `nub` | Via future compat adapters only |

Explicit import/migration commands may read foreign files later; ordinary load does not.
