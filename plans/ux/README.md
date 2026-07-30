# Mew CLI Presentation and UX Program

This directory is a standalone implementation program for modernizing the human-facing CLI of Mew. It is deliberately separated from the primary `plans/00xx-*` product roadmap.

## Independence contract

- Local plan IDs use the `UX-` namespace.
- The program owns its own [`INDEX.md`](INDEX.md) and [`CHECKLIST.md`](CHECKLIST.md).
- The recommended repository location is `plans/cli-ux/`.
- The root roadmap may contain one link to this directory, but it does not need to duplicate the individual UX plans.
- Main-roadmap plans referenced by this program are prerequisites, not members of this numbering sequence.
- Completing a UX plan must not automatically advance the main product MVP.

## Plan sequence

1. [`UX-0001`](UX-0001-cli-presentation-contract.md) — CLI Presentation Contract and Architecture
2. [`UX-0002`](UX-0002-terminal-capabilities-and-design-system.md) — Terminal Capabilities and Design System
3. [`UX-0003`](UX-0003-errors-help-and-static-output.md) — Errors, Help, and Static Command Output
4. [`UX-0004`](UX-0004-install-and-mutation-experience.md) — Install and Mutation Experience
5. [`UX-0005`](UX-0005-runner-and-workspace-experience.md) — Runner and Workspace Experience
6. [`UX-0006`](UX-0006-prompts-and-accessibility.md) — Prompts and Accessibility
7. [`UX-0007`](UX-0007-advanced-help-pager-and-markdown.md) — Advanced Help, Pager, and Markdown
8. [`UX-0008`](UX-0008-cli-ux-certification-and-rollout.md) — CLI UX Certification and Rollout

## Program boundary

This program changes presentation, terminal capability handling, prompts, and human-readable output. It does not change dependency resolution, transaction semantics, runner semantics, security policy, machine-output schemas, child-process behavior, or the stock Node.js execution boundary.

Interactive script or package selection remains deferred unless a separate product plan explicitly implements it.
