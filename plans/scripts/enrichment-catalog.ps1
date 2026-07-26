# Enrichment catalog for all numbered Mew MVP plans.
# Dot-source from enrich-plans.ps1 / generate-*.ps1

$script:EnrichmentCatalog = @{}

function Add-Enrichment {
    param(
        [Parameter(Mandatory)][string]$Id,
        [Parameter(Mandatory)][string]$Phase,
        [Parameter(Mandatory)][string[]]$Packages,
        [string[]]$ForbiddenImports = @(),
        [string[]]$FeatureRows = @(),
        [string]$DataFlow = "",
        [string]$CommandsFlags = "N/A — no new public command surface in this MVP.",
        [string]$Artifacts = "N/A — no new persistent on-disk product artifacts.",
        [Parameter(Mandatory)][hashtable]$ChecklistGroups,
        [string[]]$Fixtures = @(),
        [string[]]$Acceptance = @(),
        [string[]]$Conformance = @(),
        [string[]]$OpenDecisions = @("None — proceed within documented scope."),
        [string[]]$TodoSummaries = @()
    )
    $script:EnrichmentCatalog[$Id] = [ordered]@{
        Id               = $Id
        Phase            = $Phase
        Packages         = $Packages
        ForbiddenImports = $ForbiddenImports
        FeatureRows      = $FeatureRows
        DataFlow         = $DataFlow
        CommandsFlags    = $CommandsFlags
        Artifacts        = $Artifacts
        ChecklistGroups  = $ChecklistGroups
        Fixtures         = $Fixtures
        Acceptance       = $Acceptance
        Conformance      = $Conformance
        OpenDecisions    = $OpenDecisions
        TodoSummaries    = $TodoSummaries
    }
}

# ---------------------------------------------------------------------------
# Foundation 0001–0009
# ---------------------------------------------------------------------------

