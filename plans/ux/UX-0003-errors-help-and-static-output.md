---
name: UX-0003 Errors, Help, and Static Command Output
overview: Replace ad-hoc human output with typed error presentation, actionable hints, responsive tables, consistent summaries, command-grouped help, and reusable rendering for static command families while preserving machine output and error codes.
todos:
  - id: p1-error-view
    content: Define ErrorView mapping from typed Mew errors without coupling apperr to styling
    status: pending
  - id: p2-hints
    content: Add deterministic hint catalog, context fields, docs references, and debug cause rendering
    status: pending
  - id: p3-help
    content: Implement grouped Cobra help templates, examples, common workflows, and experimental visibility rules
    status: pending
  - id: p4-static-commands
    content: Migrate version, features, project, config, doctor, list, outdated, explain, plan, store, cache, audit, and policy human output
    status: pending
  - id: p5-tables
    content: Adopt responsive table and key-value components without changing structured results
    status: pending
  - id: p6-streams
    content: Certify stdout/stderr ownership for every migrated command
    status: pending
  - id: p7-tests
    content: Add golden, redaction, help, width, no-color, machine-output, and error-code tests
    status: pending
isProject: false
---

# UX-0003 — Errors, Help, and Static Command Output

## Goal

Make non-live Mew commands clear, consistent, concise, and actionable.

This plan establishes the visible language users encounter most often: errors, warnings, success messages, summaries, tables, and command help. It removes scattered formatting decisions from command handlers while leaving product results and machine schemas unchanged.

## User-visible deliverables

Representative error:

```text
× Installation failed

  Lockfile does not match package.json.

  Project   C:\work\app
  Lockfile  m.lock
  Code      ERR_M_LOCKFILE

  Run `m install` to update the lockfile.
```

Representative help:

```text
MewJS — JavaScript package manager and toolchain

Usage
  m <command> [flags]

Common workflows
  m install             Install project dependencies
  m add <package>       Add a dependency
  m run <script>        Run a package script
  m exec <binary>       Run a project binary

Inspect
  ls        outdated    explain
  project   doctor      features

Use `m <command> --help` for command details.
```

## Explicitly out of scope

- Install live progress; see UX-0004.
- Runner/workspace live output; see UX-0005.
- Interactive prompts; see UX-0006.
- Long-form Markdown help and pager integration; see UX-0007.
- Changing typed error codes or exit semantics except to correct proven bugs through separate approval.
- Showing internal stack traces by default.
- Embedding style data in `apperr`.

## Error presentation architecture

Typed errors remain owned by existing error packages. Presentation maps them to a display model.

```go
type ErrorView struct {
    Severity  Severity
    Title     string
    Message   string
    Code      string
    Operation string
    Subject   string
    Context   []KeyValue
    Hints     []Hint
    Causes    []CauseView
    DocsTopic string
}
```

### Mapping rules

- Preserve the canonical `ERR_M_*` code.
- Preserve exit mapping.
- Redact before display.
- Generate a concise title from code/category/operation mapping.
- Keep the primary message factual and direct.
- Include only high-value context.
- Include at most one primary action hint by default; secondary hints are allowed when necessary.
- Show cause chain only under debug/verbose policy.
- Never expose Go type names or stack traces in normal mode.
- Never claim rollback or non-mutation unless product state proves it.

## Error title catalog

Create a deterministic catalog keyed by stable error category/code, not fragile string matching.

Examples:

```text
ERR_M_USAGE        -> Invalid command usage
ERR_M_CONFIG       -> Configuration is invalid
ERR_M_NOT_FOUND    -> Required item was not found
ERR_M_INTEGRITY    -> Integrity verification failed
ERR_M_NETWORK      -> Network operation failed
ERR_M_LOCKFILE     -> Lockfile validation failed
ERR_M_TRANSACTION  -> Project update failed
ERR_M_POLICY       -> Operation blocked by policy
ERR_M_CANCELLED    -> Operation cancelled
ERR_M_INTERNAL     -> Mew encountered an internal error
```

