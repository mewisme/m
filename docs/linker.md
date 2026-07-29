# Linker modes

Mew supports two `node_modules` layouts selected by `install.linker` or
`m install --linker=<mode>`.

## Hoisted (default)

`auto` and `hoisted` use the copy/smart-link layout from MVP **0016**/**0018**.
Dependencies are hoisted to the project `node_modules` root when versions do
not conflict.

### Placement identity

Each physical install is tracked by a `PlacementID`:

```text
parentPlacement | importer | edgeName | packageKey | hoistLevel | peerContextDigest
```

`edgeName` is the exposed dependency name from `graph.Edge.Name` (the
`package.json` key or alias label), which may differ from the resolved package
name. The linker may place the same `packageKey` at multiple `destDir` paths when
peer-provider environments diverge or version conflicts force nesting. Placement
lists are sorted deterministically by `PlacementID`.

### Bins

Root `node_modules/.bin` receives shims only for **root direct** dependencies.
Packages installed in nested `node_modules` trees get local `.bin` directories
when they declare `bin` entries.

### Validation

Install-time validation walks the link plan per `Placement`: staged directory
exists, `package.json` identity matches the resolved graph, declared dependency
edges are reachable, and bin targets exist.

## Isolated (experimental)

`isolated` uses a pnpm-style virtual store:

```text
node_modules/
  .pnpm/<storeID>/node_modules/<pkg>/   # package content
  <edgeName>/  -> .pnpm/<storeID>/node_modules/<pkg>/   # top-level alias
  .mew/modules.v1.json                  # layout metadata
```

Each package instance only sees its declared dependencies in its private
`node_modules` folder, which blocks **phantom dependencies** (requiring a
transitive package that is not declared).

### StoreID

Virtual-store directory names use a readable `name@version` prefix. When the
identity would exceed 120 characters or peer providers are present, a
collision-resistant digest is always appended:

```text
sanitize(prefix)@sha256(fullIdentity)[:16]
```

Peer-provider keys are included in the full identity hash. Collision during layout
planning returns `ERR_M_INTEGRITY`.

### Private and peer links

- Private dependency links use `edge.Name` for directory segments.
- Peer links target the peer-context-specific provider instance.

### Enable isolated mode

1. Set `MEW_EXPERIMENTAL_ISOLATED_LINKER=1`
2. Run `m install --linker=isolated` or set `"install.linker": "isolated"` in
   `m.jsonc`

Without the experimental gate, `isolated` returns `ERR_M_USAGE`.

### Lockfile

`m.lock` `settings.linker` records the linker mode. `m install --frozen-lockfile`
uses the lock linker setting when present.

## Fixtures

Hoisted placement graphs live under `fixtures/linker/hoisted/` (multi-version,
scoped conflicts, nested bins, cycles, peer-context instances, alias+target).
Isolated fixtures live under `fixtures/linker/isolated/`. Integration tests cover
isolated layout, phantom-dependency blocking via real Node `require()`, and
`.bin` shims.

## Trade-offs

| Mode | Pros | Cons |
|---|---|---|
| Hoisted | npm-like, fewer links | Phantom deps possible |
| Isolated | Strict dependency boundaries | More paths, experimental gate |

## Deferred

- `m explain` dependency explanations (use instead of deferred `m why`)
- `public-hoist-pattern` selective hoisting (later MVP)
- Isolated layout crash-during-publication integration test (deferred)
