# Mew Implementation Plan Index

The numeric prefix is the canonical implementation order. Dependencies inside each file take precedence when parallel work is considered.

Tracking:

- Master checklist: [`CHECKLIST.md`](CHECKLIST.md)
- Release train: [`../docs/release-train.md`](../docs/release-train.md) / [`../features/milestones.json`](../features/milestones.json)
- Regenerate: `.\plans\scripts\enrich-and-generate.ps1`
- Enriched template: [`_ENRICHED_TEMPLATE.md`](_ENRICHED_TEMPLATE.md)

Each MVP line links the contract plan.

## Foundation

- [`0000-README.md`](0000-README.md) - Archive overview and product contract.
- [`0001-program-charter.md`](0001-program-charter.md) - Define Mew as a standalone Go-based JavaScript toolchain and package manager, with `m` as the primary toolchain binary and `mx` as the executable runner, while explicitly documenting Mew-specific improvements.
- [`0002-feature-inventory.md`](0002-feature-inventory.md) - Maintain a complete, testable inventory of Nub capabilities and Mew extensions, organized by module and implementation milestone.
- [`0003-target-architecture.md`](0003-target-architecture.md) - Define the final Go architecture, module boundaries, dependency direction, and the small embedded JavaScript surface required by Node extension APIs.
- [`0004-repository-bootstrap.md`](0004-repository-bootstrap.md) - Create a reproducible Go repository with strict quality gates, cross-platform builds, fixture management, and agent-friendly contributor guidance.
- [`0005-error-observability.md`](0005-error-observability.md) - Establish stable error codes, structured diagnostics, progress events, tracing, and redaction before implementing networked or destructive operations.
- [`0006-configuration-identity.md`](0006-configuration-identity.md) - Implement layered configuration and package-manager identity detection without reading branded configuration from an unrelated incumbent manager.
- [`0007-data-model-interfaces.md`](0007-data-model-interfaces.md) - Freeze canonical manifest, dependency graph, resolution, importer, package, policy, plan, snapshot, and lockfile models shared across the core.
- [`0008-testing-strategy.md`](0008-testing-strategy.md) - Build the test infrastructure required to port behavior safely and verify package-manager compatibility without depending on public registries.
- [`0009-release-train-overview.md`](0009-release-train-overview.md) - Define the ordered delivery train from package-manager core through complete Nub parity and Mew extensions.
- CLI presentation UX program (standalone): [`ux/README.md`](ux/README.md)

## Package-Manager Core

