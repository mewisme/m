---
name: "0086 Cross-Cutting — AI Agent Implementation Protocol"
overview: "Give coding agents a deterministic workflow for implementing MVPs without losing architectural intent, compatibility context, or verification evidence."
todos:
  - id: reading
    content: "Required reading order"
    status: pending
  - id: scope
    content: "Predecessor checks"
    status: pending
  - id: evidence
    content: "Template"
    status: pending
  - id: security
    content: "Escalation triggers"
    status: pending
  - id: threads
    content: "Thread schema"
    status: pending
  - id: docs
    content: "Publish agent protocol"
    status: pending
isProject: false
---

# 0086 - Cross-Cutting — AI Agent Implementation Protocol

## Source contract

Canonical scope lives in [`plans/0086-ai-agent-protocol.md`](../0086-ai-agent-protocol.md). Do not expand scope in this Cursor plan; update the repo plan and regenerate.

## Done when

- All required tests pass on supported operating systems.
- No unresolved correctness, integrity, or data-loss issue remains.
- Public behavior and intentional deviations are documented.
- The next dependent MVP can consume stable interfaces without reaching into internals.

## Out of scope

- Do not implement features assigned to later indexed MVPs.
- Do not silently diverge from a documented compatibility contract.
- Do not optimize by weakening integrity, determinism, or crash safety.

## Predecessors

0004, 0008, 0009

Block implementation until predecessors are merged and meet their own exit criteria.

## Implementation steps

1. Confirm predecessors satisfied: 0004, 0008, 0009
2. Read source contract `plans/0086-ai-agent-protocol.md` and `0003`/`0007`/`0008` as required.
3. Implement packages: AGENTS.md, docs/agents/, .claude/skills/, .agents/
4. Add fixtures and tests listed in the source contract.
5. Run focused `go test` for touched packages, then applicable integration fixtures.
6. Update feature inventory / docs when public behavior changes.
7. Provide AI-agent handoff evidence (6 items in source contract).

## Verification

1. Agents have deterministic workflow doc
2. Evidence template lists 6 handoff items
3. Human-owned decisions called out

Suggested commands (once code exists):

```powershell
go test ./AGENTS.md/... -count=1
```

Adjust package paths to those actually created. Always include a clean-home fixture test for install-family work.

## Handoff

Before submitting work provide:

1. Behavior summary and compatibility target
2. Files and public interfaces changed
3. Test/benchmark/static-analysis commands
4. Known gaps and platform limits
5. Determinism evidence for generated files
6. Rollback note for persistent-format changes
