# Accessibility

Mew supports an accessible CLI path for humans who need append-only, color-
independent output and keyboard-only prompts.

## Enable

| Surface | Value |
|---|---|
| Flag | `--accessible` |

Accessible mode preserves rich output formatting while selecting the numbered
prompt adapter. Use `--no-color` or `--ascii` for additional accessibility
adjustments.

Related: `--no-color`, `--ascii`, `--output=plain`, `--no-progress`.

## Prompt contract

Accessible prompts are Mew-owned (not Huh accessible mode):

1. Title and description on stderr
2. Redacted field rows
3. Numbered options `1.` … `N.`
4. `Selection [default]:` line
5. Answers on stdin only

EOF and closed stdin use the safe default (typically deny). Invalid numbers
re-prompt. Secrets are cleared from returned values.

Non-interactive / CI / structured (`json` / `ndjson` / `silent`) paths never
prompt.

## Screen-reader checklist

Manual checklist and Windows evidence:
[`plans/ux/accessibility-evidence.md`](../plans/ux/accessibility-evidence.md).

UX-0008 extends that file with a certification checklist. Full multi-platform
screen-reader lab certification is out of scope; automated coverage asserts
append-only / no-ANSI under `--accessible` and policy fail-closed behavior.

## Architecture

Resolution and Charm isolation:
[`docs/architecture/cli-presentation.md`](architecture/cli-presentation.md).
