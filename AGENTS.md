# AGENTS.md — canonical agent guidance for Mew

This is the canonical entrypoint for AI coding agents working in the Mew repository. Read it before editing. A dispatched agent must then read [`L1_AGENTS.md`](L1_AGENTS.md) and any named thread, plan, issue, or skill.

An untracked `AGENTS.local.md` may add maintainer-local paths or tools. It must not weaken repository rules.

Before starting, confirm the current default-branch HEAD, `git status`, and the diff you actually own. Do not assume a prompt, prior conversation, or cached index matches the checkout.

## 1. Read the product and the code

1. Read [`docs/charter.md`](docs/charter.md), [`docs/naming.md`](docs/naming.md), and [`docs/compatibility-axes.md`](docs/compatibility-axes.md).
2. Read [`docs/architecture/README.md`](docs/architecture/README.md), [`package-map.md`](docs/architecture/package-map.md), and [`forbidden-imports.md`](docs/architecture/forbidden-imports.md).
3. Read [`docs/engineering.md`](docs/engineering.md), [`docs/testing.md`](docs/testing.md), [`docs/errors.md`](docs/errors.md), and the domain document for the task.
4. Check [`features/inventory.json`](features/inventory.json) before changing capability status or claims.
5. Treat current source and tests as truth. Plans and trackers may lag implementation.

## 2. Product and architecture contract

- **MewJS** is the product; **Mew** is the short name. `m` is the primary binary, `mx` is the package executable runner, and `m.lock` is the native lockfile.
- The shipped control plane is Go. Mew runs stock Node.js and may augment it through supported loader, preload, environment, addon, or command-line surfaces. Do not fork or patch Node.
- Keep `cmd/m` and `cmd/mx` thin. Parsing and dispatch live in `internal/cli`; orchestration lives in `internal/app`; domain and mutation logic stay in their owning packages.
- Resolve a complete immutable graph before filesystem mutation. Install-family changes follow `inspect -> resolve -> plan -> fetch -> verify -> stage -> validate -> commit`, with rollback on failure.
- Preserve incumbent package-manager and lockfile identity unless migration is explicit. Lossy conversion must be explicit, reported, and tested.

The enforced dependency direction is documented in [`docs/architecture/forbidden-imports.md`](docs/architecture/forbidden-imports.md). Domain packages must not import `internal/presentation`, `charm.land/*`, or `github.com/charmbracelet/*`.

## 3. Current CLI presentation contract

- Human output defaults to `rich`. Public output modes are exactly `rich`, `plain`, `json`, `ndjson`, and `silent`.
- Presentation mode is controlled by CLI flags. Do not restore output auto-selection, legacy presentation, or environment/config overrides for output, color, progress, Unicode, accessibility, or summary behavior.
- Canonical controls are `--output`, `--no-color`, `--no-progress`, `--ascii`, `--accessible`, and `--no-summary`.
- `ui.markdown_theme` is a user-scoped Markdown preference with default `dark` and values `dark`, `light`, `dracula`, `tokyo-night`, and `notty`. A project config must not force it. `GLAMOUR_STYLE` must not override Mew.
- `stdout` owns requested data, structured output, completion scripts, and child stdout. `stderr` owns human progress, notices, prompts, errors, and summaries. Structured output must remain ANSI-free.

See [`docs/architecture/cli-presentation.md`](docs/architecture/cli-presentation.md). Presentation libraries stay inside `internal/presentation` and its `help`, `pager`, and `prompt` adapters.

## 4. Non-negotiables

- Keep changes inside the assigned scope and acceptance criteria. Stop before choosing a new public default, security posture, compatibility promise, file format, config key, environment variable, or major architecture without authorization.
- Never partially mutate `package.json`, a lockfile, or `node_modules`. Do not weaken integrity checks, archive validation, lifecycle policy, transaction recovery, or rollback to make a test pass.
- Never claim compatibility, parity, performance, security, or completion without a fixture, exact command, test result, benchmark, or source reference.
- Do not commit secrets, credentials, private URLs, user conversation text, machine-local paths, generated logs, or unreviewed binary artifacts.
- Prefer stable `ERR_M_*` errors, deterministic ordering, bounded concurrency, cancellation through `context.Context`, and explicit ownership of goroutines and channels.

