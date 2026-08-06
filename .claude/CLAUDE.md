<!-- CODEGRAPH_START -->
## CodeGraph

This project has a CodeGraph MCP server configured, exposing a single tool: `codegraph_explore`. CodeGraph is a tree-sitter-parsed knowledge graph of every symbol, edge, and file. Reads are sub-millisecond and return structural information grep cannot.

### Use codegraph_explore instead of reading files

Reach for `codegraph_explore` before grep/find or Read for any **structural** question — how does X work, how does X reach Y, what calls what, where is X defined, or surveying an area. It takes a natural-language question or a bag of symbol/file names and returns the relevant symbols' **verbatim, line-numbered source** grouped by file (the same `<n>\t<line>` shape Read gives you, safe to Edit from), plus the call paths between them — including dynamic-dispatch hops (callbacks, React re-render, JSX children) grep can't follow — and a blast-radius summary of what depends on them. Name a file or symbol in the query to read its current source.

**Always pass `projectPath`.** Every `codegraph_explore` call must include `projectPath` set to the absolute path of the workspace root (or any directory inside the indexed project). Do not omit it — omitting relies on a session default that may be missing (monorepos, multi-root workspaces, or when the server root has no `.codegraph/` of its own).

### Rules of thumb

- **Answer directly — don't delegate exploration.** ONE `codegraph_explore` usually answers the whole question; follow up with another `codegraph_explore` naming more specific symbols if you need more. Codegraph IS the pre-built index, so spawning a separate file-reading sub-task/agent — or running a grep + read loop — repeats work codegraph already did and costs more for the same answer.
- **Trust codegraph results.** They come from a full AST parse. Do NOT re-verify them with grep — that's slower, less accurate, and wastes context.
- **Don't grep or Read first** to find or understand indexed code — one `codegraph_explore` returns the relevant source in a single round-trip. Reach for raw Read/Grep only to confirm a specific detail codegraph didn't cover, or for what it doesn't index (configs, docs).
- **Index lag — check the staleness banner, don't guess a wait.** When a codegraph response starts with "⚠️ Some files referenced below were edited since the last index sync…", the listed files are pending re-index — Read those specific files for accurate content. Files NOT in that banner are fresh and codegraph is authoritative for them.

### If `.codegraph/` doesn't exist

The MCP server returns "not initialized." Ask the user: *"I notice this project doesn't have CodeGraph initialized. Want me to run `codegraph init` to build the index?"*
<!-- CODEGRAPH_END -->

<!-- I-HAVE-ADHD_START -->
## i-have-adhd

The reader has ADHD. Output is not just brief. It is shaped so an ADHD brain can act on it.

### Persistence

These rules apply to every response for the rest of the session, not only this one. They do not expire after a few turns and they do not lapse when the topic changes. If you are unsure whether they still apply, they do.

Turn them off only when the reader says "stop adhd mode" or "normal mode". Confirm in one line, then return to your default style.

### What ADHD changes about reading

Five facts drive every rule below:

1. Working memory is small. Anything not on screen is forgotten. Do not ask the reader to "keep in mind X."
2. Knowing the answer is not doing the answer. The friction between "got it" and "done it" is where work dies.
3. Starting is the hardest step. The first action must be obvious, small, and doable now.
4. Time estimates feel uniform. "A bit of work" and "a few hours" register the same. Vague estimates fail.
5. Dopamine is scarce. Visible progress matters. Buried wins do not register.

### Rules

#### 1. Lead with the next action

The first line is something the reader can do. Not context. Not a plan. The action.

Bad: "Let's think about this. Your auth flow has a few moving pieces..."
Good: "Run `npm install jsonwebtoken`, then edit `src/auth.ts:42`."

If the answer is a command, path, or snippet, it goes first. Prose comes after, if at all.

#### 2. Number multi-step tasks

If the work takes more than one step, write a numbered list. Each step is one bounded action. No step contains "and then" twice.

Use the fewest steps that still work. Cut any step the reader does not need, and fold trivial steps into the one before. A short path finished beats a complete path abandoned.

