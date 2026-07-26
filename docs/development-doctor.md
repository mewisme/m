# Development doctor

Contract for `m development doctor`.

## Stub (MVP 0004)

The current command:

- Prints `status=ok` for the running Go toolchain (`runtime.Version()`).
- Prints `status=stub` for golangci-lint and govulncheck with versions from
  [`tools/versions.env`](../tools/versions.env) when readable.
- Always exits **0** so CI can invoke it without failing the job.
- Does not install tools, mutate the environment, or fail on missing optional tools.

```powershell
go run ./cmd/m development doctor
```

## Future non-stub checks (not yet implemented)

When the stub is replaced, doctor should report actionable failures for:

1. Go version below the `go.mod` floor
2. Missing or wrong-version golangci-lint / govulncheck when `--strict` is set
3. Unreadable or mismatched `tools/versions.env`
4. Broken module download (`go list ./...`)
5. Common contributor misconfigurations (later: Node floor, fixture paths)

Exit codes (future):

| Code | Meaning |
|---|---|
| 0 | Healthy (or stub) |
| 1 | One or more required checks failed |
| 2 | Doctor itself could not run |

See also [`docs/engineering.md`](engineering.md) and [`CONTRIBUTING.md`](../CONTRIBUTING.md).