Add-Enrichment -Id '0001' -Phase 'Foundation' `
    -Packages @('docs/charter.md', 'docs/compatibility-axes.md', 'AGENTS.md') `
    -FeatureRows @(
        '| Product identity `m`/`mx` | Nub `nub`/`nubx` | Mew naming | 0001 |',
        '| Direct script shortcuts policy | Absent in Nub | Intentional extension | 0001, 0042 |',
        '| Lockfile preservation rule | Nub policy | Mandatory Mew policy | 0001, 0023-0025 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  charter[ProductCharter] --> axes[CompatibilityAxes]
  axes --> naming[StableNaming]
  naming --> adr[DecisionRecords]
  adr --> later[LaterMVPs]
```
'@ `
    -CommandsFlags @'
| Surface | Notes |
|---|---|
| `m --version` / `mx --version` | Identity strings only until 0010 |
| Docs-only commands | No implementation required in 0001 |

Environment: none yet. Exit codes: document reserved ranges only.
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| `docs/charter.md` | Product contract |
| `docs/compatibility-axes.md` | Parity / divergence / extension / deferred matrix |
| ADR stubs under `docs/adr/` | Irreversible format decisions |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Write product charter covering Mew, Mewx, m.lock, and Nub parity goal',
            'Define compatibility axes: CLI grammar, lockfile, config, runtime, layout',
            'Document supported OS/arch and Node floor',
            'Freeze binary, config, cache, env, and error-code naming conventions',
            'Document experimental-feature and versioning policy'
        )
        'Core logic' = @(
            'Create compatibility-state vocabulary: parity, intentional divergence, extension, deferred',
            'Document dispatch precedence reserved for 0042 script shortcuts',
            'Document existing-lockfile preservation and new-project m.lock default',
            'List signature Mew differentiators with owning MVP IDs'
        )
        'CLI / UX' = @(
            'Draft user-facing identity strings for --version placeholders',
            'Draft migration narrative outline for npm/pnpm/Yarn/Bun/Nub users'
        )
        'Tests & fixtures' = @(
            'Review charter against representative npm, pnpm, Bun, Yarn, and Nub projects',
            'Verify every later INDEX module maps to an explicit product objective',
            'Add charter consistency checklist used by later MVP reviews'
        )
        'Docs & observability' = @(
            'Publish charter in docs/ and link from README/AGENTS.md',
            'Create ADR template for irreversible decisions',
            'Record open human-owned decisions with owners'
        )
    } `
    -Fixtures @(
        'fixtures/charter/npm-app — existing package-lock project for preservation wording',
        'fixtures/charter/pnpm-app — pnpm-lock.yaml project',
        'fixtures/charter/nub-app — nub.lock project',
        'fixtures/charter/empty — greenfield m.lock default wording'
    ) `
    -Acceptance @(
        'Charter names m, mx, m.lock, and Nub as behavioral reference without source-port language',
        'Compatibility axes table covers CLI, lockfile, config, runtime, and layout',
        'Every INDEX MVP maps to at least one charter objective',
        'Direct script shortcuts listed as intentional Mew extension',
        'ADR process documented before any persistent format is designed'
    ) `
    -Conformance @(
        'Nub product positioning — parity intent | parity',
        'Nub MIT/repo conventions — process only | defer',
        'Direct m <script> — Mew extension | extension'
    ) `
    -OpenDecisions @(
        'Exact Node LTS floor for v1 (link 0084/0089)',
        'Whether github.com/mewisme/m/mewx alias binaries ship in v1 installers'
    ) `
    -TodoSummaries @(
        'contracts: Freeze charter and compatibility axes',
        'naming: Freeze binary/config/env/error naming',
        'policy: Lockfile preservation and experimental gates',
        'docs: Publish charter + ADR template',
        'verify: Map INDEX MVPs to charter objectives'
    )

Add-Enrichment -Id '0002' -Phase 'Foundation' `
    -Packages @('internal/features', 'cmd/m (features subcommand stub)', 'testdata/features') `
    -ForbiddenImports @('internal/resolver', 'internal/linker', 'internal/runtime') `
    -FeatureRows @(
        '| m features table/json | Planned | Machine-readable inventory | 0002 |',
        '| Full capability matrix | Nub surface + Mew extensions | Single inventory schema | 0002 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  nubDocs[NubDocs] --> extract[InventoryExtract]
  charter[Charter] --> extract
  extract --> schema[features.json]
  schema --> cli[m features]
  schema --> ci[CIGates]
```
'@ `
    -CommandsFlags @'
| Command | Flags | Notes |
|---|---|---|
| `m features` | `--format table\|json`, `--module`, `--status` | May be stubbed until CLI 0010 |
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| `features/inventory.schema.json` | Versioned schema |
| `features/inventory.json` | Authoritative inventory |
| Generated docs tables | Human-readable views |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Define versioned feature-inventory JSON schema',
            'Define statuses: planned, in-progress, shipped, intentional-omit, deferred',
            'Define compatibility_class: parity, extension, divergence, deferred',
            'Require fields: id, module, nub_status, mew_status, primary_mvp, tests[]'
        )
        'Core logic' = @(
            'Extract all public Nub commands, flags, config keys, and documented behaviors',
            'Add Mew-only features from charter',
            'Assign every feature to exactly one primary MVP',
            'Link conformance test IDs where known'
        )
        'CLI / UX' = @(
            'Specify m features table and JSON output shapes',
            'Hide internal source paths from user-facing output'
        )
        'Tests & fixtures' = @(
            'Schema validation tests',
            'Inventory-to-command-tree consistency test (after 0010)',
            'Inventory-to-documentation consistency test',
            'CI fails when shipped commands are absent from inventory'
        )
        'Docs & observability' = @(
            'Generate human-readable tables from inventory',
            'Document how agents update inventory on behavior changes'
        )
    } `
    -Fixtures @(
        'testdata/features/minimal-inventory.json',
        'testdata/features/invalid-missing-mvp.json',
        'testdata/features/golden-table.txt'
    ) `
    -Acceptance @(
        'Schema rejects inventory rows missing primary_mvp',
        'Every INDEX MVP owns at least one inventory row',
        'Mew extensions marked compatibility_class=extension',
        'm features --format json validates against schema'
    ) `
    -Conformance @(
        'Nub CLI verb inventory coverage | parity',
        'Mew-only features flagged | extension'
    ) `
    -OpenDecisions @(
        'Whether inventory ships inside the binary via go:embed or remains docs-only until 0010'
    ) `
    -TodoSummaries @(
        'schema: Freeze inventory JSON schema',
        'extract: Populate Nub + Mew feature rows',
        'assign: Map features to primary MVPs',
        'cli: Specify m features output',
        'ci: Add inventory consistency gates'
    )

Add-Enrichment -Id '0003' -Phase 'Foundation' `
    -Packages @('cmd/m', 'cmd/mx', 'internal/app', 'internal/cli', 'internal/manifest', 'internal/project', 'internal/workspace', 'internal/registry', 'internal/resolver', 'internal/lockfile', 'internal/fetch', 'internal/archive', 'internal/store', 'internal/linker', 'internal/transaction', 'internal/lifecycle', 'internal/policy', 'internal/runner', 'internal/process', 'internal/runtime', 'internal/transform', 'internal/node', 'internal/pmmanager', 'internal/compat', 'internal/testkit', 'assets/runtime') `
    -ForbiddenImports @('cmd/* must not import internal implementation details beyond app/cli', 'internal/cli must not import linker/resolver directly') `
    -FeatureRows @(
        '| Go control plane | Nub Rust CLI | Go packages | 0003 |',
        '| Embedded JS loaders | Nub runtime/*.mjs | go:embed assets | 0003, 0050 |',
        '| Transform service | nub-native OXC | Go transform + IPC | 0003, 0051 |'
    ) `
    -DataFlow @'
```mermaid
flowchart TB
  cmd[cmd/m mx] --> app[internal/app]
  app --> cli[internal/cli]
  app --> pm[manifest project workspace registry resolver lockfile]
  pm --> mutate[fetch archive store linker transaction]
  mutate --> policy[lifecycle policy]
  app --> run[runner process runtime transform node]
