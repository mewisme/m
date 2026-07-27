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
parentPlacement | importer | depName | packageKey | hoistLevel
```

The linker may place the same `packageKey` at multiple `destDir` paths when
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
  <pkg>/  -> .pnpm/<storeID>/node_modules/<pkg>/   # top-level alias
  .mew/modules.v1.json                  # layout metadata
```

Each package instance only sees its declared dependencies in its private
`node_modules` folder, which blocks **phantom dependencies** (requiring a
transitive package that is not declared).

### StoreID v2

Virtual-store directory names use a readable `name@version` prefix and, when
peer providers are present, a short SHA-256 digest of the **resolved provider
keys** (not declared ranges):

```text
lodash@4.17.21
@scope+pkg@1.0.0@<peerDigest>
```

Store IDs are capped at 120 characters and use Windows-safe characters only.

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
scoped conflicts, nested bins, cycles, peer-context instances). Integration
tests cover isolated layout, phantom-dependency blocking via real Node
`require()`, and `.bin` shims.

## Trade-offs

| Mode | Pros | Cons |
|---|---|---|
| Hoisted | npm-like, fewer links | Phantom deps possible |
| Isolated | Strict dependency boundaries | More paths, experimental |

## Deferred

- `m why` dependency explanations (**0028**)
- Peer auto-install (**0020**)
- `public-hoist-pattern` selective hoisting (later MVP)
