---
name: UX-0007 Advanced Help, Pager, and Markdown
overview: Add optional long-form terminal documentation, topic help, pager selection, width-aware Markdown rendering, safe hyperlink policy, and richer explanatory surfaces without changing ordinary Cobra help or making a pager/ANSI output mandatory.
todos:
  - id: p1-topics
    content: Define help-topic registry, ownership, source files, and command grammar
    status: pending
  - id: p2-glamour
    content: Evaluate and pin Glamour v2 for width-aware Markdown rendering
    status: pending
  - id: p3-pager
    content: Implement deterministic pager resolution, safe process launch, non-TTY behavior, and cancellation
    status: pending
  - id: p4-content
    content: Add curated topics for errors, compatibility, lifecycle trust, snapshots/capsules, runner execution, and migration
    status: pending
  - id: p5-links
    content: Define hyperlink and plain URL policy by terminal capability
    status: pending
  - id: p6-tests
    content: Add Markdown, width, pager, pipe, no-color, Windows, broken-pipe, and link validation tests
    status: pending
isProject: false
---

# UX-0007 — Advanced Help, Pager, and Markdown

## Goal

Provide deeper terminal-native documentation for workflows that are too complex for concise Cobra help or one-line error hints.

This is an optional layer. Ordinary `m --help` and `m <command> --help` remain fast, concise, and pager-free by default unless project policy explicitly changes.

## User-visible surface

Proposed grammar:

```text
m help <topic>
m help errors <ERR_M_CODE>
m help compatibility
m help lifecycle-trust
m help snapshots
m help capsules
m help runner
```

Exact grammar must avoid collision with Cobra's existing help command and aliases.

Potential error hint:

```text
Run `m help errors ERR_M_LOCKFILE` for details.
```

## Explicitly out of scope

- Replacing all Cobra help with Markdown.
- Opening a browser automatically.
- Requiring a pager.
- Rendering remote documentation fetched from the network.
- Adding a documentation server.
- Interactive search UI.
- Making hyperlinks necessary to understand output.
- Adding long-form content that duplicates authoritative docs without an ownership policy.

## Dependency

Optional:

```go
import "charm.land/glamour/v2"
```

Before adoption:

- exact version and license review;
- binary-size/startup impact;
- Markdown feature/security review;
- ANSI/no-color behavior;
- width behavior;
- Windows behavior;
- dependency overlap with Lip Gloss.

If dependency cost is excessive, implement a minimal renderer or keep plain Markdown. This plan must not force Glamour adoption merely because it is available.

## Content architecture

```text
docs/terminal-help/
  errors/
    ERR_M_LOCKFILE.md
    ERR_M_INTEGRITY.md
  compatibility.md
  lifecycle-trust.md
  snapshots.md
  capsules.md
  runner.md
  configuration.md
```

Or generate topic content from existing authoritative docs. Do not maintain conflicting copies without a synchronization rule.

## Topic registry

```go
type HelpTopic struct {
    ID          string
    Title       string
    Summary     string
    SourcePath  string
    Aliases     []string
    Related     []string
    Experimental bool
}
```

Validation:

- unique IDs and aliases;
- source path inside repository/embed root;
- no orphan topic file;
- deterministic ordering;
- valid related-topic references;
- link validation.

## Embedded vs filesystem content

Decide and document one model:

### Embedded

Advantages:

- works in installed binary;
- version-matched content;
- no external files.

Costs:

- binary size;
- build-time embedding rules.

### Filesystem

Not suitable as the only production source unless installers guarantee files. May be allowed in development override mode.

Recommended: embed curated terminal help and link to repository docs for full detail.

## Markdown rendering policy

- Render only trusted project-authored Markdown.
- Disable or sanitize unsupported raw HTML.
- Wrap to terminal width.
- Respect no-color and accessible mode.
- Do not render images.
- Code blocks remain copyable.
- Tables must fall back cleanly at narrow widths.
- Headings remain understandable in plain output.
- Links display their destination when terminal hyperlinks are unavailable.

## Pager policy

### Modes

```text
--pager auto|always|never
MEW_PAGER
PAGER
```

Recommended precedence:

1. CLI flag.
2. `MEW_PAGER`.
3. Mew config.
4. `PAGER`.
5. platform default or no pager.

