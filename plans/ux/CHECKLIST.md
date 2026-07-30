# Mew CLI Presentation and UX Master Checklist

## Program status

- Current standalone plan: **UX-0007 — Advanced Help, Pager, and Markdown**
- Status: active standalone program
- Recommended repository path: `plans/ux/`
- Numbering namespace: `UX-0001` through `UX-0008`
- Main roadmap status: independent; no `00xx` IDs are consumed
- Prerequisite note: main plan 0046 remains required for UX-0005 only (waived for UX-0001)
- Language: English
- Scope: human CLI presentation modernization without product-semantic changes
- Last updated: 2026-07-31

## UX-0001 — CLI Presentation Contract and Architecture

- [x] Audit every output and prompt site.
- [x] Freeze output modes and legacy flag compatibility.
- [x] Freeze stdout/stderr ownership.
- [x] Freeze exit, cancellation, broken-pipe, and reporter-failure behavior.
- [x] Add presentation package boundary.
- [x] Add import-boundary test.
- [x] Add capability/options resolver contract.
- [x] Add semantic event extensions only where necessary.
- [x] Add controller lifecycle and suspend/resume contract.
- [x] Add temporary rollout fallback switch.
- [x] Pass structured-output and stream regression tests.

## UX-0002 — Terminal Capabilities and Design System

- [x] Pin and review Lip Gloss v2.
- [x] Implement immutable capability snapshot.
- [x] Implement color/progress/Unicode/interactive resolution.
- [x] Implement semantic themes.
- [x] Implement Unicode and ASCII symbols.
- [x] Implement status, key-value, notice, hint, summary, delta, and table components.
- [x] Implement width 40/60/80/120 behavior.
- [x] Implement first-class plain renderer.
- [x] Migrate low-risk pilot commands.
- [x] Pass no-ANSI, width, theme, Unicode, Windows, and startup tests.

## UX-0003 — Errors, Help, and Static Command Output

- [x] Implement typed ErrorView.
- [x] Implement title and hint catalogs.
- [x] Preserve error codes and exits.
- [x] Implement grouped root help.
- [x] Implement command help sections and examples.
- [x] Certify completion output is ANSI-free.
- [x] Migrate low-risk static commands.
- [x] Migrate inspection/diagnostic commands.
- [x] Migrate security/artifact commands.
- [x] Add direct-printing static check or allowlist.
- [x] Pass human/plain/structured/redaction snapshots.

## UX-0004 — Install and Mutation Experience

- [x] Audit install phase instrumentation.
- [x] Add typed phase and outcome events.
- [x] Implement append-only plain progress first.
- [x] Pin/review Bubble Tea and Bubbles as needed.
- [x] Implement inline rich install renderer.
- [x] Add package and lifecycle summaries.
- [x] Migrate install/add/remove/update/ci/dedupe/prune.
- [x] Render rollback/recovery truthfully.
- [x] Certify cancellation and broken pipe.
- [x] Certify renderer failure cannot strand transaction resources.
- [x] Pass Windows and race tests.

## UX-0005 — Runner and Workspace Experience

- [x] Audit child terminal and stream contracts.
- [x] Add concise execution-preparation views.
- [x] Implement renderer suspend/resume.
- [x] Preserve raw single-child stdout/stderr/stdin.
- [x] Implement workspace aggregate renderer.
- [x] Implement workspace stream renderer.
- [x] Coordinate `mx` consent stages.
- [x] Add snapshot/capsule safe labels.
- [x] Certify signals, exits, cancellation, and interactive children.
- [x] Pass PTY/console, partial-line, binary, Windows shim, and race tests.

## UX-0006 — Prompts and Accessibility

- [x] Define prompt interface and policy.
- [x] Pin/review Huh v2.
- [x] Implement rich prompt adapter.
- [x] Implement accessible numbered prompt adapter.
- [x] Preserve lifecycle trust semantics.
- [x] Preserve `mx` consent ordering.
- [x] Audit destructive confirmations.
- [x] Disable prompts in non-TTY/CI/structured modes.
- [x] Add safe defaults, EOF, cancellation, and redaction tests.
- [x] Produce manual accessibility evidence.

## UX-0007 — Advanced Help, Pager, and Markdown

- [ ] Resolve help-topic grammar.
- [ ] Define topic registry and content ownership.
- [ ] Evaluate/pin Glamour v2.
- [ ] Implement plain topic renderer.
- [ ] Implement styled Markdown renderer.
- [ ] Implement safe optional pager.
- [ ] Add error, compatibility, lifecycle, snapshot/capsule, and runner topics.
- [ ] Validate links and examples.
- [ ] Pass Windows/no-pager, width, no-color, accessible, and broken-pipe tests.

## UX-0008 — CLI UX Certification and Rollout

- [ ] Create versioned conformance matrix.
- [ ] Certify output modes.
- [ ] Certify stdout/stderr byte ownership.
- [ ] Certify terminal cleanup.
- [ ] Certify Linux, macOS, and Windows.
- [ ] Certify widths, themes, no-color, Unicode, and ASCII.
- [ ] Certify accessibility.
- [ ] Certify JSON/NDJSON and completions.
- [ ] Benchmark startup, binary size, and live rendering.
- [ ] Review licenses, vulnerabilities, and dependency pins.
- [ ] Execute staged rollout.
- [ ] Update docs, inventory, plan index, and checklist.
- [ ] Remove temporary legacy path after stability window.

## Global non-negotiable gates

- [ ] No product/domain package imports Charm.
- [ ] No progress on stdout.
- [ ] No ANSI/cursor control in plain, pipe, CI, JSON, NDJSON, or completion output.
- [ ] No prompt in non-TTY or structured mode.
- [ ] No child stdio, signal, or exit-code regression.
- [ ] No transaction truthfulness regression.
- [ ] No security/integrity warning suppression.
- [ ] No full-screen default UI.
- [ ] No interactive script/package picker introduced.
- [ ] No telemetry added.
