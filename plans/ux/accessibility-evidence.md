# UX-0006 accessibility evidence

**Date:** 2026-07-31  
**Host OS:** Windows 10 (build 26200)  
**Mode:** `--accessible` / numbered AccessiblePrompter (append-only)

## Reading order (lifecycle trust)

Deterministic top-to-bottom stderr lines; no cursor-up redraw:

1. Title: `Package <name> requests permission to run an install script.`
2. Description: `Allow this package?`
3. Fields: `Package: <name>` (already redacted)
4. `Choose an action:`
5. Numbered options `1.` … `N.`
6. `Selection [1]:` (default = Deny)

## Keyboard-only steps verified

| Step | Input | Outcome |
|---|---|---|
| Default deny | Enter (empty) | Option `deny` |
| Allow once | `2` + Enter | Option `allow-once` (not persisted) |
| Trust project | `3` + Enter | Option `trust-project` (persisted) |
| Invalid | `9` then `1` | Re-prompt then deny |
| EOF | closed stdin | Deny (SafeDefaultID) |
| Width 40 | forced width | Title wraps; no ANSI |
| NO_COLOR / ASCII | accessible path | No color escapes |

## Constraints checked

- No cursor-up / live repaint dependency for accessible prompts.
- Prompt text on stderr; answers on stdin only.
- JSON/NDJSON/silent/`--interactive=never`/CI auto: no prompt.
- Secret input values cleared from `PromptAnswer.Value` before return.
- Live progress Suspend runs before Prompt; Resume afterward.

## Automated coverage

- `internal/prompt` policy matrix tests
- `internal/presentation/prompt` accessible parsing / EOF / suspend tests
- `internal/lifecycle` ask approve/deny/allow-once/non-TTY tests
- `internal/runner/dlx` consent order and Prompter tests

Manual screen-reader walkthrough is recorded here as checklist evidence for UX-0006;
full multi-platform a11y certification remains UX-0008.

---

# UX-0008 accessibility certification checklist

**Date:** 2026-07-31
**User doc:** [`docs/accessibility.md`](../../docs/accessibility.md)
**Platform evidence:** [`docs/evidence/cli-ux/`](../../docs/evidence/cli-ux/)

## Automated (required)

| Check | Evidence |
|---|---|
| Accessible / plain / CI / pipe emit no CSI | presentation resolve + renderer tests; Windows redirected smoke in `windows-2026-07-31.md` |
| Numbered prompt parse / EOF / safe default | `internal/presentation/prompt` |
| Policy: no prompt in CI / structured / non-TTY | `internal/prompt` |
| Completions ANSI-free | `internal/cli` `TestCompletionNoANSI` |
| Width 40/60/80/120 for static + accessible text | design-system / prompt width tests |

## Manual checklist (operator)

Perform on at least one Windows console host (done 2026-07-31 for UX-0006 path)
and record Linux/macOS when those evidence slots are filled:

1. Enable `--accessible` (or `MEW_ACCESSIBLE=1`).
2. Confirm install/help/version output is append-only top-to-bottom (no cursor-up).
3. Confirm readable without color (`NO_COLOR` or `--color=never`).
4. Confirm ASCII path with `--unicode=never` still conveys status.
5. Keyboard-only lifecycle trust prompt: Enter → deny; `2` → allow-once; EOF → deny.
6. Force width ~40 columns: title wraps; options remain numbered; no CSI.
7. Structured mode (`--output=json`): no prompt, no human ANSI on stdout.

## Platform notes

| Platform | Status |
|---|---|
| Windows native | Checklist + UX-0006 keyboard steps recorded on build 26200 |
| Linux | Slot open — Docker daemon unavailable on measurement host; fill with Docker or Ubuntu CI |
| macOS | Slot open — record via GitHub Actions job SHA |

## Explicit non-claim

This is not a screen-reader lab certification. It is product-checklist evidence
plus automated append-only / no-ANSI coverage.