Bad: "First open the file, find the function, swap it out, then run the tests."

Good:
```
1. Open `src/auth.ts`
2. Replace `verifyToken` (lines 42 to 58) with the snippet below
3. Run `npm test -- auth.spec.ts`
```

#### 3. End with one concrete next action

If anything is left open, name ONE thing the reader can do in under two minutes. Even "open the file" counts.

Bad: "Hope that helps. Let me know if you want to dig deeper."
Good: "Next: run `npm test` and paste the first failing line."

#### 4. Suppress tangents

If a second issue exists, finish the first, then offer the second as a separate question.

Bad: "Here's the fix. By the way, your dependency is also stale, and your README is out of date, and..."
Good: "Here's the fix. Separately: there is also a stale dependency. Want me to handle that next?"

A question that comes up mid-work is not a tangent: answer it yourself if you can and fold the result in. If it still needs the reader, surface it once, at the end.

#### 5. Restate state every turn

The reader cannot hold "we are on step 3 of 5" between messages. Restate it.

Bad: "Done. Ready for the next part?"
Good: "Step 3 of 5 done: schema updated. Next: backfill the new column. Run the script?"

If the harness has a task or plan tool, use it for multi-step work: one item per step, one in progress at a time. The checklist does the restating; do not also narrate the full plan as prose.

#### 6. Give specific time estimates

Vague estimates fail. Ballpark in concrete units.

Bad: "This will take some work."
Good: "About 15 minutes if tests already cover this. An afternoon if not."

#### 7. Make completed work visible

Show what now works, in concrete terms. Do not bury wins in a recap.

Bad: "I've made some changes to the auth flow. Among other things..."
Good: "Login now works with magic links. Try: `npm run dev`, open `/login`."

#### 8. Matter-of-fact tone for errors

Never use "Uh oh," "Oh no," or "There seems to be a problem." State cause and fix.

Bad: "Uh oh, the test is failing. There seems to be an issue..."
Good: "Test fails at `auth.spec.ts:42`: expected 200, got 401. Cause: missing auth header. Fix: add `Authorization: Bearer ${token}` to the request."

#### 9. Cap lists at 5 items

If a list grows past five, split into "do now" vs "later," or "must" vs "nice to have." Five items ranked beats ten unranked.

#### 10. No preamble, no recap, no closing pleasantries

Forbidden openers: "Great question," "Let me...", "I'll...", "Sure!", "Looking at your...", "To answer your question..."

Forbidden recaps after a completed task: "I've now done X, Y, and Z, which means..."

Forbidden closers: "Let me know if you need anything else," "Hope this helps," "Happy to clarify," "Feel free to ask."

Start with the answer. End when the answer is done.

### When to break the rules

Override the defaults when:

1. User asks to "explain" or "walk me through." Explain fully. Still no preamble, still no closer, but the body runs as long as the topic needs. Add headers so the reader can skim back.
2. Destructive action ahead (`rm -rf`, force push, schema migration, dropping a table). Confirm before acting. Safety wins over brevity.
3. Debug spiral. If the last three turns have been "still broken," stop iterating on code. Name the assumption that might be wrong. Ask one diagnostic question.
4. Real ambiguity in the request. One short clarifying question beats guessing and rewriting.
5. A rule fights the task. When a rule would delete the answer itself, the task wins; the shape stays. Example: "what are my options" gets 2 to 4 ranked options with one-line trade-offs, recommendation first, not one path. The options are the answer.
6. A rule fights the harness. Inside an agent harness, the system prompt outranks this skill: announce a tool call when the harness requires it, do the work instead of asking "want me to," point time estimates at whoever executes the steps. Same principle as 5: the constraint wins, the shape stays.

### Pre-send check

Before sending, delete:

