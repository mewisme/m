# L1_AGENTS.md — operating manual for a dispatched Mew agent

You own one bounded part of a larger Mew effort. Read [`AGENTS.md`](AGENTS.md), the dispatch prompt, and every named thread, issue, plan, or document. `AGENTS.md` contains the repository-local Ponytail, CodeGraph, Git commit, and ADHD rules injected from `.cursor/rules`; apply those exact rules and do not substitute externally fetched variants.

## 1. Start with current state

1. Confirm the current HEAD, branch/worktree, `git status`, and owned diff.
2. Restate the assigned goal, acceptance criteria, and files you may change.
3. Read the owning package, its callers, tests, and relevant architecture/domain docs.
4. When `codegraph_explore` is available, call it first for structural or flow work and always pass `projectPath` as required by `AGENTS.md`.
5. Stop immediately if the dispatch premise is false or requires an unauthorized public/default/security decision.

Report progress in a stateful form such as: `Step 2 of 4 complete: affected callers identified.`

## 2. Stay bounded and minimal

Use the Ponytail ladder after tracing the real flow: avoid unnecessary work, reuse repository code, prefer standard-library or native behavior, reuse approved dependencies, then write the smallest correct diff.

Fix the root cause at the owning boundary. Do not patch only the named symptom when sibling callers share the same defect.

Do not expand into adjacent cleanup, speculative abstractions, new dependencies, public APIs, config, environment variables, or defaults unless correctness and the acceptance criteria require it.

Do not weaken transaction safety, integrity verification, archive validation, lifecycle trust, rollback, redaction, structured-output contracts, or accessibility.

## 3. Worktree and commit discipline

- Use an isolated worktree for substantive implementation when the environment supports it. Never reset, stash, or switch branches in another agent's working tree.
- Stage only owned files. Inspect `git status --porcelain`, `git diff`, and `git diff --staged` before committing.
- Apply the exact Git commit rules injected into `AGENTS.md`: Conventional Commit title, imperative mood, 72-character hard limit, non-redundant bullet body, and final-state wording for amends.
- Never bypass hooks, change Git config, force-push the default branch, add agent co-author trailers, or commit secrets, credentials, local paths, conversation text, generated logs, or unrelated changes.
- Do not commit, push, open a PR, merge, amend, or alter remote state unless the dispatch authorizes that action.

## 4. Verification loop

1. Run the real behavior in a temporary fixture, not only compilation.
2. Run focused tests and vet for affected packages.
3. Run full applicable gates before claiming completion.
4. Review the final diff, callers, shutdown/error paths, docs, and structured output.
5. Record exact commands and outcomes; state skipped platforms or checks explicitly.

Typical commands:

```sh
gofmt -w <changed-go-files>
go test ./affected/... -count=1
go vet ./affected/...

go test ./... -count=1
go vet ./...
make lint
make allowlist
make build
```

Add `make race` for concurrency-sensitive work, `make vuln` for dependency/security changes, and the relevant conformance/certification command. Use isolated HOME/XDG/cache state. Validate Windows-specific behavior on Windows.

## 5. Handoff format

Your final response is an orchestration payload, not a sign-off. Apply the ADHD output rules injected into `AGENTS.md` and use at most five sections:

1. **Status** — completed, blocked, or findings-only; state the current step.
2. **Result** — what changed or what was discovered, including behavior now working.
3. **Artifacts** — files, branch, commit, PR, fixtures, or logs actually produced.
4. **Verification** — exact commands, results, and skipped checks/platforms.
5. **Next action** — one concrete action; name any maintainer decision required.

Lead with the status or blocker. For an error, state location, cause, and fix. Do not add a preamble, tangent, recap, vague estimate, or closing pleasantry. A bare `done` is incomplete.
