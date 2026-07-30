# Mew CLI Presentation and UX Master Checklist

## Program status

- Current standalone plan: **UX-0003 — Errors, Help, and Static Command Output**
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

- [ ] Implement typed ErrorView.
- [ ] Implement title and hint catalogs.
- [ ] Preserve error codes and exits.
- [ ] Implement grouped root help.
- [ ] Implement command help sections and examples.
- [ ] Certify completion output is ANSI-free.
- [ ] Migrate low-risk static commands.
- [ ] Migrate inspection/diagnostic commands.
- [ ] Migrate security/artifact commands.
- [ ] Add direct-printing static check or allowlist.
- [ ] Pass human/plain/structured/redaction snapshots.

## UX-0004 — Install and Mutation Experience

- [ ] Audit install phase instrumentation.
- [ ] Add typed phase and outcome events.
- [ ] Implement append-only plain progress first.
- [ ] Pin/review Bubble Tea and Bubbles as needed.
- [ ] Implement inline rich install renderer.
- [ ] Add package and lifecycle summaries.
- [ ] Migrate install/add/remove/update/ci/dedupe/prune.
- [ ] Render rollback/recovery truthfully.
- [ ] Certify cancellation and broken pipe.
- [ ] Certify renderer failure cannot strand transaction resources.
- [ ] Pass Windows and race tests.

## UX-0005 — Runner and Workspace Experience

- [ ] Audit child terminal and stream contracts.
- [ ] Add concise execution-preparation views.
- [ ] Implement renderer suspend/resume.
- [ ] Preserve raw single-child stdout/stderr/stdin.
- [ ] Implement workspace aggregate renderer.
- [ ] Implement workspace stream renderer.
- [ ] Coordinate `mx` consent stages.
- [ ] Add snapshot/capsule safe labels.
- [ ] Certify signals, exits, cancellation, and interactive children.
- [ ] Pass PTY/console, partial-line, binary, Windows shim, and race tests.

## UX-0006 — Prompts and Accessibility

- [ ] Define prompt interface and policy.
- [ ] Pin/review Huh v2.
- [ ] Implement rich prompt adapter.
- [ ] Implement accessible numbered prompt adapter.
- [ ] Preserve lifecycle trust semantics.
- [ ] Preserve `mx` consent ordering.
- [ ] Audit destructive confirmations.
- [ ] Disable prompts in non-TTY/CI/structured modes.
- [ ] Add safe defaults, EOF, cancellation, and redaction tests.
- [ ] Produce manual accessibility evidence.

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
