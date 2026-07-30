# Sources and planning assumptions

## Repository sources reviewed

The plan package was prepared from the current Mew repository architecture and the conversation's prior repository review, including these areas:

- `go.mod`
- `internal/diagnostics`
- `internal/cli`
- `internal/app`
- `internal/transaction`
- `internal/runner`
- `internal/process`
- `internal/lifecycle`
- `plans/INDEX.md`
- `plans/CHECKLIST.md`

Implementation agents must re-audit current `main` before coding because repository state may advance after this package was generated.

## Official Charm references

- Charm libraries overview: https://charm.land/libs/
- Bubble Tea: https://charm.land/bubbletea
- Bubble Tea, Lip Gloss, and Bubbles v2 release overview: https://charm.land/blog/v2/
- Lip Gloss: https://github.com/charmbracelet/lipgloss
- Bubbles: https://github.com/charmbracelet/bubbles
- Huh: https://charm.land/huh
- Glamour: https://github.com/charmbracelet/glamour

## Proposed dependency imports

```text
charm.land/bubbletea/v2
charm.land/bubbles/v2
charm.land/lipgloss/v2
charm.land/huh/v2
charm.land/glamour/v2
```

Exact versions are deliberately not frozen in these plans. Each implementation plan requires a dependency review and exact pin based on the repository state at implementation time.

## Important assumptions

- `UX-0001` through `UX-0008` form an independent numbering namespace.
- These files live under `plans/ux/` (standalone UX program directory).
- The root repository roadmap may link to this program, but does not own its numbering or checklist.
- Existing machine-output schemas and error mappings remain authoritative.
- Existing runner stabilization and process behavior remain prerequisites.
- Interactive script selection remains deferred.