## 5. Repository-local operating rules

The following rule bodies are injected from this repository at
`main` HEAD `7ec58635e569c11caf1a909d0ac36defdc410d6b`.
They are copied from `.cursor/rules` only; no external version is a
source of truth. Cursor YAML frontmatter is omitted because this file is
not a Cursor rule file. When a project rule conflicts with Mew product,
security, architecture, or task-specific instructions, the stricter
higher-level Mew rule wins.

### 5.1 `ponytail`

Source: [`.cursor/rules/ponytail.mdc`](.cursor/rules/ponytail.mdc) — blob `b17938c08d1e626215ea276f99d938c736bad5a6`.

<!-- PONYTAIL_START -->
# Ponytail, lazy senior dev mode

You are a lazy senior developer. Lazy means efficient, not careless. The best code is the code never written.

Before writing any code, stop at the first rung that holds:

1. Does this need to be built at all? (YAGNI)
2. Does it already exist in this codebase? Reuse the helper, util, or pattern that's already here, don't re-write it.
3. Does the standard library already do this? Use it.
4. Does a native platform feature cover it? Use it.
5. Does an already-installed dependency solve it? Use it.
6. Can this be one line? Make it one line.
7. Only then: write the minimum code that works.

The ladder runs after you understand the problem, not instead of it: read the task and the code it touches, trace the real flow end to end, then climb.

Bug fix = root cause, not symptom: a report names a symptom. Grep every caller of the function you touch and fix the shared function once — one guard there is a smaller diff than one per caller, and patching only the path the ticket names leaves a sibling caller still broken.

Rules:

- No abstractions that weren't explicitly requested.
- No new dependency if it can be avoided.
- No boilerplate nobody asked for.
- Deletion over addition. Boring over clever. Fewest files possible.
- Shortest working diff wins, but only once you understand the problem. The smallest change in the wrong place isn't lazy, it's a second bug.
- Question complex requests: "Do you actually need X, or does Y cover it?"
- Pick the edge-case-correct option when two stdlib approaches are the same size, lazy means less code, not the flimsier algorithm.
- Mark deliberate simplifications that cut a real corner with a known ceiling (global lock, O(n²) scan, naive heuristic) with a `ponytail:` comment naming the ceiling and upgrade path.

