# Transform Parity Reference

Mew is a transpiler, not a type checker. This document records intentional
differences from the TypeScript compiler (`tsc`) and the current support
level for each transform feature.

## JSX

| Feature | Support | Notes |
|---|---|---|
| Classic runtime (`jsx: "react"`) | Full | `React.createElement` calls |
| Automatic runtime (`jsx: "react-jsx"`) | Full | `react/jsx-runtime` imports |
| Development mode (`jsx: "react-jsxdev"`) | Full | Mapped to automatic runtime |
| Preserve (`jsx: "preserve"`) | Full | JSX left intact for downstream tool |
| Custom factory (`jsxFactory`) | Full | e.g. `h` for Preact classic |
| Custom fragment (`jsxFragmentFactory`) | Full | e.g. `Fragment` |
| Custom import source (`jsxImportSource`) | Full | e.g. `preact` for Preact automatic |
| `.tsx` / `.jsx` file extensions | `.tsx` supported; `.jsx` deferred (0053) | |

### JSX default importSource

When `jsx: "react-jsx"` is set without `jsxImportSource`, Mew defaults to
`react` (same as esbuild and tsc). Explicit `jsxImportSource` overrides.

## Decorators

| Feature | Support | Notes |
|---|---|---|
| Legacy TypeScript decorators | Full | Via esbuild; enabled by tsconfig `experimentalDecorators` |
| `emitDecoratorMetadata` | Parity | Carried in NormalizedOptions and cache keys; esbuild does not emit metadata — Mew carries the flag for future metadata emission |
| TC39 standard decorators | Deferred | Pending esbuild upstream support |

### Decorator metadata strategy

Mew carries `emitDecoratorMetadata` in NormalizedOptions and includes it in
cache keys. Actual metadata emission is not yet implemented. The decision
between Go-native metadata reflection and embedded JS helpers is deferred to
a later MVP. Nub parity for metadata is separately certified.

## Source Maps

| Feature | Support | Notes |
|---|---|---|
| No source map (`sourceMap: false`) | Default | |
| Inline source maps (`inlineSourceMap: true`) | Full | `sourceMappingURL` data URL in output |
| External source maps (`sourceMap: true`) | Full | Separate `.map` returned in TransformResult |
| Source content inclusion (`inlineSources`) | Default include | esbuild default includes sources; zero-value bool ambiguity noted |
| Source root (`sourceRoot`) | Full | Passed through to esbuild |
| Map root (`mapRoot`) | Carried | In NormalizedOptions for cache keys; esbuild does not use mapRoot directly |
| Stack trace mapping | Via Node `--enable-source-maps` | |

## Module Format

| tsconfig `module` | Mew behavior |
|---|---|
| `CommonJS` | `FormatCJS` |
| `ES6`, `ES2015`, `ES2020`, `ES2022`, `ESNext`, `NodeNext`, `Node16` | `FormatESM` |
| `Preserve` | Keeps the loader-inferred format |
| Unset | Inferred from file extension (`.mts`/`.mjs` → ESM, `.cts`/`.cjs` → CJS) |

## Target

| tsconfig `target` | esbuild target |
|---|---|
| `ES3` | `ES5` (esbuild minimum) |
| `ES5` | `ES5` |
| `ES2015` / `ES6` | `ES2015` |
| `ES2016` – `ES2024` | Matched exactly |
| `ESNext` | `ESNext` |
| Unset | Node major version heuristic (≥22 → ESNext, ≥20 → ES2023, ≥18 → ES2022, else ES2020) |

## Differences from TypeScript compiler (tsc)

1. **No type checking.** Mew strips types and emits JavaScript. Use `tsc --noEmit`
   for type checking in CI.

2. **No const enum inlining.** esbuild does not inline const enums from other
   modules. Use regular enums or inline constants.

3. **No declaration files.** `.d.ts` generation requires `tsc`.

4. **No path alias resolution.** `baseUrl` and `paths` are carried for cache
   key stability but not yet resolved (0053).

5. **Decorators always transpiled.** esbuild does not gate decorator syntax on
   `experimentalDecorators` (the flag only affects the type checker in tsc).
   Mew always transforms decorators when present.

6. **No `emitDecoratorMetadata` emission.** The flag is recognized and included
   in cache keys but no metadata is emitted.

7. **Source map content always included by default.** esbuild includes source
   content in source maps; explicit `inlineSources: false` is not distinguished
   from absent (zero-value bool). If this matters, a tristate option type is
   needed.

## Cache key coverage

Every field on NormalizedOptions flows into the transform cache key via
canonical JSON serialization. Cache keys change when:
- JSX mode, factory, fragment factory, or import source changes
- Decorator flags change
- Source map options change
- Target or module format changes
- Path aliases change

## Diagnostic code frames

Transform errors and warnings include file path, line, column, length, and
source snippet (line text) from esbuild. These point to the original source,
not the emitted JavaScript.

## Unsupported tsconfig options

The `UnsupportedOptions` function in `internal/transform/tsconfig.go`
identifies compilerOptions keys that Mew does not recognize. Currently
supported keys:

`target`, `module`, `moduleResolution`, `useDefineForClassFields`,
`verbatimModuleSyntax`, `importHelpers`, `baseUrl`, `paths`, `jsx`,
`jsxFactory`, `jsxFragmentFactory`, `jsxImportSource`, `sourceMap`,
`inlineSourceMap`, `inlineSources`, `sourceRoot`, `mapRoot`,
`experimentalDecorators`, `emitDecoratorMetadata`

All other compilerOptions keys are classified as unsupported. The unsupported
list is available programmatically; CLI diagnostic emission is deferred to
0053 (resolution-aware diagnostics).
