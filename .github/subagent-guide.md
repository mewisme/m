# Dispatched sub-agent operating guide

You are completing one bounded piece of a larger Mew effort. The dispatch prompt and this guide define your contract.

## Return value

Your final response must contain status, concrete work, artifacts, exact verification, caveats, and one next action. It is an orchestration payload, not a conversational sign-off.

## Decisions

Report facts and recommendations. Do not unilaterally choose a public default, security posture, compatibility promise, file format, API, configuration key, or environment variable unless the prompt explicitly authorizes that decision.

## Worktree discipline

Use an isolated Git worktree based on the current default branch for substantive work. Do not reset, stash, or switch branches in a shared working tree. Stage only owned files. Never force-push the default branch.

## Local verification before push

Run the relevant sequence:

```sh
gofmt -w <changed-go-files>
go test ./affected/... -count=1
go vet ./affected/...
go test -race ./affected/... -count=1   # concurrency-sensitive work
bash scripts/check-agent-guidance.sh    # guidance changes
```

Exercise the actual `m` or `mx` behavior in a temporary fixture. Use an isolated HOME/XDG/cache directory for tests involving global state. Use Windows CI or a Windows VM for Windows process and shim behavior.

## Handoff

Do not wait indefinitely for CI inside a bounded sub-agent. Push only after local verification, report the commit and CI state, and return control to the orchestrator unless the dispatch explicitly owns CI triage.

Do not self-merge a substantive PR without explicit authorization.
