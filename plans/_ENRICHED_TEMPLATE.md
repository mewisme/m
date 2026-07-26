# Enriched MVP Plan Template

This document defines the required structure for every `plans/00xx-*.md` file after detail expansion.
Do not treat this file as an implementable MVP.

## Required section order

1. Title (`# NNNN — …`)
2. Document Control
3. Objective
4. Sequence and Dependencies
5. Nub Reference Mapping
6. User-Visible Surface
7. In Scope
8. Explicit Non-Goals
9. Architecture and Interfaces
10. **Feature Inventory Links** *(enriched)*
11. **Go Package Map** *(enriched)*
12. **Data Flow** *(enriched; mermaid when useful)*
13. **Commands and Flags** *(enriched; N/A allowed)*
14. **Persistent Artifacts** *(enriched; N/A allowed)*
15. Detailed Implementation Checklist *(expanded 15–35 items, grouped)*
16. Test Plan *(MVP-specific items + standard layers)*
17. **Concrete Test Fixtures** *(enriched)*
18. **Acceptance Scenarios** *(enriched)*
19. **Nub Conformance Targets** *(enriched)*
20. Performance Requirements
21. Security and Trust Requirements
22. Risks and Mitigations
23. **Open Decisions** *(enriched)*
24. Deliverables
25. Exit Criteria
26. AI-Agent Handoff Contract
27. Optional MVP-specific appendix tables (inventory matrices, migration maps, etc.)

## Size targets

| Class | IDs | Target lines |
|---|---|---|
| Foundation | 0001–0009 | 250–400 |
| Core PM | 0010–0031 | 300–500 |
| Runners / runtime / managers / product | 0040–0074 | 250–450 |
| Cross-cutting | 0080–0090 | 200–350 |

## De-duplication

- Keep one copy of generic test-layer and security boilerplate.
- Prefer links to `0087-definition-of-done.md` for shared exit standards.
- Every MVP must still carry MVP-specific checklist, fixtures, and acceptance scenarios.

## Markers for generators

Enrichment blocks are bounded by HTML comments:

```markdown
<!-- ENRICHMENT:BEGIN -->
...
<!-- ENRICHMENT:END -->
```

Checklist expansion replaces the short checklist between:

```markdown
## Detailed Implementation Checklist
...
## Test Plan
```

Regenerate with:

```powershell
.\plans\scripts\enrich-plans.ps1
.\plans\scripts\generate-cursor-plans.ps1
.\plans\scripts\generate-checklist.ps1
.\plans\scripts\update-manifest.ps1
```
