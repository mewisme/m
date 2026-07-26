# Compatibility Targets

Treat each materially different package-manager major version as a separate certification target. At minimum, maintain explicit entries for active npm versions, pnpm 9/10/11 as applicable, Yarn Classic, supported Yarn Berry generations, supported Bun releases, Nub lock generations, and the native `m.lock` generation.

For every target, record:

- Detection signals and precedence.
- Configuration files and field ownership.
- Lockfile reader status.
- Lockfile writer status.
- Frozen or immutable validation command.
- Filesystem layout expectations.
- Known semantic losses or intentional churn.
- Minimum fixture corpus and pinned tool version.
