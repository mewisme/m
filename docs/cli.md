# CLI foundation

`m` / `mew` (primary) and `mx` / `mewx` (package executor) share the Cobra shell in
[`internal/cli`](../internal/cli). Process state lives in [`internal/app`](../internal/app).

**MewJS** is the product name; **Mew** is the abbreviated form.

## Binaries and aliases

| Binary | Alias | Help / version label |
|---|---|---|
| `m` | `mew` | invoked basename (`m` or `mew`) |
| `mx` | `mewx` | invoked basename (`mx` or `mewx`) |

Installer-shipped `mew` / `mewx` symlinks are MVP **0072**. Until then, renaming or
symlinking the binary changes `Use` and `version` labels via basename detection.

## Global flags

| Flag | Default | Effect |
|---|---|---|
| `--cwd` | process cwd | Project discovery root; loads into `app.Context.CWD` |
| `--config` | | Extra JSONC overlay (CLI layer) |
| `--offline` | false | Force offline |
| `--prefer-offline` | false | Prefer cache |
| `--reporter` | env / `default` | `default` \| `ndjson` \| `json` \| `silent` |
| `--debug` | false | Verbose diagnostics |
| `--color` | `auto` | `auto` \| `always` \| `never` (TTY-aware) |
| `--no-color` | false | Force no ANSI |

## Version

```text
m version
m version --json
```

Text form: `m <version> (<commit>)` plus optional `built <date>` when ldflags set
`buildDate`. JSON object fields: `binary`, `version`, `commit`, `buildDate`.

Build metadata is injected at link time (`-X main.version=…` etc.). Dev default:
`0.0.0-dev`.

## Completion

```text
m completion bash|zsh|fish|powershell
mx completion bash|zsh|fish|powershell
```

Writes a script to stdout. Do not check generated scripts into the repo.

## Reserved names and stubs

Primary package-manager verbs are reserved so scripts cannot shadow them (script
fallback is MVP **0042**). Unimplemented verbs return `ERR_M_UNIMPLEMENTED`
(exit **1**) with the owning MVP id in the message.

Stubs on `m` today: `run`, `exec`, `init`, `link`.

Shipped built-ins (also reserved): `version`, `features`, `development`,
`config`, `project`, `pkg`, `cache`, `view`, `resolve`, `fetch`, `lock`,
`install` (`i`), `add`, `remove` (`rm`), `update`, `ci`, `outdated`, `dedupe`,
`prune`, `ls` (`list`), `store`, `doctor`, `audit`, `sbom`, `policy`, `verify`,
`completion`, `help`, hidden `__dispatch`.

Global flag: `--filter` — workspace package filter (pnpm-style), passed to
install-family commands. Requires [`workspaces.md`](workspaces.md) gate.

## Registry

```text
m view lodash [--json]
m cache dir
m cache verify [--json]
m cache metadata inspect lodash [--json]
```

See [`registry.md`](registry.md).

## Resolve

```text
m resolve [--plan] [--json] [--trace]
```

Dry dependency resolution without install. See [`resolver.md`](resolver.md).

## Explain

```text
m explain <name> [--json]
m explain peer <name> [--json]
```

Version selection and peer conflict trees. See [`explain.md`](explain.md).

## Plan

```text
m plan [--json] [--output <file>] [install flags]
m plan update [pkg...] [--json] [--output <file>]
```

Install mutation preview (same dry-run engine as `m install --dry-run`). See
[`plan.md`](plan.md).

## History and diff

```text
m history [--json]
m diff lock [--from <a> --to <b>] [other] [--json]
m lock diff [--from <a> --to <b>] [other] [--json]
```

`m history` lists install snapshots newest-first with delta summaries.
`m diff lock` / `m lock diff` compare lock graphs (human summary + optional JSON).
Snapshot restore: `m snapshot restore <id>` (see [`install.md`](install.md)).

`m shell --snapshot` / `m run --snapshot` — deferred to MVP **0045**.

## Fetch

```text
m fetch --plan-file plan.json [--dir dest] [--json]
```

Download, verify, and extract tarballs from a JSON plan. See [`fetch.md`](fetch.md).

## Doctor

```text
m doctor [--json] [--strict]
```

Project and package-manager health checks (lock, cache, store, filesystem probe,
transaction journals, config). See [`doctor.md`](doctor.md).

## Audit

```text
m audit [--json] [--fix]
```

Vulnerability scan against the cached OSV advisory database. `--fix` prints
suggested safe version bumps without writing manifests. See [`audit.md`](audit.md).

## SBOM

```text
m sbom [--format cyclonedx|spdx] [--redact-internal] [--redact-pattern <regex>]
```

Export CycloneDX or SPDX from the lock graph. See [`sbom.md`](sbom.md).

## Policy

```text
m policy check [--json]
```

Evaluate `mew.policy.json` / `.mew/policy.json` deny rules. Install-family
commands enforce error-severity violations in the validate phase. See
[`policy.md`](policy.md).

## Verify

```text
m verify provenance [<pkg>] [--attestation <path>]
```

Verify npm provenance attestation JSON against lock integrity. See
[`pack-publish.md`](pack-publish.md) (publish hook) and MVP **0030** provenance
fixtures.

## Capsule

```text
m capsule create [--output <path>]
m capsule restore <path>
```

Export or import a portable archive of lock, manifests, and cached blobs for
air-gapped bootstrap. See [`offline.md`](offline.md).

## Bench

```text
m bench install [--cold|--warm] [--fixture <path>] [--json]
```

End-to-end install benchmark with phase timing. See [`performance.md`](performance.md).

## Lock

```text
m lock format [--json]
m lock validate [--frozen] [--json]
```

Canonicalize or validate native `m.lock`. `--frozen` checks manifest specifier
drift (same check as `m install --frozen-lockfile`). See [`lockfile.md`](lockfile.md).

## Command precedence

1. Built-in command
2. Built-in alias
3. Exact `package.json` script (MVP **0042**, not yet)
4. Optional local executable lookup when the command contract enables it
5. Suggestion and error

Use `m run <script>` to force a script when a name collides with a built-in
(once **0040** / **0042** land).

## Hidden `__dispatch`

```text
m __dispatch <name>
```

Prints `kind=builtin|alias|unknown` and a resolved path. Prep for script
fallback; no script lookup yet.

## Install family

`m install`, `m add`, `m remove`, and `m ci` — see [`install.md`](install.md).

Maintenance and reporting commands — see [`pm-commands.md`](pm-commands.md):

```text
m outdated [-r] [--json]
m dedupe [--dry-run] [--json]
m prune [--prod] [--dry-run] [--json]
m ls [--depth N] [--prod] [--json]
```

Workspace options (gated — see [`workspaces.md`](workspaces.md)):

```text
m install -r
m install --filter <pattern>
m --filter <pattern> install
m add <pkg> --filter <pattern>
m ls -r [--depth N]
```

## Signals

`SIGINT` / `SIGTERM` cancel the root context (`signal.NotifyContext`).
Cancellation maps to `ERR_M_CANCELLED` (exit **130**).

## Process context

`PersistentPreRunE` builds `app.Context` (CWD, effective config, reporter, build
metadata) and stores it on the command context for subcommands.
