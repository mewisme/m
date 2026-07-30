# CLI foundation

`m` / `mew` (primary) and `mx` / `mewx` (package executor) share the Cobra shell in
[`internal/cli`](../internal/cli). Process state lives in [`internal/app`](../internal/app).

**MewJS** is the product name; **Mew** is the abbreviated form.

## Binaries and aliases

| Binary | Alias | Help / version label |
|---|---|---|
| `m` | `mew` | invoked basename (`m` or `mew`) |
| `mx` | `mewx` | invoked basename (`mx` or `mewx`) |

Installer-shipped `mew` / `mewx` aliases are provided by the
[development installer](../CONTRIBUTING.md#development-installation) (`scripts/install-dev.*`).
Official release installers remain planned (MVP **0072**); for local development use
`make install-dev` or the platform script.

## Global flags

| Flag | Default | Effect |
|---|---|---|
| `--cwd` | process cwd | Project discovery root; loads into `app.Context.CWD` |
| `--config` | | Extra JSONC overlay (CLI layer) |
| `--offline` | false | Force offline |
| `--prefer-offline` | false | Prefer cache |
| `--reporter` | env / `default` | Legacy alias; see `--output` |
| `--output` | `auto` | `auto` \| `rich` \| `plain` \| `json` \| `ndjson` \| `silent` |
| `--progress` | `auto` | `auto` \| `always` \| `never` — live/plain install phase progress on stderr |
| `--unicode` | `auto` | `auto` \| `always` \| `never` |
| `--interactive` | `auto` | `auto` \| `always` \| `never` — prompts only when policy allows (stdin TTY, human mode; never in CI/`json`/`ndjson`/`silent` for `auto`) |
| `--log-level` | `error` | `error` \| `warn` \| `info` \| `debug` |
| `--accessible` | false | Accessible append-only output and numbered prompts |
| `--no-summary` | false | Suppress success summaries (never suppresses errors or security/lifecycle notices) |
| `--debug` | false | Verbose diagnostics |
| `--color` | `auto` | `auto` \| `always` \| `never` (TTY-aware; `--color=always` overrides `NO_COLOR`) |
| `--no-color` | false | Force no ANSI (overridden by `--color=always`) |
| `-r` / `--recursive` | false | Workspace recursive mode (`m run`; install-family commands have local `-r`) |
| `--filter` | | pnpm-style workspace package filter (install family and `m run`) |

Requires [`workspaces.md`](workspaces.md) gate (`MEW_EXPERIMENTAL_WORKSPACES=1` or
`workspaces.enabled`).

## Version

```text
m version
m version --json
```

Text form uses the presentation design system: a status line with the binary and
version, plus optional key-value rows for `commit` and `buildDate` when ldflags
set them. JSON object fields: `binary`, `version`, `commit`, `buildDate`.

Build metadata is injected at link time (`-X main.version=…` etc.). Dev default:
`0.0.0-dev`.

## Help

Root `m --help` lists commands in workflow groups (install, inspect, security,
cache/store, configuration). Per-command help may include `Examples` and `Related`
sections when registered in `internal/cli/help.go`. Completion scripts remain
plain text with no ANSI.

### Topic help

Long-form terminal topics are optional and embedded (no network fetch):

```text
m help <topic>
m help errors
m help errors ERR_M_LOCKFILE
m help runner
m help lifecycle-trust
```

If a name is both a command and a topic id/alias, **the command wins**
(`m help trust` shows the `trust` command; use `m help lifecycle-trust` for the
topic). Ordinary `m <command> --help` stays concise and does not open a pager.

Topic sources live under [`docs/terminal-help/`](terminal-help/) and are curated
separately from authoritative docs; each topic ends with a See also pointer.

Auto mode uses Glamour on a human color stdout TTY. Pipe, CI, `TERM=dumb`,
accessible, `--color=never`, and `--output=plain` stay on the plain renderer
(headings keep `#` markers; no ANSI). IDE terminals that report non-TTY still
get Glamour when color is forced:

```text
m --color=always help --pager=never runner
```

`--color=always` overrides `NO_COLOR` for help the same way it does for other
human output. Structured modes (`json` / `ndjson`) never use Glamour.
`--output=rich` also selects Glamour for topic help when rich mode is accepted
(interactive stderr); use `--color=always` when the terminal is non-TTY.

Glamour's standard style follows effective `ui.theme` (`auto`\|`light`\|`dark`;
`accessible`/`none` map to Glamour `notty`). `auto` uses the terminal background
hint (`COLORFGBG`). Theme selection still applies under `--color=always` /
ForceColor on non-TTY stdout.