1. The first sentence if it announces what you are about to do.
2. The last sentence if it asks "anything else?" or recaps what just happened.
3. Any "by the way" sidebar.
4. Any hedging adverb adding no information ("perhaps," "might," "could possibly"). Keep a hedge that carries real uncertainty; deleting it manufactures confidence.
5. Any idiom or figurative phrase ("circle back," "get the ball rolling," "on the same page"). Replace with the literal action.

Then verify: if the reader reads only the first line and the last line, do they know (a) what to do next, and (b) what just happened?

If yes, send.
<!-- I-HAVE-ADHD_END -->

## Project essentials

Module: `github.com/mewisme/mew` — Go 1.26.5+. Product: **MewJS** (short: **Mew**). Binaries: `m` (primary), `mx` (package runner). Lockfile: `m.lock` (native), `nub.lock` (compat). Behavioral reference: **Nub** (Go-native architecture, not a Rust port).

Authoritative architecture: [`docs/architecture/package-map.md`](docs/architecture/package-map.md). Key docs: `docs/charter.md`, `docs/engineering.md`, `docs/errors.md`, `docs/testing.md`, `docs/architecture/forbidden-imports.md`.

### Repository tools

[`TOOLS.md`](TOOLS.md) is the canonical inventory and usage guide for every repository-local tool. Read it before creating a script, composing a multi-command workflow, changing build/test/generation/validation tooling, modifying the Makefile or CI command paths, or claiming no existing command covers a task.

Prefer, in order:
1. Documented Make targets (`make help` lists them all)
2. Documented canonical Python or shell tools
3. Repository-native Go commands
4. Direct low-level command sequences only when no canonical tool exists

Do not reimplement behavior a documented tool already provides. Do not bypass repository wrappers when they add validation, portability, or deterministic output. Use documented check modes (`make generate-check`, `make assets-check`) before manually inspecting or rewriting generated files.

Any change that adds, removes, renames, or changes the behavior of a repository tool must update [`TOOLS.md`](TOOLS.md) in the same change.

### Scripts and generated content

When a file is produced by a documented script or generator (checklist, manifest, plan enrichment, runtime asset manifest, generated fixtures, conformance outputs), run the script to produce the update instead of hand-editing the generated output. If the script's output is inappropriate, incomplete, or incorrect after the run, update the script (or its input sources) to produce the correct content, then regenerate. Never manually patch a generated file while leaving the generator out of sync — the next regeneration will revert the hand-edit.

Check the generator exists and is documented before assuming a file is hand-maintained. Indicators that a file is generated: a `Regenerate:` or `Generated by` comment near the top, a `make` target that writes it, or a companion script in `tools/` or `plans/scripts/`.

### Build, test, lint

```powershell
# Build
CGO_ENABLED=0 go build -o bin/m.exe ./cmd/m
CGO_ENABLED=0 go build -o bin/mx.exe ./cmd/mx

# Test (hermetic — never hits public npm)
go test ./... -count=1                    # all hermetic tests
go test -tags crash ./tests/integration/... -run Crash -count=1 -timeout 30m

# Single package / test
go test ./internal/resolver/... -count=1
go test ./internal/resolver/... -run TestIncrementalDiff -count=1

# Vet + lint
go vet ./...
golangci-lint run ./...                   # config: .golangci.yml

# Race (requires CGO)
$env:CGO_ENABLED = "1"
go test -race ./internal/transaction/... ./internal/store/... ./internal/resolver/... -count=1

# Fuzz smoke
python3 tools/fuzz_smoke.py

# Conformance
go test ./tests/conformance/... -count=1

# Vulnerability + dependency allowlist
govulncheck ./...
go run ./tools/check-license
go run ./tools/check-deps
```

### Architecture

Four-layer dependency direction. Presentation must not own domain logic. Domain resolves a complete immutable graph before any mutation.

