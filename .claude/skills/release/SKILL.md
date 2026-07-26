---
name: release
description: Prepare, verify, publish, and communicate a Mew release across Go binaries, checksums, signatures, SBOMs, installers, package wrappers, containers, and release notes. Use only for an authorized release or release rehearsal.
---

# Release

1. Confirm version, release scope, target commit, and authorization.
2. Verify the full local and CI gate on the release commit.
3. Build the supported OS/architecture matrix from a clean environment.
4. Verify `m version`, `mx version`, binary metadata, and embedded runtime assets.
5. Generate checksums, signatures or attestations, and SBOMs according to policy.
6. Test install and upgrade paths in clean Linux, macOS, and Windows environments.
7. Publish GitHub artifacts, package wrappers, containers, and package-manager channels atomically where possible.
8. Write factual release notes from merged changes and known limitations.
9. Comment the shipped version on included issues and PRs.
10. Verify download URLs and installation after publication.

Never publish from an uncommitted tree or reuse artifacts from a different commit.