### Topic pager

```text
m help --pager=auto|always|never <topic>
```

Precedence for the pager executable: `--pager` mode flag selects auto/always/never;
the command string comes from `MEW_PAGER` → `ui.pager` → `PAGER` → none
(Windows has no assumed `less`). Auto pages only on a human stdout TTY when
content is long enough and not in CI/accessible mode. Missing pager in auto
writes directly to stdout. Pager argv is split safely (no shell); content is
passed on stdin.

## Human errors

Typed CLI failures render through `ErrorView` (title, message, context, code,
hints) on stderr in human modes. `--presentation-legacy` keeps the pre-UX-0003
error format. JSON/NDJSON error documents are unchanged.

## Completion

```text
m completion bash|zsh|fish|powershell
mx completion bash|zsh|fish|powershell
```

Writes a script to stdout. Do not check generated scripts into the repo.

## Reserved names and stubs

Primary package-manager verbs are reserved so scripts cannot shadow them. Direct
`m <script>` shortcuts ship in MVP **0042** (gated). Unimplemented verbs return
`ERR_M_UNIMPLEMENTED` (exit **1**) with the owning MVP id in the message.

Stubs on `m` today: `init`, `link`.

Shipped: `run`, `exec` (MVP **0043**).

## `mx` DLX (MVP **0044**)

`mx` is enabled by default (no experimental gate). After reserved built-ins,
unrecognized argv is parsed as DLX:

```text
mx <package-spec> [child-args…]          # Mode A — local-first when unversioned
mx -p <spec>… <command> [child-args…]     # Mode B — explicit command
mx --yes <package-spec> …                 # non-interactive consent
mx --offline <package-spec> …            # offline warm cache only
mx cache prune [--older-than 7d] [--dry-run]
```

Reserved before DLX: `version`, `completion`, `cache`. Use `mx -p cache <bin>` for
a registry package named `cache`.

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

```text
m benchmark runner [--profile smoke|full] [--case <id>] [--json] [--output <path>] [--compare <baseline>] [--force]
```

Runner hot-path benchmark using local fixtures only. Default profile is `smoke`.
`--case` and `--profile` are mutually exclusive. `--compare` is informational and
never mutates the baseline. See [`runner-compatibility.md`](runner-compatibility.md).

## Conformance

```text
m conformance run core [--json] [--filter <suite-id>]
m conformance run runner [--json] [--output <path>] [--group <group>] [--filter <suite-id>] [--force]
m conformance verify runner --report <path>... --output <summary> [--force]
```

Runner certification uses [`tests/conformance/runner-matrix/manifest.json`](../tests/conformance/runner-matrix/manifest.json).
Cross-platform aggregation requires one report per platform. See
[`runner-compatibility.md`](runner-compatibility.md) and [`runner-waivers.md`](runner-waivers.md).

## Lock

```text
m lock format [--json]
m lock validate [--frozen] [--json]
```

Canonicalize or validate native `m.lock`. `--frozen` checks manifest specifier
drift (same check as `m install --frozen-lockfile`). See [`lockfile.md`](lockfile.md).

## Run

```text
m run <script> [--if-present] [-- <args>...]
m -r run <script> [--if-present] [-- <args>...]
m --filter <pattern>... run <script> [--if-present] [-- <args>...]
```

Run a `package.json` script with npm-compatible hooks, environment, and exit
codes. Regex selectors use `/pattern/` and run matching scripts in sorted name
order.

**Workspace orchestration** (MVP **0041**): set global `-r` / `--recursive` or one
or more global `--filter` patterns to run the script across selected workspace
members. Workspace-only flags (invalid without `-r` or `--filter`):

