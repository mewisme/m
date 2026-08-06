# Runtime Known Limitations

Documented as of 0057 (runtime stabilization gate). Each entry includes the limitation, impact, and planned resolution path.

## Transformer

### OXC divergence

**Limitation**: Mew uses esbuild for TypeScript/JSX/TSX transforms, not OXC (the Nub reference). Esbuild covers the same syntax surface but has different diagnostic formatting, different JSX dev-mode output, and different decorator handling for legacy TypeScript decorators.

**Impact**: Identical TypeScript input may produce semantically equivalent but byte-different output compared to Nub. This is intentional divergence — Mew targets behavioral parity, not bit-for-bit output parity.

**Resolution**: No planned migration to OXC. Divergence documented as permanent.

### Decorator metadata emission

**Limitation**: `emitDecoratorMetadata` tsconfig flag is parsed and carried in `NormalizedOptions` and cache keys, but actual metadata emission strategy (Go-native vs embedded JS) is deferred.

**Impact**: TypeScript projects using `emitDecoratorMetadata` with reflection metadata (`reflect-metadata` package) will not receive type metadata at runtime.

**Resolution**: Planned for 0060+ (Node manager integration). Decision between Go-native metadata emission and embedded JS bridge pending.

### TC39 decorators (stage 3)

**Limitation**: Standard TC39 decorators are not supported. Only legacy TypeScript experimental decorators work (via esbuild).

**Impact**: Projects using `@decorator` syntax in `.js`/`.mjs` files (not TypeScript) will fail to transform.

**Resolution**: Deferred to esbuild upstream. Will be enabled when esbuild supports TC39 decorators natively.

## Loader Bridge

### TypeScript type checking

**Limitation**: Mew is a transpiler, not a type checker. No semantic diagnostics are performed — type errors are silently accepted.

**Impact**: Invalid TypeScript (type mismatches, missing properties) will execute without errors. Developers must run `tsc --noEmit` separately for type checking.

**Resolution**: By design. Type checking is out of scope for MewJS. Documented as permanent divergence.

### PnP unplugged mode

**Limitation**: PnP detection requires `.pnp.cjs` at the project root. `.pnp.data.json` without `.pnp.cjs` (Yarn PnP "unplugged" mode) is detected but PnP resolution is skipped.

**Impact**: Projects using Yarn PnP in unplugged mode will not get PnP-aware resolution. They fall through to tsconfig paths and stock Node resolution.

**Resolution**: Planned for 0060+ when PnP integration is revisited.

### Custom loader ordering

**Limitation**: Custom loaders specified via `--loader` are injected before Mew's ts-loader in the hook chain. This means custom loaders run first and can claim resolutions that ts-loader would otherwise handle. This is the correct ordering for hook chaining, but may surprise users expecting Mew's loader to run first.

**Impact**: A custom loader that claims all `.ts` resolutions will prevent Mew's ts-loader from running, breaking path alias resolution and extension mapping.

**Resolution**: Documented behavior. Add `m resolve-module` diagnostics (0053) for debugging loader chains.

## Runtime

### Worker threads

**Limitation**: Worker threads inherit the preload chain from the main thread, but worker-specific transform configuration (separate tsconfig) is not supported.

**Impact**: Workers use the same transform options as the main thread entrypoint. Multi-package monorepos where workers need different tsconfig settings are not supported.

**Resolution**: Planned for 0060+ with per-package transform configuration.

### Web Storage

**Limitation**: `localStorage` and `sessionStorage` globals are polyfilled in-memory only. Data does not persist across process restarts.

**Impact**: Packages that depend on persistent Web Storage (e.g., caching user preferences across runs) will lose data on restart.

**Resolution**: Disk persistence planned for 0060+. Currently gated behind `MEW_EXPERIMENTAL_RUNTIME=1`.

### Watch mode

**Limitation**: Watch mode uses fsnotify with a polling fallback on platforms/filesystems where inotify/FSEvents/ReadDirectoryChangesW are unavailable. The polling interval is fixed at 1 second.

**Impact**: On network filesystems (NFS, CIFS) or inside some Docker configurations, watch mode falls back to polling and may have up to 1 second of latency.

**Resolution**: Configurable polling interval planned for 0060+.

### Inspector passthrough

**Limitation**: `--inspect` and `--inspect-brk` flags are passed through to Node's V8 inspector verbatim. Mew does not integrate with the inspector protocol for source-map-aware debugging or breakpoint resolution in original TypeScript source.

**Impact**: Debugging TypeScript in Chrome DevTools or VS Code shows transformed JavaScript, not original TypeScript source. Source maps are emitted but not automatically consumed by the debugger.

**Resolution**: Source-map-aware debugging via inspector integration planned for 0060+.

## Transform Service

### Service lifecycle

**Limitation**: The transform service is started on-demand per `m` invocation and shut down when the parent process exits. There is no persistent daemon mode.

**Impact**: Cold starts pay the full service startup cost (~50-100ms). Repeated `m` invocations (e.g., in watch mode restarts) each pay the startup cost.

**Resolution**: Persistent daemon mode planned for 0060+ (Node manager integration).

### Concurrent transform limits

**Limitation**: The transform service processes requests sequentially over a single Unix socket connection. Concurrent transforms from multiple files (e.g., in a large project) are serialized.

**Impact**: Transform throughput is bounded by single-core esbuild performance. Multi-core parallelism is not exploited.

**Resolution**: Connection pool or multiple service instances planned for post-0060.

## Node Version Support

| Node version | Status |
|---|---|
| 18.x | Supported |
| 20.x | Supported |
| 22.x | Supported (primary) |
| 24.x | Supported |

Node 16.x and earlier are unsupported. The minimum supported Node version is 18.x.

## Gated Features

Features behind experimental flags as of 0057:

| Feature | Gate | Status |
|---|---|---|
| Runtime augmentation | `MEW_EXPERIMENTAL_RUNTIME=1` | Experimental |
| Direct dispatch bins | `MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH=1` | Experimental |
| Watch mode | `MEW_EXPERIMENTAL_RUNTIME=1` | Experimental |
| Web Storage | `MEW_EXPERIMENTAL_RUNTIME=1` | Experimental |
| Debug inspection | `--inspect`/`--inspect-brk` | Passthrough only |