| Layer | Packages | Owns |
|---|---|---|
| Entry | `cmd/m`, `cmd/mx` | Process exit codes only. May only import `internal/app`, `internal/cli`, stdlib. |
| Presentation | `internal/cli`, `internal/app` | Parsing, dispatch, orchestration, user output |
| Domain | `manifest`, `project`, `workspace`, `registry`, `resolver`, `lockfile`, `graph`, `plan`, `snapshot`, `capsule`, `policy` | Read/plan models. Free of network/mutate. |
| Mutation | `fetch`, `archive`, `store`, `linker`, `transaction` | Staged filesystem changes |
| Runtime | `runner`, `process`, `runtime`, `transform`, `node` | Execution and Node launch. Free of PM engine. |

Key import rules enforced by `internal/archcheck`:
- `internal/resolver` must not import `linker`, `transaction`, `fetch`, `store` (resolve-complete-before-mutate)
- Domain packages must not import `internal/presentation` or Charm (`charm.land/*`, `github.com/charmbracelet/*`)
- `internal/config`, `internal/project` stay free of mutate path
- `internal/graph`, `plan`, `snapshot`, `manifest`, `policy`, `capsule` stay free of network/mutate

Full rules: [`docs/architecture/forbidden-imports.md`](docs/architecture/forbidden-imports.md).

### Transaction boundary

Every install-family mutation follows this pipeline. Previous manifest, lockfile, and `node_modules` remain usable until commit. On failure before `committed`, rollback restores pre-mutation state.

```
inspect → resolve → plan → fetch → verify → stage → validate → plan journal
  → backup → commit (all live publishes) → post-commit cleanup
  ↘ rollback on failure (before committed)
```

Single mutation entrypoint: `BeginMutation` acquires project lock at `.mew/txn/lock`, runs idempotent recovery, refuses to begin when incomplete journals remain. `BeginMutationSession` wraps this for install-family commands — live reads only after lock acquisition.

Full docs: [`docs/architecture/transaction-boundary.md`](docs/architecture/transaction-boundary.md).

### Error codes

Pattern: `ERR_M_<DOMAIN>_<DETAIL>`. Every public failure returns `*apperr.Error` (or wraps into one at CLI boundary). Package: `internal/apperr` (`New`, `Wrap`, `CodeOf`, `ExitCode`). Stable codes: `ERR_M_USAGE` (2), `ERR_M_CANCELLED` (130), `ERR_M_TRANSACTION`, `ERR_M_INTEGRITY`, `ERR_M_RESOLVE`, `ERR_M_LOCKFILE`, `ERR_M_STORE`, `ERR_M_POLICY`, etc. Unknown codes → exit 1. Full registry: [`docs/errors.md`](docs/errors.md).

### Engineering conventions

- **Hermetic tests only.** Never hit public npm. Fixture registry at `fixtures/registry/v1/` with SHA-256 manifest. `testkit.CleanEnv`/`TempHome` isolates home dirs.
- **errcheck** on all resource cleanup. `defer func() { _ = f.Close() }()`. `fmt.Print*` excluded (broken pipe not recoverable).
- **Dependency allowlist** at `tools/allowlist/modules.txt` — update in same PR as new deps. Prefer stdlib.
- **Tool versions** pinned in `tools/versions.env`. No floating `latest` in CI.
- **Experimental features** behind `MEW_EXPERIMENTAL_<NAME>=1` or `--experimental-<name>` flags.
- **Crash tests** use `crash` build tag, excluded from default `go test ./...`. Use `-tags crash`.
- **CGO_ENABLED=0** for production builds. Race tests are the only CGO exception.
- **Fixtures** are source-of-truth. Never mutate checked-in fixtures from tests — copy via `testkit.CopyFixture`.

## Serena usage

Use Serena automatically when semantic code understanding is beneficial.

Prefer Serena for:
- Finding symbols, definitions, implementations, and references
- Understanding relationships across multiple files
- Cross-file refactoring and symbol renaming
- Replacing or rewriting complete function, class, or method bodies
- Navigating large or unfamiliar codebases

Prefer Claude Code built-in tools for:
- Small edits in a known file
- Configuration, documentation, JSON, YAML, and Markdown
- Plain text or regex searches
- Shell commands, Git, builds, and tests

Do not use Serena unnecessarily for trivial text edits.