- [`0010-cli-foundation.md`](0010-cli-foundation.md) - Ship a stable command shell for `m` and `mx`, global flags, help, version output, command dispatch, and reserved-name policy.
- [`0011-manifest-project-discovery.md`](0011-manifest-project-discovery.md) - Reliably discover projects and workspaces, read and update `package.json` without destructive reformatting, and expose normalized dependency declarations.
- [`0012-registry-cache.md`](0012-registry-cache.md) - Fetch npm-compatible package metadata safely, efficiently, and reproducibly with authentication, proxies, retries, and an offline-capable metadata cache.
- [`0013-semver-basic-resolver.md`](0013-semver-basic-resolver.md) - Resolve registry dependencies and transitive dependencies using npm-compatible semver and produce a deterministic canonical graph with decision traces.
- [`0014-fetch-integrity-extraction.md`](0014-fetch-integrity-extraction.md) - Download package tarballs concurrently, verify integrity before use, and extract archives without path traversal or partial-store corruption.
- [`0015-m-lock.md`](0015-m-lock.md) - Design and implement Mew’s deterministic native lockfile with complete graph, importer, policy, integrity, peer-context, and compatibility metadata.
- [`0016-basic-installer.md`](0016-basic-installer.md) - Deliver the first usable `m install`, `m add`, and `m remove` path using `m.lock`, a project-local staging area, and a conservative hoisted layout.
- [`0017-transaction-rollback.md`](0017-transaction-rollback.md) - Make dependency mutations atomic at the product level and introduce install journals, snapshots, history, recovery, and instant rollback.
- [`0018-global-store-smart-linker.md`](0018-global-store-smart-linker.md) - Introduce an immutable global content-addressable store and automatically choose safe hardlink, reflink, copy, symlink, or junction strategies per filesystem.
- [`0019-isolated-linker.md`](0019-isolated-linker.md) - Implement a pnpm/Nub-style isolated dependency layout that prevents phantom dependencies while retaining a compatibility hoisted mode.
- [`0020-advanced-resolver.md`](0020-advanced-resolver.md) - Complete resolver semantics for peer dependencies, optional dependencies, overrides, aliases, platforms, workspace protocols, and deterministic incremental updates.
- [`0021-lifecycle-sandbox.md`](0021-lifecycle-sandbox.md) - Run required package lifecycle scripts under explicit trust policy, capability restrictions, reproducible build caching, and complete audit logs.
- [`0022-workspaces-catalogs.md`](0022-workspaces-catalogs.md) - Support monorepo discovery, workspace dependency graphs, catalogs, filters, root checks, and atomic multi-importer installation.
- [`0023-nub-pnpm-lock-bridge.md`](0023-nub-pnpm-lock-bridge.md) - Read, preserve, write, validate, diff, and explicitly migrate Nub and supported pnpm lockfile generations through the canonical graph.
- [`0024-npm-locks.md`](0024-npm-locks.md) - Support modern package-lock and npm-shrinkwrap formats while preserving npm project identity and install semantics.
- [`0025-bun-yarn-locks.md`](0025-bun-yarn-locks.md) - Implement Bun lock compatibility and a staged Yarn strategy covering classic read support and certified Berry/PnP read/write behavior.
- [`0026-pm-command-surface.md`](0026-pm-command-surface.md) - Complete the package-manager command family with a coherent Mew grammar, documented pnpm-compatible areas, and safe transaction-backed mutations.
- [`0027-advanced-sources-publish.md`](0027-advanced-sources-publish.md) - Support non-registry package sources, package patches, deterministic packing, and authenticated publication with provenance hooks.
- [`0028-explain-plan-history.md`](0028-explain-plan-history.md) - Expose every resolver and installer decision, preview all mutations, compare dependency graphs semantically, and run or restore historical snapshots.
- [`0029-performance-offline-capsules.md`](0029-performance-offline-capsules.md) - Optimize cold and warm installs, make offline behavior first-class, and package reproducible dependency environments for CI and containers.
- [`0030-security-audit-sbom.md`](0030-security-audit-sbom.md) - Provide comprehensive dependency risk analysis, signed provenance verification, SBOM export, age policies, and enforceable organizational rules.
- [`0031-core-stabilization.md`](0031-core-stabilization.md) - Certify the package-manager core for daily use before beginning runner and runtime parity work.

## Script and Executable Runners

- [`0040-script-runner.md`](0040-script-runner.md) - Implement `m run` with npm-compatible environment construction, lifecycle hooks, argument forwarding, signal propagation, and deterministic output.
- [`0041-workspace-runner.md`](0041-workspace-runner.md) - Run scripts across selected workspace packages with topology, concurrency control, failure policy, and structured output.
- [`0042-direct-script-shortcuts.md`](0042-direct-script-shortcuts.md) - Allow exact package.json script names such as `m dev` and `m start` while preserving deterministic built-in command precedence.
- [`0043-local-exec.md`](0043-local-exec.md) - Implement network-free local binary execution through `m exec`, workspace-aware `.bin` discovery, and robust platform shims.
- [`0044-mx-dlx.md`](0044-mx-dlx.md) - Implement secure temporary package execution with local-first behavior, consent, version pinning, execution cache, and offline support.
- [`0045-unified-execution.md`](0045-unified-execution.md) - Unify `m exec`, `mx`, historical snapshots, and capsules behind one environment builder and executable resolver.
- [`0046-runner-stabilization.md`](0046-runner-stabilization.md) - Certify script and executable execution for interactive development, CI, workspaces, PnP, and cross-platform signal behavior.

## Runtime and Watch

