---
name: UX-0006 Prompts and Accessibility
overview: Introduce a secure prompt abstraction with Huh-backed rich prompts and accessible plain fallbacks, modernize lifecycle trust and confirmations, enforce non-TTY fail-closed behavior, and make all human output usable without animation, color, Unicode, or cursor repaint.
todos:
  - id: p1-prompt-contract
    content: Define prompt request/response, policy, cancellation, audit, and non-TTY contracts
    status: pending
  - id: p2-huh
    content: Integrate Huh v2 behind internal/presentation/prompt without exposing it to domain packages
    status: pending
  - id: p3-accessible
    content: Implement screen-reader-friendly numbered prompts and append-only accessible output mode
    status: pending
  - id: p4-lifecycle
    content: Migrate lifecycle trust confirmation while preserving trust-store and fail-closed semantics
    status: pending
  - id: p5-confirmations
    content: Migrate approved destructive confirmations and define yes/non-interactive behavior
    status: pending
  - id: p6-policy
    content: Define prompt suppression in CI, structured modes, pipes, and interactive child contexts
    status: pending
  - id: p7-tests
    content: Add TTY, non-TTY, cancellation, default-choice, screen-reader, redaction, and trust-policy tests
    status: pending
isProject: false
---

# UX-0006 — Prompts and Accessibility

## Goal

Provide polished prompts without weakening security policy or excluding users who rely on screen readers, append-only terminals, no-color environments, or non-interactive automation.

Prompt UX must be driven by domain policy. The prompt library never decides whether an operation is allowed.

## User-visible deliverables

Rich lifecycle prompt:

```text
Package esbuild requests permission to run an install script.

  Package   esbuild
  Script    install
  Project   my-app

Allow this package?

> Deny
  Allow once
  Trust for this project
```

Accessible fallback:

```text
Package esbuild requests permission to run an install script.
Package: esbuild
Script: install
Project: my-app

Choose an action:
1. Deny
2. Allow once
3. Trust for this project
Selection [1]:
```

Non-TTY:

```text
ERROR Lifecycle script approval is required
Package esbuild is not trusted and interactive approval is unavailable.
Code: ERR_M_POLICY
Hint: Review and approve the package explicitly before running this command.
```

Use actual repository grammar and codes.

## Explicitly out of scope

- Interactive script picker.
- Interactive package search.
- Command palette.
- Multi-page setup wizard unless separately approved.
- Mouse-required interactions.
- Prompting in JSON/NDJSON mode.
- Prompting in CI by default.
- Changing trust-store scope or security semantics.
- Treating accessibility as color-only high contrast.

## Dependency

```go
import "charm.land/huh/v2"
```

Huh is isolated behind an internal adapter. Domain and app packages consume a Mew prompt interface.

Review:

- exact version;
- accessible-mode behavior;
- Bubble Tea dependency impact;
- terminal cleanup;
- Windows console behavior;
- cancellation handling;
- binary-size/startup impact;
- license/security.

## Prompt abstraction

```go
type PromptKind string

const (
    PromptConfirm PromptKind = "confirm"
    PromptSelect  PromptKind = "select"
    PromptInput   PromptKind = "input"
)

type PromptRequest struct {
    ID          string
    Kind        PromptKind
    Title       string
    Description string
    Fields      []KeyValue
    Options     []PromptOption
    DefaultID   string
    Dangerous   bool
    Secret      bool
}

type PromptAnswer struct {
    OptionID string
    Value    string
    Cancelled bool
}

type Prompter interface {
    Prompt(context.Context, PromptRequest) (PromptAnswer, error)
}
```

Rules:

- Stable option IDs, localized labels later if ever needed.
- Domain logic interprets option ID.
- Prompt adapter does not write trust store or perform mutation.
- Secret input is never logged or returned in diagnostics.
- Prompt request passes already-redacted display fields.

## Prompt policy

```go
type InteractivePolicy string

const (
    InteractiveAuto   InteractivePolicy = "auto"
    InteractiveAlways InteractivePolicy = "always"
    InteractiveNever  InteractivePolicy = "never"
)
```

### Auto permits prompt only when

- stdin is a TTY;
- human output mode;
- not CI;
- no child currently owns the terminal;
- command contract allows interaction;
- prompt provider is available;
- accessibility mode has a supported prompt implementation.

### Always

Even forced interactivity cannot bypass security or impossible terminal constraints. If no usable TTY exists, return a typed usage/policy error instead of hanging.

### Never

No prompt. Domain receives an unavailable/denied result according to policy.

## Default choices

Security-sensitive prompt defaults are fail-closed.

- Lifecycle trust default: deny.
- Destructive operation default: cancel/no.
- Never auto-accept because stdin reaches EOF.
- Enter with empty input selects only an explicitly safe default.
- Timeout/cancellation does not accept.

## Lifecycle trust integration

Preserve current trust semantics:

- already trusted package proceeds without prompt;
- deny mode remains deny unless trusted;
- ask mode prompts only when interactive policy allows;
- non-TTY ask fails closed;
- trust-for-project persists through existing trust store;
- allow-once, if supported, is request-scoped and not persisted;
- prompt cancellation returns typed cancellation/policy result according to existing contract.

The prompt must not broaden trust from package/version/script scope beyond current product policy.

## Consent and remote execution

Where `mx` requires consent:

