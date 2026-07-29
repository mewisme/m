# Install plan preview

`m plan` previews install-family mutations without writing `package.json`,
`m.lock`, or `node_modules` (MVP **0028**). It is the dedicated UX for the same
dry-run path as `m install --dry-run`.

## Commands

```text
m plan [--json] [--output <file>] [install flags]
m plan update [pkg...] [--json] [--output <file>] [update flags]
```

| Command | Behavior |
|---|---|
| `m plan` | Install preview from current manifest (and lock when present) |
| `m plan update [pkg...]` | Update preview; empty args refresh direct deps only |

## Flags

Install-family flags accepted where applicable:

| Flag | Notes |
|---|---|
| `--prod` | Omit devDependencies |
| `--frozen-lockfile` | Fail on manifest/lock drift |
| `--linker` | `hoisted` or `isolated` |
| `--ignore-scripts` | Omit lifecycle `script` ops from plan detail |
| `-r` / `--recursive` | Workspace importers (workspaces gate) |
| `--filter` | Limit workspace importers (global flag) |
| `--json` | Emit `app.InstallResult` JSON (includes `plan`, counts) |
| `--output <file>` | Write `plan.Plan` JSON only (CI review artifact) |

`--output` does not persist plans by default elsewhere; it is an explicit
export for agents and review pipelines.

## Output shape

### Human mode

Summary lines matching install dry-run (`dry-run: +N -M ~K packages`).

### JSON mode (`--json`)

`app.InstallResult`:

| Field | Meaning |
|---|---|
| `added`, `removed`, `changed`, `packages` | Package key delta counts |
| `plan` | `plan.Plan` (see below) |

### Plan file (`--output`)

`plan.Plan` (`internal/plan`):

| Field | Meaning |
|---|---|
| `schemaVersion` | Plan schema (currently `1`) |
| `desired` | Target package keys + integrity |
| `operations` | `fetch`, `link`, `unlink`, `script` steps |
| `commits` | Commit-phase actions (lock write, root swap, etc.) |

Schema reference: [`data-model.md`](data-model.md).

## Relation to install dry-run

| Path | Entry |
|---|---|
| `m plan` | `app.PlanInstall` → `InstallOptions.DryRun = true` |
| `m install --dry-run` | Same resolver + `BuildMutationPlan` chain |
| `m plan update` | `app.PlanUpdate` → update resolve + dry-run |

`m plan --json` and `m install --dry-run --json` produce identical `plan`
payloads for the same flags and project state (regression:
`tests/integration/plan_test.go`).

## Non-mutation guarantee

Plan commands never:

- write `package.json` or lockfiles
- create or modify `node_modules`
- open an install transaction journal

Integration coverage: `tests/integration/plan_test.go`,
`tests/integration/explain_plan_diff_nomutation_test.go`.

## Deferred

- Plan signing / checksum hardening (schema has `schemaVersion` only today)
- `m shell --snapshot` — MVP **0045**

See also [`install.md`](install.md), [`explain.md`](explain.md).
