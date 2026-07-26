# Architecture Decision Records

Mew uses lightweight ADRs for **irreversible or high-cost** decisions. Create an ADR **before** designing or shipping:

- Persistent file formats (`m.lock`, adapter extensions, cache layouts)
- Public configuration keys and environment variables
- Compatibility promises and semantic versioning boundaries
- Security or trust posture changes
- Default behavior that affects existing projects

## Process

1. Copy [`0000-template.md`](0000-template.md) to `NNNN-short-title.md` (next sequential number).
2. Fill all sections; mark status **Proposed**.
3. Review in the implementing PR; link the ADR from the PR body.
4. On merge, set status **Accepted**.
5. If superseded, set old ADR to **Superseded by NNNN** — do not delete.

## Numbering

- `0000` — template only (never accepted)
- `0001+` — accepted decisions in chronological order

## Relationship to plans

| Document | Role |
|---|---|
| `plans/00xx-*.md` | Implementation contract and task breakdown |
| `docs/charter.md` | Stable product contract |
| `docs/adr/*.md` | Point-in-time decisions with context |

Plans may reference ADRs; ADRs must not silently expand plan scope.

## Status values

| Status | Meaning |
|---|---|
| Proposed | Under review; do not implement irreversible parts |
| Accepted | Decision is active |
| Deprecated | Still in force but scheduled for removal |
| Superseded | Replaced by a newer ADR |