- show resolved package/spec and reason;
- do not show credentials or authenticated URLs;
- artifact fetch ordering remains enforced by product code;
- default is deny;
- non-TTY requires explicit existing approval flag/policy;
- prompt result may be audited through semantic events without recording private input.

## Destructive confirmations

Audit commands that currently or potentially require confirmation:

```text
prune/remove operations with destructive scope
rollback/recovery choices
cache/store destructive maintenance
publish actions
migration actions
```

Do not add prompts to deterministic non-interactive commands just because a prompt library exists.

For each prompt define:

- risk being confirmed;
- exact effect;
- default;
- `--yes` or equivalent behavior if already supported;
- non-interactive behavior;
- audit record;
- cancellation behavior.

## Accessible mode

Accessible mode is an end-to-end presentation policy.

Recommended implications:

```text
output: plain/append-only human
progress: never/live repaint disabled
unicode: ASCII or verified safe subset
color: never or explicitly high-contrast non-essential
prompt: numbered line-oriented form
animation: disabled
reading order: deterministic
```

Do not rely on screen readers interpreting cursor-up redraws.

## No-color and non-Unicode requirements

Every prompt must remain clear with:

- no color;
- ASCII symbols;
- terminal width 40;
- no arrow-key navigation;
- pasted numeric/text input;
- Enter default;
- EOF;
- Ctrl+C.

## Prompt rendering lifecycle

Before prompt:

1. suspend live progress renderer;
2. restore cursor;
3. ensure no child owns stdin;
4. flush pending status lines.

After prompt:

1. clear transient prompt safely without erasing unrelated output, or leave a compact answer summary;
2. resume live renderer only if operation continues;
3. never print secret answer;
4. emit typed audit/status event if contract requires.

## Answer summaries

For security-sensitive choices, show a concise non-secret result:

```text
✓ Trusted esbuild for this project
```

or:

```text
! Lifecycle script was not approved
```

Do not echo arbitrary input values by default.

## Structured mode behavior

- JSON/NDJSON mode never launches Huh.
- Domain returns a structured policy/interaction-required error or consumes explicit flags.
- No prompt text leaks into stdout.
- Prompt-required state may use an existing/versioned structured error field.

## Implementation phases

### Phase 1 — Prompt inventory and policy

Inventory current prompts and confirmation-like reads. Freeze per-command behavior.

### Phase 2 — Mew prompt interface

Implement request/answer models and plain test prompter.

### Phase 3 — Huh adapter

Add rich prompt renderer with injected I/O and terminal settings.

### Phase 4 — Accessible adapter

Implement line-oriented numbered prompts with no cursor repaint.

### Phase 5 — Lifecycle and consent migration

Migrate lifecycle trust and approved runner consent paths.

### Phase 6 — Destructive confirmation audit

Migrate only commands whose existing semantics require confirmation.

### Phase 7 — Certification

TTY, non-TTY, CI, structured, cancellation, EOF, width, Windows, and accessibility tests.

## Error behavior

| Case | Behavior |
|---|---|
| Prompt required, stdin non-TTY | typed policy/usage error; no hang |
| Prompt required in JSON mode | structured interaction-required error |
| User selects deny | domain-specific policy result |
| User presses Ctrl+C | existing cancellation mapping |
| stdin EOF | safe default deny/cancel |
| Invalid selection | re-prompt with bounded clear message in interactive mode |
| Prompt renderer failure | typed I/O error; operation does not proceed |
| Trust-store save failure | typed product error; do not claim trust persisted |
| Live renderer cannot suspend | prompt does not start; typed output error or plain fallback per policy |

## Tests

### Unit

- policy resolver;
- safe defaults;
- option validation;
- duplicate option IDs;
- secret redaction;
- accessible parsing;
- EOF;
- invalid input;
- cancellation;
- prompt lifecycle suspend/resume.

### Integration

- trusted package no prompt;
- ask + TTY approve/deny;
- ask + non-TTY failure;
- `--interactive=never`;
- CI auto behavior;
- JSON mode;
- trust persistence;
- trust-store failure;
- `mx` consent order;
- Windows console;
- width 40;
- screen-reader/accessible mode.

### Security

- approval cannot occur on empty input;
- artifact fetch cannot begin before required consent;
- secret/token values absent from snapshots;
- prompt cannot alter trust scope;
- structured output cannot be corrupted by prompt text.

## Acceptance criteria

- Huh is isolated behind Mew interfaces.
- Lifecycle trust remains fail-closed.
- Non-TTY commands never hang waiting for input.
- Accessible mode is append-only and keyboard-simple.
- Prompt cancellation preserves existing exit behavior.
- Structured modes never prompt.
- Live renderer suspends safely.
- No secret answer is logged.
- Interactive selection features not covered by existing product policy remain deferred.

## Risks

| Risk | Mitigation |
|---|---|
| Rich prompt changes trust semantics | domain interprets answer; adapter only renders |
| CI hangs | strict non-TTY/CI gating and timeout tests |
| Screen reader cannot follow repaint | dedicated append-only prompt implementation |
| EOF accepts dangerous default | fail-closed defaults and tests |
| Huh owns terminal incorrectly | suspend lifecycle and PTY tests |
| Prompt leaks secret input | secret field policy and snapshot redaction tests |

## Estimated effort

**12–20 focused engineering hours**.

## References

- Huh v2: https://charm.land/huh