| Flag | Default |
|---|---|
| `--workspace-concurrency N` | `GOMAXPROCS` (`0` = `GOMAXPROCS`, capped by task count) |
| `--workspace-order` | `topological` |
| `--workspace-output` | `stream` |
| `--workspace-bail` | `true` (`--no-workspace-bail` to continue after failures) |

See [`runner.md`](runner.md).

Human mode may print a short stderr prep banner and optional completion summary
(`--no-summary` suppresses the summary). Workspace human status is append-only
on stderr; child streams stay on stdout/stderr with `[pkg]` prefixes in stream
mode.

## Exec

```text
m exec <binary> [--package <dependency>] [-- <args>...]
m exec --snapshot <id> <binary> [-- <args>...]
m exec --capsule <path> <binary> [-- <args>...]
m --cwd <importer> exec <binary>
m --filter <pattern> exec <binary>   # exactly one workspace member
```

Source flags (`--snapshot`, `--capsule`, `--package`) are parsed **before** the
binary selector. Tokens after the selector belong to the child process.

## Env inspect

```text
m env inspect project [--project-dir <dir>] [--package <owner>] [<binary>]
m env inspect dlx -p <package>... <binary>
m env inspect snapshot <id> [<binary>]
m env inspect capsule <path> [<binary>]
```

Plan-only: never executes, never materializes cold shared-cache environments,
never acquires execution leases, and never contacts the registry for snapshot or
capsule sources. JSON output uses schema v1 (`"v": 1`).

Execute a **local** package binary from the current importer context. Network-free;
never uses the Mew registry client.

| Rule | Behavior |
|---|---|
| Importer cardinality | Exactly one importer per invocation |
| `-r` / `--recursive` | `ERR_M_USAGE` — never workspace-orchestrate bins |
| `--filter` | Allowed only when it resolves to **exactly one** member |
| `--package` | Importer-visible dependency or alias providing the bin (not a workspace selector) |
| Miss | Suggests `m exec --package <dependency> <bin>`; never guesses package from command name |

**Direct bin dispatch** (step 4, gated separately from scripts):

```text
m eslint
m --cwd packages/api eslint
```

**Gate:** `runner.exec.direct_dispatch.enabled` or
`MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH=1`. Requires verified bin metadata
(`OwnershipVerified=true`); unowned shims fall through to suggestions or require
explicit `m exec`.

Script workspace orchestration (`m -r dev`) is unchanged from MVP **0041**; bin
dispatch is always single-importer.

See [`runner.md`](runner.md#local-binary-execution-m-exec).

## Command precedence

1. Built-in command
2. Built-in alias
3. Exact `package.json` script when `runner.direct_scripts.enabled` or
   `MEW_EXPERIMENTAL_DIRECT_SCRIPTS=1` (case-sensitive key only)
4. Verified local binary when `runner.exec.direct_dispatch.enabled` or
   `MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH=1` (**0043**; metadata-verified only)
5. Typed suggestions (max 3) and `ERR_M_USAGE`

Use `m run <script>` to force a script when a name collides with a built-in.

### Direct shortcuts (Mew extension, gated)

```text
m dev
m build --mode production
m --cwd ./app build --mode production
m -r build
m --filter api... test
```

**Gate:** `runner.direct_scripts.enabled` or `MEW_EXPERIMENTAL_DIRECT_SCRIPTS=1`.

**Workspace direct shortcuts** also require `workspaces.enabled` or
`MEW_EXPERIMENTAL_WORKSPACES=1`.

**Argument rules:**

- Root globals (`--cwd`, `--reporter`, `-r`, `--filter`, workspace flags before
  the selector) are parsed only **before** the script name.
- Tokens after the selector are forwarded verbatim to the script child.
- `m dev --reporter ndjson` passes `--reporter` to the script, not Mew.
- One `--` after the selector is stripped; following tokens forward verbatim.

**Bare `m`:** always `ERR_M_USAGE` (exit 2); lists up to 10 script names when a
valid manifest exists; never executes or prompts.

## Hidden `__dispatch`

```text
m __dispatch <name>
```

Side-effect-free JSON introspection (`schemaVersion: 1`). Same resolution logic as
dispatch; never executes scripts. Internal diagnostic only.

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