Use actual repository codes. Do not invent codes that do not exist; map to current categories where necessary.

## Hint catalog

Hints are semantic and testable.

```go
type HintRule struct {
    Code       string
    Operation  string
    Predicate  func(ErrorMetadata) bool
    Hint       Hint
}
```

Rules:

- Do not infer fixes from arbitrary message substrings.
- Hints must be safe and non-destructive.
- Destructive recovery suggestions require explicit wording.
- Commands in hints must use current CLI grammar.
- Do not suggest network access in offline mode unless the hint explains the conflict.
- Do not show more than three hints in default mode.

## Error layout

### Default

1. status symbol and title;
2. blank line;
3. primary message;
4. optional context block;
5. error code;
6. actionable hint.

### Narrow terminal

Use stacked key/value fields and wrap messages without breaking command tokens where possible.

### Plain/CI

```text
ERROR Installation failed
Lockfile does not match package.json.
Project: C:\work\app
Lockfile: m.lock
Code: ERR_M_LOCKFILE
Hint: Run `m install` to update the lockfile.
```

### JSON/NDJSON

No human layout changes. Existing structured error schemas remain authoritative and receive only versioned intentional changes.

## Warning and notice policy

Define severity:

```text
info
notice
warning
deprecation
security-warning
```

Rules:

- Warnings go to stderr in human/plain modes.
- Structured modes encode warnings according to existing command schema/event policy.
- Avoid repeating the same warning more than once per operation unless state changes.
- Group repeated lifecycle or package warnings into a summary when possible.
- Security warnings cannot be hidden by `--no-summary`; they follow existing silent policy.

## Success and summary policy

Do not print success banners for commands whose stdout result is self-explanatory, such as a single queried value.

Print a summary when it adds useful state:

- install/mutation counts;
- snapshot creation;
- recovery outcome;
- audit result;
- policy result;
- benchmark/conformance outcome.

Summary duration must use injected/recorded operation duration, not render-time measurement.

## Help information architecture

### Root help groups

Proposed groups, adjusted to actual registered commands:

```text
Common workflows
Project and dependencies
Run and execute
Inspect and diagnose
Security and policy
Cache, store, and artifacts
Configuration and development
```

Do not show unavailable or hidden commands as shipped.

### Command help sections

```text
Usage
Summary
Examples
Arguments
Flags
Global flags
Related commands
```

Only render sections that contain content.

### Experimental commands and flags

- Mark experimental surfaces explicitly.
- Keep internal/development flags hidden by default where current policy permits.
- Include gate/config information only when useful to the user.
- Never imply an experimental shortcut is required for the canonical workflow.

### Examples

Every example must be executable against current grammar and must avoid destructive or network-heavy behavior unless clearly explained.

## Cobra integration

- Define shared usage/help templates at the root.
- Preserve `SilenceUsage` and `SilenceErrors` behavior.
- Usage is printed for usage errors only according to existing policy.
- Runtime errors must not dump full command usage.
- Completion command output remains shell-compatible and unstyled.
- Shell completion directives must never contain ANSI.

## Static command migration inventory

Migrate human output in controlled groups.

### Group A — low-risk identity and configuration

```text
m version
mx version
m features
m project
m pkg
m config get/list
```

### Group B — diagnostics and inspection

```text
m doctor
m ls
m outdated
m explain
m plan
m history
m snapshot list
m cache
m store
```

### Group C — security and artifacts

```text
m audit
m policy check
m verify provenance
m sbom
m builds
m pack
m capsule inspect/list where applicable
```

Actual command names must be confirmed in code.

## Command-result models

Do not make renderers parse command strings or JSON intended for machine output.

Create internal view models where needed:

```go
type VersionView struct { ... }
type FeatureTableView struct { ... }
type ProjectView struct { ... }
type DoctorView struct { ... }
type PackageListView struct { ... }
type OutdatedView struct { ... }
type AuditView struct { ... }
```

Rules:

- View models contain already-redacted display data or typed safe values.
- Machine encoders continue using authoritative result models.
- Sorting happens before rendering and is deterministic.
- Renderers do not read project files or perform network calls.

## Doctor UX

Suggested output:

```text
Mew doctor

✓ Project manifest
✓ Lockfile
✓ Dependency tree
! 2 lifecycle scripts require review
✓ Global store

1 warning
Run `m builds` to review blocked scripts.
```

Failure detail:

```text
× Lockfile
  package.json and m.lock describe different dependencies.
  Run `m install` to update the lockfile.
```

Doctor must preserve existing check ordering and exit policy.

## List and outdated UX

Use responsive tables with deterministic sorting.

Wide:

```text
PACKAGE       CURRENT   WANTED   LATEST   TYPE
react         19.1.0    19.1.1   19.1.1   dependency
typescript    5.8.2     5.8.3    5.8.3    dev
```

Narrow: stacked records from UX-0002.

Machine output remains unchanged.

## Explain and plan UX

- Use clear headings for decision, reason, source, and effect.
- Keep deep traces behind verbose/debug options.
- Use tree or indented lists only when width-safe.
- Avoid decorative boxes around large traces.
- Preserve exact package identifiers and error codes.

## Stream rules by command class

| Command class | stdout | stderr |
|---|---|---|
| single-value query | value | warnings/debug |
| table result | table | warnings/debug |
| JSON result | JSON | fatal setup errors only according to structured policy |
| generated document | document | progress/warnings |
| doctor/audit human | primary report unless current contract says stderr | warnings/debug |
| help/version | help/result | usually empty |

Audit existing behavior before changing stream ownership. Any intentional change requires compatibility approval and tests.

## Implementation phases

### Phase 1 — Error view and catalog

Implement mapping, context selection, hints, redaction, and renderers.

### Phase 2 — Shared help templates

Implement root and command help layout, group metadata, examples, and no-ANSI completion tests.

### Phase 3 — Low-risk command migration

Migrate Group A and freeze component APIs.

### Phase 4 — Inspection command migration

Migrate Group B with responsive tables and summaries.

### Phase 5 — Security/artifact command migration

Migrate Group C with special attention to redaction and security-warning visibility.

### Phase 6 — Remove ad-hoc formatting

Add lint/static checks or a maintained allowlist for direct human `fmt.Print*` usage in `internal/cli`.

## Test strategy

### Error snapshots

Test:

- no-color;
- dark/light themes;
- width 40/80/120;
- Windows/POSIX paths;
- redacted registry URL;
- one cause and multiple causes;
- debug cause chain;
- no hint;
- one hint;
- cancellation;
- internal error.

### Help tests

- root help command groups;
- command help sections;
- hidden commands absent;
- experimental labels;
- completion output has no ANSI;
- aliases show correct binary name;
- `mew`/`mewx` basename behavior remains compatible;
- width wrapping.

### Command regression

For every migrated command:

- human snapshot;
- plain snapshot;
- no-color snapshot;
- JSON/NDJSON equivalence;
- stdout/stderr ownership;
- exit code;
- redaction.

## Acceptance criteria

- Human errors use one consistent structure.
- Error codes and exit codes are unchanged.
- Hints are actionable and grammar-verified.
- Root and command help are grouped and readable.
- Static command tables respond to width.
- Machine output remains schema-compatible.
- No ANSI appears in redirected output or completions.
- Direct human formatting in migrated commands is removed.
- Security and policy output remains fail-closed and redacted.

## Risks

| Risk | Mitigation |
|---|---|
| Hint suggests an invalid command | grammar-backed tests and command registration checks |
| Error title hides important detail | message and code always retained |
| Help becomes too long | curated root help and command-specific detail |
| Table output changes scripts | preserve machine modes and audit human stream compatibility |
| Security warning is suppressed by summary settings | severity policy and explicit tests |
| Direct printing continues to spread | static check/allowlist and migration inventory |

## Estimated effort

**18–28 focused engineering hours**.
