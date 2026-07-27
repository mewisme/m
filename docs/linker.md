# Linker modes

Mew supports two `node_modules` layouts selected by `install.linker` or
`m install --linker=<mode>`.

## Hoisted (default)

`auto` and `hoisted` use the copy/smart-link layout from MVP **0016**/**0018**.
Dependencies are hoisted to the project `node_modules` root when versions do
not conflict.

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

### Enable isolated mode

1. Set `MEW_EXPERIMENTAL_ISOLATED_LINKER=1`
2. Run `m install --linker=isolated` or set `"install.linker": "isolated"` in
   `m.jsonc`

Without the experimental gate, `isolated` returns `ERR_M_USAGE`.

### Lockfile

`m.lock` `settings.linker` records the linker mode. `m install --frozen-lockfile`
uses the lock linker setting when present.

## Trade-offs

| Mode | Pros | Cons |
|---|---|---|
| Hoisted | npm-like, fewer links | Phantom deps possible |
| Isolated | Strict dependency boundaries | More paths, experimental |

## Deferred

- `m why` dependency explanations (**0028**)
- Peer auto-install (**0020**)
- `public-hoist-pattern` selective hoisting (later MVP)
