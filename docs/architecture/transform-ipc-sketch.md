# Transform IPC protocol sketch

Status: sketch only. Full implementation is MVP 0051. Open decision (MVP 0089):
whether v1 uses local IPC, in-process transform only, or both.

## Goals

- Serve synchronous Node loader hooks that need TypeScript/JSX transform.
- Keep Go as the transform owner (divergence from Nub OXC N-API addon).
- Support auth between Node child and local Go service, and cancellation.

## Framing (sketch)

Length-prefixed JSON frames over a local stream (Unix domain socket or named
pipe on Windows):

```text
[u32 little-endian payload length][utf-8 JSON body]
```

### Request

```json
{
  "v": 1,
  "id": "opaque-request-id",
  "op": "transform",
  "path": "relative/or/absolute/source.ts",
  "source": "...",
  "cancel_token": "optional"
}
```

### Response

```json
{
  "v": 1,
  "id": "opaque-request-id",
  "ok": true,
  "code": "...",
  "map": null,
  "error": null
}
```

## Auth

- Parent Go process creates a one-time token and passes it via environment to the
  Node child and to the transform listener.
- Listener rejects frames without a matching token (header field or first hello).

## Cancellation

- Client may send `op: "cancel"` with the same `id`, or close the stream.
- Server aborts in-flight work when `context.Context` is canceled.

## Go sketch

See [`internal/transform/protocol.go`](../../internal/transform/protocol.go) for
version constants and encode/decode helpers exercised by unit tests.
