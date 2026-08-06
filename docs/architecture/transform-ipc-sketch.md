# Transform IPC protocol

Status: production. Protocol version 2. Implemented in
[`internal/transform/`](../../internal/transform/).

## Goals

- Serve synchronous Node loader hooks that need TypeScript/JSX transform.
- Keep Go as the transform owner.
- Support auth between Node child and local Go service.
- Support in-flight cancellation of active transforms.

## Framing

Length-prefixed JSON frames over local TCP (127.0.0.1):

```text
[u32 little-endian payload length][utf-8 JSON body]
```

Maximum frame size: 48 MiB (MaxFrameSize).

### Op codes

| Op | Description |
|----|-------------|
| `hello` | Authenticate session (first frame on connection) |
| `health` | No-op health check |
| `transform` | Request a file transform |
| `cancel` | Cancel an in-flight transform by cancel token |
| `shutdown` | Close the connection gracefully |

### Hello (auth)

First frame on every connection. Go validates the token with constant-time
comparison.

Request:
```json
{"v": 2, "token": "<hex-encoded 32-byte random>"}
```

Response:
```json
{"v": 2, "ok": true}
```

Failure:
```json
{"v": 2, "ok": false, "err_code": "ERR_M_TRANSFORM_AUTH", "reason": "unauthorized"}
```

### Transform request

```json
{
  "v": 2,
  "id": "<opaque request id>",
  "op": "transform",
  "path": "relative/or/absolute/source.ts",
  "source": "<file contents>",
  "source_digest": "<sha256 hex>",
  "loader": "ts|tsx|mts|cts",
  "format": "esm|cjs",
  "options": "<JSON-encoded NormalizedOptions>",
  "opts_digest": "<sha256 hex>",
  "node_major": 22,
  "source_map": "none|inline|external",
  "cancel_token": "<opaque token for OpCancel>"
}
```

### Transform response

```json
{
  "v": 2,
  "id": "<matching request id>",
  "ok": true,
  "code": "<transformed JavaScript>",
  "map": "<source map string>",
  "digest": "<sha256 hex of output>",
  "cache": "hit|miss|bypass"
}
```

Error:
```json
{
  "v": 2,
  "id": "<matching request id>",
  "ok": false,
  "err_code": "ERR_M_TRANSFORM_SYNTAX",
  "error": "<sanitized message>"
}
```

### Cancel request

```json
{
  "v": 2,
  "id": "<this cancel frame's id>",
  "op": "cancel",
  "cancel_token": "<token from the target transform request>"
}
```

Cancel is acknowledged with a separate response frame:
```json
{"v": 2, "id": "<cancel request id>", "ok": true}
```

The cancelled transform receives its own terminal error response.

### Shutdown

```json
{"v": 2, "id": "<request id>", "op": "shutdown"}
```

Response:
```json
{"v": 2, "id": "<request id>", "ok": true}
```

After the response, the server closes the connection.

## Concurrency design

### Frame reading is decoupled from transform execution

The connection read loop in `handleConn` reads frames sequentially. When it
receives an `op: "transform"` frame it:

1. **Synchronously** (in the read loop): validates the request, checks for
   duplicate request IDs, creates a per-request context with timeout derived
   from the session context, and registers the cancel token in the session's
   active-cancels map.
2. **Dispatches** the remaining work to a goroutine, then continues reading
   the next frame immediately.

This means `op: "cancel"` frames arriving on the same connection are
processed while the target transform is still running — they don't wait
for engine completion or timeout.

### Per-connection write serialization

Multiple transform goroutines may attempt to write responses to the same
TCP connection concurrently. A per-connection `sync.Mutex` (`writeMu`)
serializes all response writes, preventing frame interleaving.

The `writeResponseLocked` helper acquires the mutex and writes a single
length-prefixed frame. Write failure (connection dead) is treated as
non-fatal — the goroutine skips the write and cleans up state. The
connection-level cleanup (deferred close, waitgroup) handles the rest.

### Worker semaphore is context-aware

Worker slot acquisition uses `select` on both the semaphore channel and
the request context:

```go
select {
case s.workers <- struct{}{}:
    defer func() { <-s.workers }()
case <-reqCtx.Done():
    // Cancelled while waiting — write cancellation response and return.
}
```

A request cancelled while queued for a worker slot exits promptly.

## Cancel-token lifecycle

1. **Registration**: The cancel token (from the `cancel_token` field) is
   registered in `Session.activeCancels` synchronously in the read loop,
   before the transform goroutine starts. The stored value is the
   `context.CancelFunc` for the per-request context.

2. **Cancellation path** (`OpCancel`): The read loop processes cancel
   frames inline. Under `activeCancelsMu`, the token is looked up. If
   found, `cancel()` is called (cancelling the request context) and the
   token is removed from the map. An acknowledgment frame is written.

