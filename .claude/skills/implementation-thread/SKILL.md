---
name: implementation-thread
description: Drive one approved Mew task from design verification through implementation, documentation, adversarial review, local verification, and a held green pull request. Use for a coherent feature, bug fix, refactor, or compatibility change that must be completed end to end.
---

# Implementation thread

One owning agent keeps continuity across the whole task. Use fresh-context reviewers for high-leverage phases, but do not split ownership into unrelated phase agents.

## Phases

1. Reconfirm scope and acceptance criteria.
2. Map relevant Go packages, interfaces, fixtures, serialized formats, and process boundaries.
3. Review the plan before editing.
4. Implement the smallest complete vertical slice.
5. Add unit, integration, conformance, and documentation updates required by the behavior.
6. Run impact analysis over call sites, adapters, schemas, lockfile round trips, filesystem mutations, and cross-platform paths.
7. Run fresh-context correctness and security review when blast radius warrants it.
8. Fix valid findings and repeat verification.
9. Open a PR with issue linkage and exact test evidence.
10. Hold at the merge gate unless merge authorization was explicit.

Pause and surface any newly discovered default, security posture, public API/config/env, compatibility promise, or irreversible migration decision.