```
'@ `
    -CommandsFlags @'
N/A — architecture document. Package import rules enforced by tests after 0004.
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| Architecture diagram in docs | Dependency direction |
| `tools/archcheck` or go test import guards | Prevent cycles |
| IPC protocol sketch for transform service | Versioned later in 0051 |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Produce full package map with one-line purpose per directory',
            'Define core interfaces: Registry, Resolver, Store, Linker, LockfileAdapter, Transaction, ScriptRunner, ProcessSupervisor',
            'Decide immutability boundaries and copy-on-write points',
            'Specify transform IPC framing, auth, cancellation sketch',
            'Define extension points without public plugin ABI'
        )
        'Core logic' = @(
            'Document stock-Node augmentation boundary (no libnode fork)',
            'Document resolve-complete-before-mutate rule',
            'Map every Nub crate to Mew package or intentional omission',
            'List forbidden import edges'
        )
        'CLI / UX' = @(
            'Keep cmd/m and cmd/mx as thin entrypoints in the diagram',
            'Document presentation vs domain separation'
        )
        'Tests & fixtures' = @(
            'Compile-time or test-time import graph checks',
            'Interface fakes proving independent testability',
            'IPC round-trip sketch tests when protocol exists'
        )
        'Docs & observability' = @(
            'Expand proposed repository tree to full listing',
            'Link architecture from AGENTS.md',
            'Record decisions that block later MVPs'
        )
    } `
    -Fixtures @(
        'docs/architecture/package-map.md',
        'docs/architecture/forbidden-imports.md'
    ) `
    -Acceptance @(
        'Every AGENTS.md package appears in the map',
        'No cyclic dependency in the documented graph',
        'JS surface limited to Node extension APIs',
        'Transaction boundary documented for all install-family mutations'
    ) `
    -Conformance @(
        'Nub stock-Node augmentation | parity',
        'OXC native addon replacement strategy | divergence'
    ) `
    -OpenDecisions @(
        'Transform IPC vs in-process only for v1 (see 0089)',
        'Whether internal/pm umbrella package exists or flat packages only'
    ) `
    -TodoSummaries @(
        'map: Finalize package map and forbidden edges',
        'ifaces: Freeze core interfaces',
        'boundary: Document Node augmentation + JS embed rules',
        'archcheck: Specify import-graph tests',
        'docs: Publish architecture package listing'
    )

Add-Enrichment -Id '0004' -Phase 'Foundation' `
    -Packages @('go.mod', 'cmd/m', 'cmd/mx', 'internal/testkit', 'Makefile', '.github/workflows', 'AGENTS.md', 'tools/') `
    -FeatureRows @(
        '| Repo bootstrap | Nub workspace CI | Go module + gates | 0004 |',
        '| Agent guidance | Nub AGENTS.md | AGENTS.md + skills | 0004 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  clone[CleanClone] --> tools[PinnedTools]
  tools --> test[go test vet lint]
  test --> ci[GitHubActionsMatrix]
  ci --> releaseReady[ReleaseTrainInput]
```
'@ `
    -CommandsFlags @'