- [`0050-node-launch-compat.md`](0050-node-launch-compat.md) - Launch the user-selected stock Node process from Go with predictable argument handling, preload injection, compatibility escape hatches, and embedded runtime assets.
- [`0051-go-transform-service.md`](0051-go-transform-service.md) - Execute TypeScript through stock Node using a Go-native transform pipeline and a small embedded Node loader bridge.
- [`0052-jsx-decorators-sourcemaps.md`](0052-jsx-decorators-sourcemaps.md) - Complete transform behavior for JSX/TSX, automatic runtimes, legacy and standard decorators, metadata emission, and production-quality diagnostics.
- [`0053-module-resolution-loaders.md`](0053-module-resolution-loaders.md) - Match Node resolution while adding TypeScript path aliases, extension mapping, custom loader chaining, and package-manager layout awareness.
- [`0054-env-modern-apis.md`](0054-env-modern-apis.md) - Provide Nub-style environment-file loading and selected browser-compatible APIs without violating plain Node semantics or worker boundaries.
- [`0055-watch-mode.md`](0055-watch-mode.md) - Restart applications safely when relevant source, configuration, environment, or package dependencies change while preserving terminal and signal behavior.
- [`0056-debugging-inspection.md`](0056-debugging-inspection.md) - Integrate Node inspector, source maps, transform traces, module traces, and support bundles for production-quality debugging.
- [`0057-runtime-stabilization.md`](0057-runtime-stabilization.md) - Certify Mew runtime augmentation across supported Node versions, syntax features, module systems, workers, loaders, watch mode, and debugging.

## Node and Package-Manager Management

- [`0060-node-manager.md`](0060-node-manager.md) - Install, verify, select, cache, and automatically provision Node versions for projects and commands.
- [`0061-pm-manager.md`](0061-pm-manager.md) - Detect, download, cache, pin, invoke, and migrate external package managers as a Corepack replacement and compatibility tool.
- [`0062-shims.md`](0062-shims.md) - Install safe cross-platform shims that resolve pinned Node, Mew, and external package-manager versions without unexpectedly augmenting plain Node calls.

## Product and Distribution

- [`0070-project-init.md`](0070-project-init.md) - Create a fast, opinionated but transparent TypeScript-first project scaffold and a minimal manifest-only mode.
- [`0071-plugins.md`](0071-plugins.md) - Support discoverable external `m-<verb>` commands without loading untrusted code into the Mew process.
- [`0072-installers-releases.md`](0072-installers-releases.md) - Produce signed, reproducible multi-platform releases and safe install/update paths for direct download and common package channels.
- [`0073-github-action.md`](0073-github-action.md) - Provide a maintained GitHub Action that installs Mew, restores verified caches, selects Node, and exposes reproducible CI modes.
- [`0074-docker-builders.md`](0074-docker-builders.md) - Provide minimal container images, cache-efficient Docker patterns, and adapters for hosted build systems.

## Cross-Cutting Certification and Future Work

- [`0080-conformance-program.md`](0080-conformance-program.md) - Continuously certify Mew against Nub and incumbent package managers by behavior, file semantics, and runtime outcomes.
- [`0081-performance-program.md`](0081-performance-program.md) - Measure and control startup, resolution, network, extraction, linking, script, transform, memory, disk, and process overhead throughout development.
- [`0082-threat-model.md`](0082-threat-model.md) - Define adversaries, assets, trust boundaries, abuse cases, and mandatory reviews for a tool that downloads and executes third-party code.
- [`0083-rust-go-migration-map.md`](0083-rust-go-migration-map.md) - Map each Nub component and behavior to a Mew Go package, a compatibility test, a replacement design, or an intentional omission.
- [`0084-release-versioning-policy.md`](0084-release-versioning-policy.md) - Define compatibility promises for CLI, APIs, lockfiles, caches, plans, capsules, plugins, Node versions, and package-manager versions.
- [`0085-dependency-roadmap.md`](0085-dependency-roadmap.md) - Choose, evaluate, pin, and periodically review external Go dependencies for CLI, semver, transformation, filesystems, networking, archives, security, and releases.
- [`0086-ai-agent-protocol.md`](0086-ai-agent-protocol.md) - Give coding agents a deterministic workflow for implementing MVPs without losing architectural intent, compatibility context, or verification evidence.
- [`0087-definition-of-done.md`](0087-definition-of-done.md) - Define the non-negotiable completion standard applied to every MVP and the final program.
- [`0088-reference-index.md`](0088-reference-index.md) - Maintain the authoritative source list for Nub behavior, incumbent package-manager formats, Node APIs, registries, security standards, and Go implementation decisions.
- [`0089-research-spikes.md`](0089-research-spikes.md) - Resolve architecture questions that could invalidate later implementation before those MVPs freeze public contracts.
- [`0090-future-backlog.md`](0090-future-backlog.md) - Capture valuable post-parity ideas without allowing them to expand the ordered implementation critical path.

## Supporting Sources

- [`sources/nub-reference-snapshot.md`](sources/nub-reference-snapshot.md)
- [`sources/compatibility-targets.md`](sources/compatibility-targets.md)