3. **Completion path**: After the engine returns, the transform goroutine
   enters the terminal response gate. Under `activeCancelsMu`, it checks:
   - Is the token still in the map? (If not, cancel already won.)
   - Is `reqCtx.Err() != nil`? (Context deadline exceeded.)

   If the token is present and the context is not done, the goroutine
   **commits to success**: it removes the token from the map (so a late
   cancel is a no-op) and writes the success response.

   If the token is absent or the context is done, it writes a
   cancellation or timeout response.

4. **Deferred cleanup**: When the transform goroutine exits, deferred
   functions clean up: request ID from `activeIDs`, cancel token from
   `activeCancels` (no-op if already removed by cancel or commit),
   `reqCancel()` called (no-op if already cancelled).

### Terminal-response guarantees

Each transform request produces **at most one** terminal result:

- **Success**: engine returned a result, commit gate passed.
- **Typed transform error**: syntax error, unsupported syntax, config error, etc.
- **Timeout**: `requestTimeout` elapsed before engine completed.
- **Cancellation**: explicit `OpCancel` or session shutdown.

The `activeCancelsMu` lock is the commit point. Only one of "success" or
"cancelled" can win:

- Cancel wins the lock → token removed, `cancel()` called → context
  cancelled → engine wakes up → goroutine sees ctx.Err() → writes cancel
  response.
- Transform wins the lock → token removed → success response written →
  late cancel arrives, sees no token → idempotent no-op.

A cancelled request never produces a success response. The client observes
at most one terminal frame for each request ID.

### Idempotent cancel

- **Unknown token**: `OpCancel` for a never-registered token → acknowledged OK.
- **Already completed**: `OpCancel` after transform finished → token not
  found → acknowledged OK.
- **Duplicate cancel**: second `OpCancel` for same token → token already
  removed → acknowledged OK.
- **Cancellation is scoped**: cancelling one token does not affect other
  active transforms.

## Timeout behavior

Each transform request has a `requestTimeout` (default 60s) derived from
the session context via `context.WithTimeout`. When the deadline expires:

1. `reqCtx.Done()` fires → engine's `Transform` sees ctx cancellation →
   returns `ctx.Err()`.
2. The transform goroutine checks `reqCtx.Err()` in the terminal response
   gate → writes `ERR_M_TRANSFORM_TIMEOUT`.
3. The deferred cleanup removes the cancel token from `activeCancels`.

The per-request timeout and explicit `OpCancel` share one cancellation
path: both cancel `reqCtx`. The terminal response gate distinguishes them:
if the token is still present in `activeCancels` when `reqCtx.Err()` is
set, it's a timeout (no explicit cancel won the race). If the token is
absent, it was an explicit cancel.

## Shutdown and session lifecycle

`Session.Close()` is idempotent and concurrency-safe. Shutdown order:

1. **Cancel session context** — propagates to all derived request contexts.
2. **Close listener** — stops the accept loop.
3. **Cancel active transforms** — iterates `activeCancels` and calls each
   `cancel()`. Every active request context is cancelled.
4. **Close tracked connections** — unblocks reads/writes.
5. **Wait for goroutines** — `sync.WaitGroup` ensures server, connection,
   and request goroutines have exited.
6. **Clean up maps** — clears `activeCancels` and `activeIDs`.

After `Close()` returns, no transform goroutines, timers, cancel
functions, writer goroutines, or pending entries remain.

## Error code stability

Error codes exposed to clients are drawn from a fixed allowlist in
`stableErrorCodes`. Source content, endpoint addresses, tokens, and
options payloads are redacted from error messages by `SanitizeErrorMessage`.

Stable codes include: `ERR_M_TRANSFORM_SYNTAX`, `ERR_M_TRANSFORM_TIMEOUT`,
`ERR_M_TRANSFORM_CANCELLED`, `ERR_M_TRANSFORM_UNAVAILABLE`,
`ERR_M_TRANSFORM_ENGINE`, and others listed in `protocol.go`.

## Loader transport

The Node loader (`ts-loader.mjs`) connects over TCP, authenticates with
the session token, and sends length-prefixed JSON frames. Key behaviors:

- **Concurrent requests**: multiple `sendTransform` calls share one TCP
  connection. Responses are dispatched by request ID.
- **Pending map**: `Map<id, {resolve, reject, timer}>` tracks in-flight
  requests. Response handler matches on `resp.id`.
- **Timeout**: each request sets a timer. On expiry, the pending entry is
  removed and a best-effort `OpCancel` frame is sent.
- **Malformed responses**: frames that parse to a known ID but lack an
  `ok` field are treated as errors. Frames with no ID fail the oldest
  pending request if any exist.
- **Late responses**: responses for IDs no longer in the pending map are
  silently dropped.
- **Retry policy**: only transient transport errors (ECONNRESET, EPIPE,
  etc.) trigger one retry. Semantic errors, timeouts, and cancellations
  are never retried. Caller abort (`AbortSignal`) prevents retry.
- **Connection loss**: all pending requests are rejected with a transient
  error on disconnect.
