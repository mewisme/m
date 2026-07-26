---
name: impact-analysis
description: Trace the full blast radius of a proposed or implemented Mew change across Go call sites, interfaces, graph models, lockfile adapters, disk formats, process boundaries, tests, documentation, and platforms. Use during significant design or mandatory implementation review.
---

# Impact analysis

Start from every changed exported or cross-package symbol, serialized field, file name, environment variable, error code, command, and persistent path.

Trace:

- callers and implementers
- interface satisfaction and mocks
- read/write pairs
- encode/decode and migration paths
- resolver-to-lockfile-to-linker data flow
- transaction and rollback behavior
- cache keys and invalidation
- process and signal behavior
- platform-specific files
- documentation and examples
- tests that assert old behavior

Report concrete affected paths, silent-risk areas, missing tests, and whether the current patch covers each impact. Do not propose unrelated cleanup.
