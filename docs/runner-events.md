# Runner diagnostic events

MVP **0045** and **0046** freeze versioned runner events consumed by reporters
and certification.

## `environment-prepared` v1

Emitted after environment materialization and lease acquisition, immediately
before the child process starts.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `type` | string | yes | Always `environment-prepared` |
| `v` | number | yes | Schema version `1` |
| `source` | string | yes | `project`, `dlx`, `snapshot`, or `capsule` |
| `cacheState` | string | yes | See enum below |
| `identityDigest` | string | yes | Lowercase SHA-256 hex, 64 characters |
| `graphDigest` | string | yes | Lowercase SHA-256 hex, 64 characters |
| `networkUsed` | boolean | yes | `true` only when registry metadata or artifact network was used during preparation |
| `prepareDurationMs` | number | yes | Milliseconds for preparation phase |

### `cacheState` enum (non-overlapping)

| Value | Meaning |
|---|---|
| `project` | Existing project environment reused |
| `warm-hit` | Previously published shared environment reused |
| `cold-built` | Shared environment materialized this invocation |
| `ephemeral` | Unique non-shared materialization |

### Ordering guarantee

1. Materialize and verify environment
2. Acquire execution lease when required
3. Emit `environment-prepared`
4. Start child process

### Reporter failure

If the reporter returns an error while emitting `environment-prepared`:

- The child does not start
- Execution lease is released
- Request-owned cleanup runs
- Shared cache entries remain intact
- Reporter error is the primary failure

Certification: `runner-event-schema` suite in the runner matrix.

## Inspect JSON v1

Plan-only inspect output (no materialization, lease, or artifact fetch).

```json
{
  "v": 1,
  "source": "snapshot",
  "identityDigest": "...",
  "graphDigest": "...",
  "cacheState": "cold",
  "materialized": false,
  "wouldMaterialize": true,
  "networkPolicy": "forbidden",
  "verified": true,
  "command": {
    "requested": "eslint",
    "owner": "eslint",
    "available": true
  },
  "warnings": []
}
```

### `source` enum

`project`, `dlx`, `snapshot`, `capsule`

### Inspect `cacheState` enum

`project`, `warm`, `cold`, `ephemeral`

Rules:

- Omit `command` when no command was requested; never emit `command: null`
- Omit `owner` inside `command` when unknown
- `warnings` is always an array (may be empty)
- No absolute paths, credentials, auth headers, or child environment leakage
- Warnings sorted deterministically

Certification: `runner-inspect-schema` suite.