| Developer command | Purpose |
|---|---|
| `make test` / `go test ./...` | Unit + integration |
| `make lint` | golangci-lint |
| `make vet` | go vet |
| `m development doctor` | Later; stub contract only |
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| `go.mod` / `go.sum` | Module identity |
| CI workflow YAMLs | Matrix builds |
| Fixture home helpers | Isolated tests |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Choose Go minimum version and document it',
            'Initialize module path and license headers',
            'Define directory skeleton matching 0003'
        )
        'Core logic' = @(
            'Add Makefile/task targets: test, vet, lint, race, fuzz-smoke, vuln',
            'Pin golangci-lint and govulncheck versions',
            'Create internal/testkit with temp home and fixture registry helpers',
            'Add license and dependency allowlist checks'
        )
        'CLI / UX' = @(
            'Stub cmd/m and cmd/mx main packages compiling to --help placeholder',
            'Document developer doctor command contract'
        )
        'Tests & fixtures' = @(
            'Clean-clone bootstrap test',
            'CI self-test that fails each quality gate intentionally in a job',
            'Cross-platform compile matrix including Windows'
        )
        'Docs & observability' = @(
            'Write AGENTS.md with ownership and reading order',
            'Add CONTRIBUTING with exact commands',
            'Document fixture checksum policy'
        )
    } `
    -Fixtures @(
        'fixtures/bootstrap/empty-module',
        'internal/testkit examples'
    ) `
    -Acceptance @(
        'Fresh clone: go test ./... passes on Linux/macOS/Windows CI',
        'Lint and vet wired in CI',
        'AGENTS.md present and linked from README',
        'cmd/m and cmd/mx build'
    ) `
    -Conformance @(
        'Nub CI discipline — process parity | parity'
    ) `
    -OpenDecisions @(
        'Makefile vs just vs task — pick one runner',
        'Module path github org/name finalization'
    ) `
    -TodoSummaries @(
        'module: Init go.mod and package skeleton',
        'gates: Wire test vet lint race vuln',
        'ci: Add OS/arch matrix',
        'testkit: Temp home + fixture helpers',
        'docs: AGENTS.md + CONTRIBUTING'
    )

Add-Enrichment -Id '0005' -Phase 'Foundation' `
    -Packages @('internal/diagnostics', 'internal/apperr', 'internal/trace') `
    -ForbiddenImports @('Must not import registry/fetch/linker') `
    -FeatureRows @(
        '| Stable error codes | Nub diagnostics | Mew codes | 0005 |',
        '| Progress events / NDJSON | Nub reporters | Event schema | 0005, 0040 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  op[Operation] --> err[TypedError+Code]
  op --> prog[ProgressEvents]
  err --> redact[Redaction]
  prog --> out[HumanOrNDJSON]
  redact --> out
```
'@ `
    -CommandsFlags @'
| Flag / env | Purpose |
|---|---|
| `--json` / `MEW_LOG_FORMAT` | Structured diagnostics |
| `--debug` / `MEW_DEBUG` | Verbose traces |
| `--color` | Presentation |
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| Error code registry doc | Stable codes |
| Event schema JSON | Progress/reporter contracts |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Define typed error with stable code, operation, subject',
            'Publish initial error code registry',
            'Define progress event schema (phase, package, bytes, …)',
            'Define redaction rules for URLs, tokens, headers'
        )
        'Core logic' = @(
            'Implement error wrapping helpers',
            'Implement human and NDJSON reporters',
            'Implement cancellation mapping to exit codes',
            'Add trace span hooks without mandatory OTel dependency'
        )
        'CLI / UX' = @(
            'Map codes to exit statuses',
            'Ensure secrets never print in default or debug modes without explicit unsafe flag'
        )
        'Tests & fixtures' = @(
            'Table tests for code→exit mapping',
            'Redaction golden tests',
            'Progress event golden tests'
        )
        'Docs & observability' = @(
            'Document codes for users and agents',
            'Document reporter formats'
        )
    } `
    -Fixtures @(
        'testdata/diagnostics/redact-cases.json',
        'testdata/diagnostics/progress-golden.ndjson'
    ) `
    -Acceptance @(
        'Every public failure path yields a stable code',
        'Tokens in registry URLs are redacted in logs',
        'JSON reporter validates against schema'
    ) `
    -Conformance @(
        'Nub reporter concepts | parity',
        'Mew error code registry | extension'
    ) `
    -OpenDecisions @(
        'Adopt OpenTelemetry optionally later vs custom spans only'
    ) `
    -TodoSummaries @(
        'codes: Freeze error code registry',
        'types: Implement typed errors + wrapping',
        'reporter: Human + NDJSON progress',
        'redact: Credential/URL redaction',
        'tests: Golden diagnostics fixtures'
    )

Add-Enrichment -Id '0006' -Phase 'Foundation' `
    -Packages @('internal/config', 'internal/project/identity') `
    -FeatureRows @(
        '| Layered config | Nub/npmrc-like | Mew config layers | 0006 |',
        '| PM identity detection | Nub | packageManager + lockfile order | 0006 |'
    ) `
    -DataFlow @'
