# Feature inventory maintenance

The authoritative machine-readable inventory is [`features/inventory.json`](../features/inventory.json). The JSON schema is [`features/inventory.schema.json`](../features/inventory.schema.json).

## When to update

Update the inventory in the same change when you:

- Add or rename a public command, flag, or config key
- Ship, defer, or intentionally omit a Nub-parity feature
- Add a Mew extension
- Change which MVP owns a feature

## How to update

1. Edit rows in [`internal/features/inventory_baseline.go`](../internal/features/inventory_baseline.go).
2. Regenerate JSON:

```powershell
$env:UPDATE_INVENTORY = "1"
go test ./internal/features/... -run TestWriteInventoryJSON -count=1
```

3. Regenerate the human-readable table (optional):

```powershell
go run ./cmd/m features --format table
go run ./cmd/m features --format json --module runner --status planned
go run ./cmd/m version
```

Cobra command definitions live in `internal/cli/`.

4. Run validation:

```powershell
go test ./internal/cli/... ./internal/features/... ./cmd/m/... ./cmd/mx/... -count=1
```

## Required fields per row

| Field | Description |
|---|---|
| `id` | Stable dotted identifier (`module.feature`) |
| `name` | Human-readable feature name |
| `module` | Grouping for `--module` filter |
| `nub_status` | Nub implementation state |
| `mew_status` | Mew implementation state |
| `compatibility_class` | `parity`, `extension`, `divergence`, or `deferred` |
| `primary_mvp` | Exactly one owning MVP (`0001`–`0090`) |
| `tests` | Conformance test IDs (may be empty until tests exist) |

## Status values

`planned`, `in-progress`, `shipped`, `intentional-omit`, `deferred`

## CI gates

- Every INDEX MVP must own at least one inventory row
- Rows missing `primary_mvp` fail validation
- Mew extensions must use `compatibility_class: extension`
- `m features --format json` omits internal `tests` from user output
