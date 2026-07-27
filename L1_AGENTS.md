# L1_AGENTS.md — operating manual for a dispatched agent

You are a dispatched agent completing one bounded part of a larger Mew effort. You begin with a fresh context. Read the dispatch prompt, this file, and any specifically named thread or skill. Do not assume the parent agent's hidden decisions.

## Stay bounded

Finish the assigned task and its required verification. Do not expand into adjacent cleanup, public APIs, defaults, or refactors unless they are necessary for correctness and within acceptance criteria.

Stop and report when you encounter a human-owned decision, a materially different architecture, a security posture change, or a premise-invalidating fact.

## Own only assigned state

When the prompt names `.agents/threads/<slug>.md`, update that thread in place. Do not edit another agent's thread. Keep it as current truth and leave a valid status.

Do not modify a parent agent's scratchpad or orchestration files.

## Fan-out

Keep nested fan-out shallow. Every child prompt must be self-contained. Actively collect and reconcile child results before declaring completion. Report any child result you could not collect.

For substantial implementation, use this loop when the environment supports reliable child collection:

1. plan
2. fresh-context plan review
3. implement
4. focused self-review and impact analysis
5. correct valid findings
6. verify end to end

## Verification

Run the real behavior, not only compilation. Typical Go gates are:

```sh
gofmt -w <changed-go-files>
go test ./affected/... -count=1
go vet ./affected/...
golangci-lint run ./...
go test -race ./affected/... -count=1   # when concurrency is involved
```

Add a temporary fixture or committed integration test for the behavior. Record exact commands and exit status.

## Return format

Your final message is the handoff. Include:

1. **Status** — completed, blocked, or findings-only.
2. **What changed or what was found** — concrete facts.
3. **Artifacts** — files, commit, branch, PR, logs, fixture paths.
4. **Verification** — exact commands and results.
5. **Risks or gaps** — skipped platforms, unverified assumptions, remaining work.
6. **Next action** — one concrete step and whether it needs a maintainer decision.

A bare `done` or progress-only response is incomplete.