```mermaid
flowchart TB
  env[Env] --> merge[ConfigMerge]
  user[UserConfig] --> merge
  project[ProjectConfig] --> merge
  merge --> identity[IdentityDetect]
  pkg[package.json packageManager] --> identity
  lock[LockfilePresence] --> identity
```
'@ `
    -CommandsFlags @'
| Command | Notes |
|---|---|
| `m config get/set/list` | May land with 0026; define keys now |
| Global flags | `--config`, `--cwd`, `--offline` |
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| User config file | `~/.config/github.com/mewisme/m/config.toml` (final path per naming doc) |
| Project config | Neutral names where convention exists |
| Effective config dump | Debug only |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Define config layer precedence',
            'Define identity detection order matching AGENTS.md',
            'List owned config keys vs pass-through npmrc keys',
            'Forbid reading unrelated branded PM config as authority'
        )
        'Core logic' = @(
            'Implement layered loader with deterministic merge',
            'Implement identity detector',
            'Implement offline/prefer-offline flags in config model',
            'Validate unknown keys policy (warn vs fail)'
        )
        'CLI / UX' = @(
            'Specify config command grammar',
            'Effective-config debug output with redaction'
        )
        'Tests & fixtures' = @(
            'Precedence table tests',
            'Identity detection fixtures for each lockfile type',
            'Malformed config fail-closed tests'
        )
        'Docs & observability' = @(
            'Document every public config key',
            'Document identity detection with examples'
        )
    } `
    -Fixtures @(
        'fixtures/identity/npm-lock',
        'fixtures/identity/pnpm-lock',
        'fixtures/identity/nub-lock',
        'fixtures/identity/packageManager-field',
        'fixtures/identity/conflict-signals'
    ) `
    -Acceptance @(
        'Detection order matches AGENTS.md',
        'Conflicting signals produce explicit errors, not silent picks',
        'Env overrides project overrides user as documented'
    ) `
    -Conformance @(
        'packageManager field precedence | parity',
        'No foreign branded config authority | extension'
    ) `
    -OpenDecisions @(
        'TOML vs JSON vs yaml for Mew-native config file'
    ) `
    -TodoSummaries @(
        'layers: Freeze config precedence',
        'identity: Implement detection order',
        'keys: Publish owned config key list',
        'tests: Identity + merge fixtures',
        'docs: Config reference'
    )

Add-Enrichment -Id '0007' -Phase 'Foundation' `
    -Packages @('internal/graph', 'internal/lockfile/canonical', 'internal/manifest/types', 'internal/policy/types', 'internal/plan') `
    -FeatureRows @(
        '| Canonical graph model | Aube models | Go types | 0007 |',
        '| Plan / snapshot models | Mew extension | Shared types | 0007, 0017, 0028 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  manifest[Manifest] --> graph[CanonicalGraph]
  graph --> lock[LockAdapters]
  graph --> plan[MutationPlan]
  plan --> tx[Transaction]
```
'@ `
    -CommandsFlags 'N/A — data model freeze only.' `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| Versioned Go types | Shared across core |
| Golden JSON/YAML encodings | Deterministic ordering tests |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Freeze Manifest, Dependency, Importer, Package, Graph, Edge types',
            'Freeze ResolutionDecision and PeerContext types',
            'Freeze Policy, Plan, Snapshot, Capsule descriptors',
            'Define deterministic sort keys for all collections'
        )
        'Core logic' = @(
            'Define immutability rules for graph values',
            'Define ID schemes for packages and importers',
            'Define integrity and tarball URL fields',
            'Define migration-friendly version fields'
        )
        'CLI / UX' = @(
            'Specify explain/plan JSON shapes consumed by 0028'
        )
        'Tests & fixtures' = @(
            'Round-trip golden encoding tests',
            'Ordering stability tests',
            'Invalid graph rejection tests'
        )
        'Docs & observability' = @(
            'Publish data-model doc with diagrams',
            'Link types to owning packages'
        )
    } `
    -Fixtures @(
        'testdata/graph/simple-app.json',
        'testdata/graph/peers.json',
        'testdata/graph/workspace.json'
    ) `
    -Acceptance @(
        'All later core MVPs can depend on these types without reaching into adapters',
        'Deterministic encoding byte-identical across platforms',
        'Version field present on every persistent model'
    ) `
    -Conformance @(
        'Aube canonical graph concepts | parity'
    ) `
    -OpenDecisions @(
        'Exact package ID string format (link 0015)'
    ) `
    -TodoSummaries @(
        'types: Freeze canonical Go models',
        'ids: Freeze package/importer ID schemes',
        'order: Deterministic collection ordering',
        'golden: Encoding fixtures',
        'docs: Data-model reference'
    )

