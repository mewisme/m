# MewJS Program Charter

This document is the product contract for **MewJS** (abbreviated **Mew**): a Go-based JavaScript toolchain and package manager. It defines identity, compatibility policy, delivery priorities, and intentional extensions. Implementation plans live under `plans/`; this charter is the stable reference for later MVPs.

## Product identity

| Concept | Name | Notes |
|---|---|---|
| Product | **MewJS** | Public product name |
| Abbreviation | **Mew** | Short form in docs and messaging |
| Primary toolchain binary | **`m`** | Alias **`mew`** — package manager, scripts, runtime orchestration |
| Executable runner | **`mx`** | Alias **`mewx`** — local and temporary package binary execution |
| Native lockfile | **`m.lock`** | Default for new Mew-owned projects |
| Nub lockfile compatibility | **`nub.lock`** | First-class read/write target when project identity is Nub |
| Behavioral reference | **Nub** | Observable semantics and product positioning — not a source-level port target |

Nub is a **behavioral reference**. Mew preserves observable semantics where parity is intentional while using Go-native architecture, concurrency, storage, and error handling. Mew does not transliterate Rust crates or embed Nub/Aube as libraries.

### Version identity strings

```text
m <version> (<commit>)     # m version / mx version; optional "built <date>" line
{"binary","version","commit","buildDate"}  # m version --json
```

Invoked basename `mew` / `mewx` changes the help `Use` and version binary label; installer aliases ship in MVP 0072. See [`cli.md`](cli.md).

## Long-term goal

**Complete toolchain coverage** across package management, script execution, runtime augmentation, Node and package-manager management, and distribution — with Mew-specific improvements documented explicitly.

Delivery priority:

1. Package-manager core (install, resolve, lock, link, lifecycle)
2. Script and executable runners (`m run`, `mx`, direct shortcuts)
3. Runtime augmentation (TypeScript, loaders, watch, debugging)
4. Node and external package-manager management
5. Project initialization, plugins, and distribution

## Signature Mew improvements

Each differentiator has an owning MVP for traceability.

| Differentiator | Owning MVP(s) |
|---|---|
| Recoverable transactional installation | 0017 |
| Instant rollback, history, dependency time travel | 0017, 0028 |
| Explainable dependency resolution and mutation plans | 0020, 0028 |
| Semantic lockfile diff, validation, and cross-manager migration | 0023–0025, 0028 |
| Smart filesystem planner (hardlink, reflink, copy, symlink, junction) | 0018 |
| Capability-based lifecycle script trust and sandbox policy | 0021 |
| Portable verified dependency capsules | 0029 |
| Direct script shortcuts (`m dev`, `m start`, …) | 0042 (policy: 0001) |

## Compatibility policy

Compatibility is evaluated on **five independent axes**. See [`compatibility-axes.md`](compatibility-axes.md) for the full matrix and [`naming.md`](naming.md) for frozen identifiers.

### Compatibility states

Every feature or surface is classified as one of:

| State | Meaning |
|---|---|
| **parity** | Mew matches Nub or the incumbent manager's documented observable behavior |
| **intentional divergence** | Mew differs by design; the difference is documented with rationale |
| **extension** | Mew-only capability with no Nub equivalent |
| **deferred** | Planned but not yet implemented; must not be implied as shipped |

### Lockfile preservation

Existing projects **preserve their incumbent lockfile format** when Mew has a certified writer for that identity:

| Signal | Lockfile preserved |
|---|---|
| npm identity | `package-lock.json` / `npm-shrinkwrap.json` |
| pnpm identity | `pnpm-lock.yaml` |
| Yarn identity | `yarn.lock` |
| Bun identity | `bun.lock` |
| Nub identity | `nub.lock` |
| Mew identity (greenfield or explicit) | `m.lock` |

Identity detection order (unless a command overrides):

1. `package.json` → `packageManager`
2. `package.json` → `devEngines.packageManager`
3. Existing recognized lockfile
4. Mew native identity

