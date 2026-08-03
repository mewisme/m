# Package-manager identity

A lockfile on disk is the authority. A `packageManager` declaration names the
manager a project intends to use, which matters only until a lock exists; it
does not override the lock that is actually there. Two lockfiles from different
managers is the one genuinely unresolvable case and errors.

## Order

1. Lockfile on disk (authoritative):
   - `nub.lock` → `nub`
   - `m.lock` → `mew`
   - `pnpm-lock.yaml` → `pnpm`
   - `package-lock.json` / `npm-shrinkwrap.json` → `npm` (shrinkwrap wins when both exist)
   - `yarn.lock` → `yarn`
   - `bun.lock` / `bun.lockb` → `bun`
2. Else `package.json` → `packageManager` (parse `name@version`)
3. Else `devEngines.packageManager`
4. Else **`mew`** (greenfield native)

The declaration is always recorded on `Project.Declared`, whether or not it won,
so callers can warn about a mismatch without changing which manager is used.

## Conflicts

Two recognized lockfiles with different identities cannot be resolved from
evidence: `DetectIdentity` returns `ERR_M_CONFIG` with subject `identity`.

A declaration that disagrees with the lockfile on disk is **not** an error. The
lockfile wins and the declaration is preserved as `Declared`. Erroring here
would strand a project that had already installed with another manager.

Identity detection only reads project signals: it creates no files and writes
nothing, inside or outside the project.

## Examples

| Signals | Result |
|---|---|
| `packageManager: pnpm@9` + `pnpm-lock.yaml` | `pnpm` |
| only `package-lock.json` | `npm` |
| only `npm-shrinkwrap.json` | `npm` |
| both shrinkwrap and package-lock | `npm` (incumbent = shrinkwrap) |
| only `nub.lock` | `nub` |
| `m.lock` / empty greenfield | `mew` |
| `packageManager: npm@10` + only `pnpm-lock.yaml` | `pnpm` (lock wins, `Declared: npm`) |
| `packageManager: yarn@4`, no lockfile | `yarn` (declaration decides) |
| `pnpm-lock.yaml` + `package-lock.json` | error |

## Branded config authority

| Identity | May use branded config as authority |
|---|---|
| `mew` | No (`.npmrc` etc. ignored by `config.Load`) |
| `npm` / `pnpm` / `yarn` / `bun` / `nub` | Via future compat adapters only |

Explicit import/migration commands may read foreign files later; ordinary load does not.
