---
name: plan-thread
description: Design a Mew change before implementation by resolving architecture, behavior, security, compatibility, and public-surface questions. Use when the approach is not settled, when the user asks for a plan or design, or when an implementation task exposes a real decision fork.
---

# Plan thread

Create `.agents/threads/<slug>.md` before dispatching work. Set `status: planning` and record the goal, constraints, open questions, and evidence required.

## Workflow

1. Map the actual code and current behavior.
2. Build minimal differential fixtures when compatibility is involved.
3. List viable approaches and concrete trade-offs.
4. Surface human-owned decisions instead of choosing silently.
5. Move answered questions into `Decisions` immediately.
6. Record the selected architecture, interfaces, data flow, migration, tests, and rollback.
7. Change status to `planned` when no design question remains.
8. Write one implementation-ready `Next step`.

A plan thread does not land production code. Small throwaway probes are allowed when they answer a design question.
