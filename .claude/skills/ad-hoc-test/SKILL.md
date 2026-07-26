---
name: ad-hoc-test
description: Create a temporary local probe for a Mew behavior that is difficult to cover immediately with the normal suite. Use for hypothesis testing, platform-neutral reproduction, reference-tool comparison, or one-off investigation before promoting a durable regression test.
---

# Ad-hoc local test

Create a self-contained fixture under a temporary directory. Do not depend on the developer's global package-manager configuration or cache unless that is the subject of the test.

Record:

- exact Mew binary
- OS and architecture
- Go and Node versions
- reference-tool versions
- fixture contents
- commands and exit codes
- output and generated files

Delete the temporary fixture after collecting evidence. Promote a stable behavior into `tests/` when it prevented or exposed a real defect.
