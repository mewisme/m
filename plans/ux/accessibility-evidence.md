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