Add-Enrichment -Id '0008' -Phase 'Foundation' `
    -Packages @('internal/testkit', 'tests/conformance', 'tests/integration', 'fixtures/registry', 'fixtures/projects') `
    -FeatureRows @(
        '| Fixture registry | Nub/Aube tests | Local registry | 0008 |',
        '| Conformance harness | Nub | Differential fixtures | 0008, 0080 |'
    ) `
    -DataFlow @'
```mermaid
flowchart LR
  fix[Fixtures] --> reg[LocalRegistry]
  reg --> mew[MewUnderTest]
  reg --> ref[ReferencePM]
  mew --> cmp[Compare]
  ref --> cmp
```
'@ `
    -CommandsFlags @'
| Harness command | Purpose |
|---|---|
| `go test ./tests/...` | Suites |
| `make conformance` | Differential runs |
| `make fuzz-smoke` | Short fuzz |
'@ `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| `fixtures/registry/v1/` | Packaged tarballs + metadata |
| `fixtures/projects/*` | Project corpora |
| Golden outputs | Lockfiles, trees, stdout |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Define fixture manifest format and checksums',
            'Define clean-home test contract',
            'Define differential comparison report schema'
        )
        'Core logic' = @(
            'Implement local fixture registry server helper',
            'Implement isolated HOME/XDG/cache redirection',
            'Implement reference PM invocation wrappers (optional when tools present)',
            'Define fuzz targets list for parsers'
        )
        'CLI / UX' = @(
            'Document how to add a fixture',
            'Document required metadata: OS, tool versions'
        )
        'Tests & fixtures' = @(
            'Smoke: install from fixture registry',
            'Failure injection helpers: network cut, disk full simulation',
            'Cross-platform path/symlink/junction probes'
        )
        'Docs & observability' = @(
            'Testing strategy doc with layout diagram',
            'Conformance inventory stub for 0080'
        )
    } `
    -Fixtures @(
        'fixtures/registry/v1/lodash-4.17.21.tgz',
        'fixtures/projects/basic-cjs',
        'fixtures/projects/basic-esm',
        'fixtures/projects/typescript-app',
        'fixtures/projects/workspace-simple'
    ) `
    -Acceptance @(
        'Tests never require public registry access',
        'Clean-home tests do not touch developer global state',
        'Fixture checksums verified on load'
    ) `
    -Conformance @(
        'Local-registry testing discipline | parity'
    ) `
    -OpenDecisions @(
        'Which reference PM versions to pin in CI images'
    ) `
    -TodoSummaries @(
        'layout: Freeze fixtures/ and tests/ layout',
        'registry: Local fixture registry helper',
        'home: Clean-home isolation helpers',
        'fuzz: List parser fuzz targets',
        'docs: Testing strategy guide'
    )

Add-Enrichment -Id '0009' -Phase 'Foundation' `
    -Packages @('docs/release-train.md', 'plans/INDEX.md') `
    -FeatureRows @(
        '| Release train | Nub modules | Ordered MVPs | 0009 |',
        '| Experimental gates | Nub | Feature flags | 0009 |'
    ) `
    -DataFlow @'