Mew never silently drops a foreign lockfile into a project it does not own. Migration is always explicit (`m lock migrate`, `m pm use`, or equivalent).

### Direct script shortcuts (Mew extension)

Mew supports running exact `package.json` script names without `run`:

```sh
m dev
m start
m build -- --mode production
```

Dispatch precedence (reserved for MVP 0042):

1. Built-in command
2. Built-in alias
3. Exact `package.json` script name
4. Optional local executable lookup (when the command contract enables it)
5. Suggestion and error

Use `m run <script>` to force script execution when a name collides with a built-in.

## Supported platforms

| OS | Architectures | Status |
|---|---|---|
| Linux | amd64, arm64 | Supported target |
| macOS | amd64, arm64 | Supported target |
| Windows | amd64, arm64 | Supported target |

### Node version floor

The exact Node LTS floor for v1 is a **human-owned decision** tracked in plans 0084 and 0089. Until frozen, implementers should target **the current Node LTS release line** and document any syntax or API assumptions in the relevant runtime MVP.

### Filesystem assumptions

- Path length, case sensitivity, junctions, symlinks, and executable shims differ by platform; all install and link paths must be validated on Windows, not inferred from Unix-only CI.
- Global store and cache directories follow XDG on Linux, known locations on macOS and Windows (see `naming.md`).

## Experimental features and versioning

- Public behavior changes ship behind explicit **experimental gates** until stabilized.
- Experimental CLI flags use `--experimental-<name>`; environment toggles use `MEW_EXPERIMENTAL_<NAME>=1`.
- Stabilization removes the gate and updates [`compatibility-axes.md`](compatibility-axes.md) and the feature inventory (MVP 0002).
- Semantic versioning policy for CLI, lockfiles, caches, and error codes is defined in plan 0084.

## Architecture positions

### Go control plane

Orchestration, package management, filesystem work, networking, process control, diagnostics, policy, and compatibility adapters are implemented in **Go**. Embedded JavaScript is limited to Node loader, preload, worker, or runtime APIs that cannot execute Go directly.

### Runtime augmentation, not a Node fork

Mew runs the user's selected or installed **stock Node** binary and augments it through supported extension surfaces (preload, loader hooks, environment overlays, optional native addons). The compatibility test:

> Could the same behavior be implemented with stock Node and a supported loader, preload, addon, environment, or command-line surface?

If not, record an ADR before implementation.

### Atomic mutation

Every install-family mutation follows:

```text
inspect → resolve → plan → fetch → verify → stage → validate → commit
                                                      └→ rollback on failure
```

The previous manifest, lockfile, and `node_modules` remain usable until commit.

## Decision records

Irreversible choices (persistent formats, public config keys, compatibility promises, security posture) require an **Architecture Decision Record** before implementation. See [`adr/README.md`](adr/README.md) and [`adr/0000-template.md`](adr/0000-template.md).

## Migration

See [`migration.md`](migration.md) for incumbent package-manager migration narratives.

## MVP review

Later MVPs must verify against [`charter-checklist.md`](charter-checklist.md) before claiming completion.

## Open decisions

| Decision | Owner / tracker |
|---|---|
| Exact Node LTS floor for v1 | Plans 0084, 0089 |
| Whether v1 installers ship `mew` / `mewx` symlinks alongside `m` / `mx` | Plan 0072 |

## Related documents

- [`compatibility-axes.md`](compatibility-axes.md) — parity matrix by axis
- [`naming.md`](naming.md) — binaries, config, cache, env, error codes
- [`migration.md`](migration.md) — npm, pnpm, Yarn, Bun, Nub user paths
- [`charter-checklist.md`](charter-checklist.md) — MVP completion checklist
- [`../AGENTS.md`](../AGENTS.md) — agent orientation
- [`../plans/0001-program-charter.md`](../plans/0001-program-charter.md) — engineering contract for this MVP
