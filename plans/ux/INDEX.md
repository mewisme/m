# Mew CLI Presentation and UX Plan Index

This directory is a **standalone implementation program** for modernizing Mew's human-facing CLI without changing package-manager, transaction, runner, or runtime semantics.

The `UX-` prefix creates an independent numbering namespace. These plans are intentionally separate from the repository's primary `plans/00xx-*` delivery train and do not consume IDs from that roadmap. The recommended repository location is `plans/cli-ux/`. The main roadmap may link to this program, but the program keeps its own index, checklist, sequencing, and completion state.

## Program principles

- Pretty when human; predictable when automated.
- Preserve stdout, stderr, exit-code, JSON, NDJSON, redaction, and child-process contracts.
- Keep product logic independent of terminal UI libraries.
- Prefer inline, append-safe terminal presentation over full-screen interfaces.
- Use Charm libraries selectively rather than making Bubble Tea the execution framework for every command.
- Treat accessibility, non-TTY behavior, CI behavior, Windows behavior, cancellation, and broken pipes as first-class product requirements.
- Keep interactive script/package selection deferred unless a separate product plan explicitly ships it.

## Proposed plan sequence

| ID | Plan | Purpose | Depends on |
|---|---|---|---|
| UX-0001 | [CLI Presentation Contract and Architecture](UX-0001-cli-presentation-contract.md) | Freeze output modes, stream ownership, package boundaries, event contracts, dependency policy, and migration strategy. | Repository prerequisites: main plans 0005, 0010, 0046 |
| UX-0002 | [Terminal Capabilities and Design System](UX-0002-terminal-capabilities-and-design-system.md) | Implement capability detection, mode resolution, theme tokens, symbols, width handling, and reusable static components. | UX-0001 |
| UX-0003 | [Errors, Help, and Static Command Output](UX-0003-errors-help-and-static-output.md) | Replace ad-hoc human printing with structured errors, hints, tables, summaries, help grouping, and responsive command output. | UX-0001, UX-0002 |
| UX-0004 | [Install and Mutation Experience](UX-0004-install-and-mutation-experience.md) | Add rich and plain progress for install-family operations while preserving transactions, rollback, and CI output. | UX-0001, UX-0002, UX-0003 |
| UX-0005 | [Runner and Workspace Experience](UX-0005-runner-and-workspace-experience.md) | Coordinate status presentation with child stdio, workspace aggregation, execution preparation, signals, and interactive children. | Repository prerequisite: main plan 0046; local: UX-0001, UX-0002, UX-0003 |
| UX-0006 | [Prompts and Accessibility](UX-0006-prompts-and-accessibility.md) | Introduce accessible prompt adapters, lifecycle trust UX, non-TTY fail-closed behavior, ASCII fallbacks, and screen-reader mode. | Repository prerequisite: main plan 0021; local: UX-0001, UX-0002, UX-0003 |
| UX-0007 | [Advanced Help, Pager, and Markdown](UX-0007-advanced-help-pager-and-markdown.md) | Add optional long-form terminal documentation, pager policy, topic help, and width-aware Markdown rendering. | UX-0002, UX-0003, UX-0006 |
| UX-0008 | [CLI UX Certification and Rollout](UX-0008-cli-ux-certification-and-rollout.md) | Certify compatibility, platform behavior, output cleanliness, performance, rollout gates, documentation, and inventory updates. | UX-0001 through UX-0007 |

## Recommended Charm dependency policy

Initial implementation should evaluate and pin compatible releases of:

```text
charm.land/lipgloss/v2
charm.land/bubbletea/v2
charm.land/bubbles/v2
charm.land/huh/v2
charm.land/glamour/v2    # introduced only by UX-0007 if approved
```

The dependency review must record exact versions, licenses, transitive dependency impact, binary-size impact, startup impact, security posture, and upgrade policy before merge.

## Repository observations that shape these plans

- Mew already has semantic diagnostics events and reporter modes for human, JSON, NDJSON, and silent output.
- The current human reporter mostly converts events into plain strings and writes them directly.
- Mew already separates many child-output events by stream and has structured runner/workspace events.
- Cobra remains the CLI command framework.
- Product packages must continue emitting semantic state rather than terminal-specific instructions.

## Completion model

The program is complete only when all of the following are true:

1. Human terminal output is consistent across command families.
2. CI and redirected output are append-only and free of ANSI/cursor control.
3. JSON and NDJSON remain valid and versioned.
4. Child stdout, child stderr, stdin ownership, signals, and exit codes are unchanged.
5. Error output is concise, typed, actionable, and redacted.
6. Accessibility mode works without animation, color dependence, or cursor repaint.
7. Windows, Linux, and macOS evidence is available.
8. Charm dependencies remain isolated to presentation and CLI packages.
9. Simple commands retain acceptable startup and binary-size characteristics.
10. A rollback switch can disable rich presentation without reverting product changes.

See [CHECKLIST.md](CHECKLIST.md) for the consolidated implementation checklist.