Not lazy about: understanding the problem (read it fully and trace the real flow before picking a rung, a small diff you don't understand is just laziness dressed up as efficiency), input validation at trust boundaries, error handling that prevents data loss, security, accessibility, the calibration real hardware needs (the platform is never the spec ideal, a clock drifts, a sensor reads off), anything explicitly requested. Lazy code without its check is unfinished: non-trivial logic leaves ONE runnable check behind, the smallest thing that fails if the logic breaks (an assert-based demo/self-check or one small test file; no frameworks, no fixtures). Trivial one-liners need no test.
<!-- PONYTAIL_END -->

### 5.2 `codegraph`

Source: [`.cursor/rules/codegraph.mdc`](.cursor/rules/codegraph.mdc) — blob `6307ab259a45fcd90300ee50deef0db2e2a53562`.

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

### 5.3 `git-commit`

Source: [`.cursor/rules/git-commit.mdc`](.cursor/rules/git-commit.mdc) — blob `7ac26f49ca246b58259fa6e736805010265fc84e`.

# Git Commit Message Generation Rules

This document outlines the strict rules for generating Git commit messages. These rules ensure history is readable,
automated tools can parse changes, and the intent of every change is clear.

***

## 1. Format Specification

- **Standard**: Follow the **Conventional Commits** specification.
- Format: `type(scope): description`
- Common Types: `feat`,`fix`,`docs`,`style`,`refactor`,`test`,`chore`.
- **Plain Text Only**: The final commit message must be **Plain Text**.
- **No Markdown**: Do not use bold (`**`), italics (`*`), or code blocks (`` ` ``).
- **Allowed Symbols**: You MAY use standard punctuation and symbols for clarity (e.g., `-`,`:`,`()`,`[]`).
***

### 2. Title Line (Header)

The first line is the most important part of the commit message.
- **Content**: A strong, concise summary of the *entire* change.
- **Context**: Include critical contextual information (e.g., dates for meetings, releases, or time-sensitive changes).
- **Length**:
- **Ideal**: < 50 characters.
- **Hard Limit**: 72 characters.
- **Case**: Use the **Imperative Mood** (e.g., "add feature" not "added feature" or "adds feature").
- **Punctuation**: Do not end the title line with a period.
- **Reference Style**: When referring to documentation sections, use the **Section Title** instead of the Section
    Number (e.g., use "Refine Migration, Verification & Secrets protocol" instead of "Update Section 9").

***

### 3. Body (Description)

The body provides the detailed context.
- **Separation**: There must be a **blank line** between the Title and the Body.
- **Structure**: Use **Bullet Points** (`-`) for all details. Do not use paragraphs of text.
- **Wrapping**: Wrap all lines at **72 characters**.
- **Content**:
- Focus on **What** changed and **Why**.
- Summarize the impact of the changes.
- **No Redundancy**: Do not repeat information already stated in the title. Provide supplementary details ONLY.
- **Meaningful Summaries**: Avoid exhaustive lists for repetitive changes (e.g., individual spell check words).
    Instead, provide a higher-level summary of the domain or impact (e.g., "Add 30+ project domain terms to workspace
    dictionary").
- **Self-Documenting Titles**: For standard style cleanups or simple deletions, the body MAY be omitted entirely if the
    title is exhaustive. Avoid adding "filler" or "gibberish" to satisfy a structure requirement.
***

### 4. Example

**Correct Format:**
```text
feat(auth): implement JWT token refresh strategy

Update the authentication flow to handle expired tokens automatically
without forcing a user logout.

- Add 'RefreshToken' service to handle token rotation.
- Update 'AuthInterceptor' to catch 401 errors.
- Add unit tests for token expiration scenarios.
- Update API documentation with new endpoint details.
```

**Incorrect Format:**

```text
Added new auth features

I added a new way to handle tokens so that users don't get logged out.
It uses a new service and I also updated the interceptor.
**Changes:**

* `RefreshToken` service

* Tests
```

### 5. Submodule Sync Commits (Parent Repository)

- **Title**: `chore(submodules): sync <submodule-name> with <descriptive-action>`
  - First line format: `chore(submodules): sync <submodule-name> with <action>`
  - Include a descriptive action that summarizes the changes
- **Short Summary**: `Updates <submodule-name> from <old-sha-short> to <new-sha-short> (<head-commit-title>)`
- **Full Metadata Body (MANDATORY)**: The body MUST contain the complete,
  structured industrial record of the submodule advance.
- **Changes Section Header**: `Changes (<submodule-name>) [<old-sha>..<new-sha>]:`
  - **Mandatory**: Use the full 40-character SHAs for both the previous pointer
    and the new pointer.
  - **Format**: Bullet points (`-`) listing commits in chronological order (older to newer)
- **Chronological Order**: The commit list MUST be ordered from **older to newer**.
- **Metadata Section Header**: `Metadata (<submodule-name>):`
- **Mandatory Fields**:
  - `Submodule: <name> -> <new-sha>`
  - `Submodule commit parent: <sha>`
    (merge: <parent1> <parent2> ← only if merge commit)
  - `Submodule commit msg: <title-line>`
  - `<body-paragraphs-if-multiline>`
  - `Submodule commit changes`: List paths and line counts (for the tip commit).
  - `Submodule commit author`: <name> <email>
  - `Submodule commit author time`: <timestamp>
  - `Submodule commit committer`: <name> <email>
  - `Submodule commit committer time`: <timestamp>
- **Registration URL**: Every sync commit MUST include the registration URL from
  `.gitmodules` at the end of the metadata block:
  `Register <submodule-name> submodule pointing to <registration-url>`
- **Content Summary**:
  - Body must list key changes introduced by the submodule update, based on
    the submodule's commit messages.
  - Do not include only the commit SHA; always summarize the actual changes.
  - Do not repeat information from the title in the body.
- **No Paraphrasing**: The submodule's original commit message body must be
  preserved without summarization or omission.
- **Format Consistency**: Use bullet points for all details, wrap lines at 72
  characters, follow all other formatting rules, and maintain standard
  imperative mood for the title.

#### 5.1 Huge-Range Bookend Variant (>500 commits)

When the submodule advance spans **more than 500 commits**, a flat
chronological listing becomes impractical (Git practical limits, reviewer
fatigue, log noise). In that case the agent MUST switch to the **bookend
variant**:

- **Changes Section Header**: still `Changes (<submodule-name>) [<old-sha>..<new-sha>] (N commits, bookend listing):` where `N` is the exact `git rev-list --count <old>..<new>` figure.
- **First-K bullets**: list the **oldest 10 commits** in chronological order.
- **Ellipsis line**: a single bullet `- ... (<N - 20> intermediate commits omitted; full list: \`git -C <submodule> log --oneline <old>..<new>\`)`.
- **Last-K bullets**: list the **newest 10 commits** in chronological order, ending with the tip commit.
- **Tip metadata block**: identical to the standard variant (parent, message, changes, author, committer, registration URL) — NEVER omitted.
- **Threshold**: 500 commits. Below the threshold, use the standard flat list. The threshold MUST be stated explicitly in the body via the `(N commits, bookend listing)` annotation so reviewers know why the list is abridged.

### 6. Summarizing Opaque or Binary Changes

When changes involve files that Git identifies as binary (e.g., encoded text, large assets, database dumps), the agent
MUST "dig down" to understand the content.

- **Content Analysis**: Use tools like `cat -v`,`file`, or specific decoders to inspect the actual changes.
- **Explicit Asset Mentions**: If specific files are added or modified, name them in the body if they are critical to
    the commit's purpose (e.g., "Includes `db_13_12_25.dump` as the baseline snapshot").
- **Encoding Transparency**: If a file is converted or its encoding changes (e.g., UTF-16 to UTF-8), explicitly state
    this in the body.
- **Dependency Details**: For dependency files (e.g., `requirements.txt`,`local.txt`), even if seen as binary,
    summarize the key package updates instead of just saying "updates dependencies".

### 7. Amending Established Commits

When amending a commit (`git commit --amend`), the message MUST describe
the **final combined state** of the commit as it stands after incorporating
the amend — not the delta that the amend introduces, and not the original
pre-amend state.

**Procedure:**

1. Read the original commit's full diff and the new staged changes.
2. Identify the union of both: what does the commit now contain in total?
3. Write a single coherent message (title + body) for that union.

**Correct (final state):**

```text
docs(rules): cross-reference stash-apply failure fallback with resolvable URLs

- Link to selective file extraction recovery path via GitHub URL that
  resolves in standalone clone
- Link to redaction-portability skill for the URL decision framework
  used here
```

This describes the commit's final content: a blockquote that
cross-references the failure fallback, and the URLs are resolvable.

**Incorrect (delta-only — would be rejected):**

```text
docs(rules): replace prose reference with SHA-pinned URLs
```

This describes only what the amend changed, not what the commit IS.

**Submodule-specific consideration.** When amending a commit inside a
registered submodule, ensure the final message also passes the standalone-
clone test per `redaction-portability` skill §0.2. The message must make
sense to a reader who only has the submodule cloned, without knowledge of
the parent repo's internal labels or layout.

### 5.4 `i-have-adhd`

Source: [`.cursor/rules/i-have-adhd.mdc`](.cursor/rules/i-have-adhd.mdc) — blob `9dc1e3c2923e6c8caa4d9be803a861bb8bcefae2`.

<!-- I-HAVE-ADHD_START -->
# i-have-adhd

The reader has ADHD. Output is not just brief. It is shaped so an ADHD brain can act on it.

## Persistence

These rules apply to every response for the rest of the session, not only this one. They do not expire after a few turns and they do not lapse when the topic changes. If you are unsure whether they still apply, they do.

Turn them off only when the reader says "stop adhd mode" or "normal mode". Confirm in one line, then return to your default style.

## What ADHD changes about reading

Five facts drive every rule below:

1. Working memory is small. Anything not on screen is forgotten. Do not ask the reader to "keep in mind X."
2. Knowing the answer is not doing the answer. The friction between "got it" and "done it" is where work dies.
3. Starting is the hardest step. The first action must be obvious, small, and doable now.
4. Time estimates feel uniform. "A bit of work" and "a few hours" register the same. Vague estimates fail.
5. Dopamine is scarce. Visible progress matters. Buried wins do not register.

## Rules

### 1. Lead with the next action

The first line is something the reader can do. Not context. Not a plan. The action.

Bad: "Let's think about this. Your auth flow has a few moving pieces..."
Good: "Run `npm install jsonwebtoken`, then edit `src/auth.ts:42`."

If the answer is a command, path, or snippet, it goes first. Prose comes after, if at all.

### 2. Number multi-step tasks

If the work takes more than one step, write a numbered list. Each step is one bounded action. No step contains "and then" twice.

Use the fewest steps that still work. Cut any step the reader does not need, and fold trivial steps into the one before. A short path finished beats a complete path abandoned.

Bad: "First open the file, find the function, swap it out, then run the tests."

Good:
```
1. Open `src/auth.ts`
2. Replace `verifyToken` (lines 42 to 58) with the snippet below
3. Run `npm test -- auth.spec.ts`
```

### 3. End with one concrete next action

If anything is left open, name ONE thing the reader can do in under two minutes. Even "open the file" counts.

Bad: "Hope that helps. Let me know if you want to dig deeper."
Good: "Next: run `npm test` and paste the first failing line."

### 4. Suppress tangents

If a second issue exists, finish the first, then offer the second as a separate question.

Bad: "Here's the fix. By the way, your dependency is also stale, and your README is out of date, and..."
Good: "Here's the fix. Separately: there is also a stale dependency. Want me to handle that next?"

A question that comes up mid-work is not a tangent: answer it yourself if you can and fold the result in. If it still needs the reader, surface it once, at the end.

### 5. Restate state every turn

The reader cannot hold "we are on step 3 of 5" between messages. Restate it.

Bad: "Done. Ready for the next part?"
Good: "Step 3 of 5 done: schema updated. Next: backfill the new column. Run the script?"

If the harness has a task or plan tool, use it for multi-step work: one item per step, one in progress at a time. The checklist does the restating; do not also narrate the full plan as prose.

### 6. Give specific time estimates

Vague estimates fail. Ballpark in concrete units.

Bad: "This will take some work."
Good: "About 15 minutes if tests already cover this. An afternoon if not."

### 7. Make completed work visible

Show what now works, in concrete terms. Do not bury wins in a recap.

Bad: "I've made some changes to the auth flow. Among other things..."
Good: "Login now works with magic links. Try: `npm run dev`, open `/login`."

### 8. Matter-of-fact tone for errors

Never use "Uh oh," "Oh no," or "There seems to be a problem." State cause and fix.

Bad: "Uh oh, the test is failing. There seems to be an issue..."
Good: "Test fails at `auth.spec.ts:42`: expected 200, got 401. Cause: missing auth header. Fix: add `Authorization: Bearer ${token}` to the request."

### 9. Cap lists at 5 items

If a list grows past five, split into "do now" vs "later," or "must" vs "nice to have." Five items ranked beats ten unranked.

### 10. No preamble, no recap, no closing pleasantries

Forbidden openers: "Great question," "Let me...", "I'll...", "Sure!", "Looking at your...", "To answer your question..."

Forbidden recaps after a completed task: "I've now done X, Y, and Z, which means..."

Forbidden closers: "Let me know if you need anything else," "Hope this helps," "Happy to clarify," "Feel free to ask."

Start with the answer. End when the answer is done.

## When to break the rules

Override the defaults when:

1. User asks to "explain" or "walk me through." Explain fully. Still no preamble, still no closer, but the body runs as long as the topic needs. Add headers so the reader can skim back.
2. Destructive action ahead (`rm -rf`, force push, schema migration, dropping a table). Confirm before acting. Safety wins over brevity.
3. Debug spiral. If the last three turns have been "still broken," stop iterating on code. Name the assumption that might be wrong. Ask one diagnostic question.
4. Real ambiguity in the request. One short clarifying question beats guessing and rewriting.
5. A rule fights the task. When a rule would delete the answer itself, the task wins; the shape stays. Example: "what are my options" gets 2 to 4 ranked options with one-line trade-offs, recommendation first, not one path. The options are the answer.
6. A rule fights the harness. Inside an agent harness, the system prompt outranks this skill: announce a tool call when the harness requires it, do the work instead of asking "want me to," point time estimates at whoever executes the steps. Same principle as 5: the constraint wins, the shape stays.

## Pre-send check

Before sending, delete:

1. The first sentence if it announces what you are about to do.
2. The last sentence if it asks "anything else?" or recaps what just happened.
3. Any "by the way" sidebar.
4. Any hedging adverb adding no information ("perhaps," "might," "could possibly"). Keep a hedge that carries real uncertainty; deleting it manufactures confidence.
5. Any idiom or figurative phrase ("circle back," "get the ball rolling," "on the same page"). Replace with the literal action.

Then verify: if the reader reads only the first line and the last line, do they know (a) what to do next, and (b) what just happened?

If yes, send.
<!-- I-HAVE-ADHD_END -->

## 6. Engineering workflow

1. **Inspect** — confirm HEAD/status, read the relevant docs, trace the path end to end, and identify the smallest owning package.
2. **Plan** — state the intended behavior, invariants, files, and verification. Keep one task in progress at a time.
3. **Implement** — preserve package boundaries, existing naming, deterministic output, and error contracts.
4. **Review** — inspect the diff, all callers affected by shared changes, concurrency shutdown, security boundaries, and user-visible docs.
5. **Verify** — run focused checks first, then every applicable repository gate before claiming completion.

Typical commands:

```sh
gofmt -w <changed-go-files>
go test ./path/to/affected/package/... -count=1
go vet ./path/to/affected/package/...

go test ./... -count=1
go vet ./...
make lint
make allowlist
make build
```

Also run `make race` for concurrency-sensitive changes, `make vuln` for dependency/security work, relevant conformance or certification targets, and the actual `m`/`mx` behavior in a temporary fixture. Use isolated HOME/XDG/cache directories for global-state tests. Validate Windows-specific process, path, shim, and terminal behavior on Windows.

Do not claim a command passed unless it was run and its exit status was observed.

## 7. Repository shape

```text
cmd/m, cmd/mx                 thin process entrypoints
internal/cli                  Cobra parsing, dispatch, help, completions
internal/app                  cross-domain orchestration
internal/config               layered config and provenance
internal/apperr               typed ERR_M_* errors and exit mapping
internal/diagnostics          semantic events, redaction, reporters
internal/presentation         output modes, controller, static/live rendering
internal/presentation/help    plain and Glamour Markdown rendering
internal/presentation/pager   safe optional topic pager
internal/presentation/prompt  rich and accessible prompt adapters
internal/prompt               stdlib-only prompt contract
internal/manifest             package.json read/normalize/edit
internal/project              project discovery and identity
internal/workspace            workspace graph, filters, catalogs
internal/registry             registry clients, auth, metadata cache
internal/semver               npm-compatible range handling
internal/resolver             immutable dependency resolution and traces
internal/lockfile             canonical graph and format adapters
internal/fetch, archive       download, integrity, safe extraction
internal/store, linker        content store and install layouts
internal/transaction          staging, journal, commit, rollback
internal/lifecycle, policy    dependency scripts, trust, sandbox policy
internal/runner, process      scripts, exec, dlx, signals, child processes
internal/runtime, transform   stock-Node augmentation and transform service
internal/node, pmmanager      Node and external package-manager management
internal/compat               Nub/npm/pnpm/Yarn/Bun compatibility
internal/trace                lightweight in-process spans
internal/archcheck            import-graph and package-map enforcement
internal/testkit              fixtures, clean homes, local registry helpers
tests, fixtures, benchmarks   conformance, integration, corpora, performance
```

The authoritative map is [`docs/architecture/package-map.md`](docs/architecture/package-map.md).

## 8. Rule synchronization

When any of the four `.cursor/rules` files above changes, update the
matching injected block in this file in the same change. Keep the
source path and blob SHA current. Do not replace repository-local rule
text with content fetched from another repository or package registry.
