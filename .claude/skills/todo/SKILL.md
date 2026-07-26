---
name: todo
description: Parse status-tagged task lines from a Markdown plan or agent thread and filter them by status or section. Use to inspect large Mew MVP plans, reconstruct open work after a context reset, or produce a machine-readable task slice for another agent.
---

# Todo status parser

Recognize these markers outside fenced code blocks:

```text
[ ] pending
[/] in progress
[x] done
[-] cancelled
[>] deferred
[?] blocked or awaiting an answer
```

Use `go run scripts/todo/main.go <file> [flags]`.

Supported flags should include `--pending`, `--in-progress`, `--done`, `--cancelled`, `--deferred`, `--blocked`, `--not-done`, `--section`, `--counts`, and `--json`.

Treat `[?]` as actionable and exclude deferred/cancelled from `--not-done`.
