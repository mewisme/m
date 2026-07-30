# MVP 0046 runner certification evidence

Committed evidence supplements CI when a platform report cannot be produced in
GitHub Actions. **CI-certified** reports take precedence over local evidence.

## When to add local evidence

- macOS runner unavailable in CI for a release window
- Reproducing a platform-specific certification fix before CI merge

Set `MEW_RUNNER_LOCAL_EVIDENCE=1` when aggregating a locally captured macOS
report so `m conformance verify runner` records `locally-certified` instead of
`ci-certified` for that platform.

## Required fields per evidence file

| Field | Example |
|---|---|
| Commit SHA | full 40-character git SHA |
| Manifest digest | from report `manifestDigest` |
| Waiver digest | from report `waiverManifestDigest` |
| OS / arch | `darwin` / `arm64` |
| Go version | `go1.26.x` |
| Node version | `v22.x.x` |
| Command | exact `m conformance run runner` invocation |
| Timestamp | ISO-8601 UTC |
| Report path | committed JSON filename |
| Report SHA-256 | hash of committed report bytes |

## Template

Create `darwin-local-<date>.md` beside the report JSON:

```markdown
# macOS local runner certification

- commit: `<sha>`
- manifestDigest: `<64-hex>`
- waiverManifestDigest: `<64-hex>`
- os/arch: darwin/arm64
- go: go1.26.x
- node: v22.x.x
- command: `go run ./cmd/m conformance run runner --json --output darwin-runner-report.json --force`
- recordedAt: 2026-07-31T00:00:00Z
- report: darwin-runner-report.json
- reportSha256: `<64-hex>`
- certification: locally-certified
```

Do not commit dirty-worktree reports (`revision.dirty: true` fails aggregation).