```mermaid
flowchart TB
  F[Foundation0001-0009] --> C1[Core0010-0016]
  C1 --> C2[Core0017-0022]
  C2 --> C3[Core0023-0031]
  C3 --> R[Runners0040-0046]
  R --> RT[Runtime0050-0057]
  RT --> M[Managers0060-0062]
  M --> P[Product0070-0074]
  P --> X[Cross0080-0090]
```
'@ `
    -CommandsFlags 'N/A — sequencing policy.' `
    -Artifacts @'
| Artifact | Purpose |
|---|---|
| Milestone dependency graph | No cycles |
| Channel criteria | alpha/beta/rc/stable |
'@ `
    -ChecklistGroups @{
        'Contracts & types' = @(
            'Create milestone dependency graph with no cycles',
            'Define alpha/beta/rc/stable criteria',
            'Define which MVPs may ship experimentally',
            'Define support windows for lock adapters and Node',
            'Define stop-the-line criteria'
        )
        'Core logic' = @(
            'Map every inventory feature to a milestone',
            'Define backport and format-migration policy',
            'Require readers before writers for public formats'
        )
        'CLI / UX' = @(
            'Document feature-flag naming for experimental commands'
        )
        'Tests & fixtures' = @(
            'Validate graph has no cycles',
            'Dry-run release checklist on empty scaffold'
        )
        'Docs & observability' = @(
            'Publish release-train doc',
            'Keep INDEX.md synchronized'
        )
    } `
    -Fixtures @(
        'docs/release-train.md',
        'testdata/release/empty-scaffold-checklist.md'
    ) `
    -Acceptance @(
        'Every inventory feature has a milestone',
        'Stabilization gates 0031/0046/0057 cannot start early',
        'Stop-the-line criteria include corruption and integrity failures'
    ) `
    -Conformance @(
        'Ordered delivery vs Nub module layout | divergence'
    ) `
    -OpenDecisions @(
        'Public versioning scheme 0.x vs 1.0 timing (0084)'
    ) `
    -TodoSummaries @(
        'graph: Publish MVP dependency graph',
        'channels: Define alpha/beta/rc/stable',
        'gates: Experimental + stop-the-line rules',
        'map: Inventory features to milestones',
        'docs: Release-train overview'
    )
