---
name: git-archaeology
description: Investigate Mew or upstream history to recover why behavior, architecture, tests, or compatibility rules exist. Use when a regression crosses refactors, comments conflict with code, a decision appears stale, or a reference implementation changed over time.
---

# Git archaeology

Start from the current symbol, file, test, issue, or behavior. Use log, blame, commit inspection, linked issues, and release notes to identify the decision path.

Separate:

- historical fact
- current behavior
- superseded rationale
- unresolved question

Read discussion and closure reasons, not only issue titles or bodies. Reproduce current behavior before reviving an old plan. Report commit IDs and paths supporting each conclusion.
