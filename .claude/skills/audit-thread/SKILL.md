---
name: audit-thread
description: Audit Mew behavior against an explicit specification or reference implementation and produce an evidence-backed gap catalog. Use for lockfile, CLI, registry, workspace, runtime, Node-version, or cross-platform parity work where the deliverable is findings and priorities rather than code.
---

# Audit thread

Define the comparison target precisely: tool, version, platform, configuration, and command surface.

## Method

1. Inventory the claimed surface from documentation and executable help.
2. Build focused fixtures, one behavior per fixture.
3. Run Mew and the reference on identical inputs.
4. Capture exit codes, stdout, stderr, files, lockfile semantics, and filesystem layout.
5. Classify each difference as intended, missing, incorrect, unsupported, or unknown.
6. Verify important differences against source or a second experiment.
7. Rank gaps by correctness, data-loss risk, security, frequency, and implementation dependency.
8. Produce an implementation handoff without changing product behavior.

Never use `parity` for an untested area. Record exact versions and commands.