### Auto pager requires

- stdout TTY;
- human mode;
- content exceeds a threshold;
- not CI;
- not accessible mode unless pager is explicitly accessible;
- executable resolves safely.

### Safety

- Avoid shell command strings when possible.
- If supporting pager arguments, parse through a documented safe mechanism.
- Do not inject untrusted topic content into command arguments.
- Pass content through stdin.
- Clear or preserve environment variables deliberately.
- Handle pager missing/failure by writing directly in auto mode.
- Forced pager failure returns typed I/O error.

## Platform defaults

Do not assume `less` exists on Windows.

Possible policy:

- use configured pager when present;
- no built-in default on unsupported platforms;
- direct output remains complete;
- never download a pager.

## Hyperlink policy

Terminal hyperlinks are optional enhancement.

- Detect support conservatively.
- Render visible text and plain URL fallback.
- Never hide the destination entirely.
- Do not create authenticated or secret-bearing URLs.
- Relative docs references resolve to public repository/documentation URLs only when the project has an authoritative base URL.
- No hyperlinks in plain/no-color/structured modes unless schema explicitly contains URLs.

## Error help topics

A topic should contain:

```text
Meaning
Common causes
How to diagnose
Safe recovery steps
Related commands
Related configuration
When to report a bug
```

Do not suggest deleting lockfiles, caches, stores, or transaction state casually.

## Compatibility topic

Explain independent axes:

```text
lockfile detection
lockfile read
byte-preserving behavior
semantic mutation
installation support
linker layout
CLI compatibility
runner behavior
platform evidence
```

Do not overclaim compatibility with other package managers.

## Lifecycle-trust topic

Explain:

- trust modes;
- project-local trust store;
- allow/deny/ask semantics;
- non-TTY behavior;
- review commands;
- best-effort sandbox limitations;
- audit implications.

## Runner topic

Explain:

- `m run` vs `m exec` vs `mx`;
- project, temporary, snapshot, and capsule sources;
- stdout/stderr behavior;
- signals and exit codes;
- no-network guarantees where applicable;
- experimental shortcuts.

## Implementation phases

### Phase 1 — Content and grammar audit

Select high-value topics and resolve Cobra help-command integration.

### Phase 2 — Topic registry and embedding

Implement validation and embedded loading.

### Phase 3 — Plain renderer

Ship plain width-aware Markdown/topic output first.

### Phase 4 — Glamour renderer

Add optional styled rendering under human capabilities.

### Phase 5 — Pager

Implement auto/forced/disabled behavior and safe process launch.

### Phase 6 — Topic content

Write curated content and link to authoritative docs.

### Phase 7 — Certification

Width, no-color, accessibility, pager failure, Windows, broken pipe, and link checks.

## Tests

### Unit

- topic registry validation;
- alias resolution;
- embed path guard;
- Markdown sanitization;
- pager resolution precedence;
- content length threshold;
- hyperlink fallback;
- no-color rendering.

### Integration

- `m help <topic>` in TTY and pipe;
- pager available/missing;
- pager exits early;
- broken pipe;
- Windows no-pager fallback;
- accessible mode;
- width 40/80/120;
- no network access;
- error topic link from actual error output.

### Documentation validation

- all relative links exist;
- topic IDs unique;
- command examples parse;
- referenced error codes exist;
- no stale feature-status claims;
- embedded files included in build.

## Acceptance criteria

- Long-form help works offline.
- Ordinary command help stays concise and fast.
- Pager is optional and safe.
- No-color and accessible output are readable.
- Windows works without assuming Unix tools.
- Topic content is authoritative and version-matched.
- No secret-bearing links.
- Glamour remains isolated to help presentation.

## Risks

| Risk | Mitigation |
|---|---|
| Documentation duplicates and drifts | clear source ownership and validation |
| Pager command injection | direct exec and constrained argument parsing |
| Binary size grows materially | curated topics and measured dependency impact |
| Markdown output is noisy in narrow terminals | width-aware plain fallback |
| Error hints point to missing topics | registry-backed tests |
| Windows lacks pager | direct output fallback |

## Estimated effort

**10–18 focused engineering hours**, including content work.

## References

- Glamour v2: https://github.com/charmbracelet/glamour
