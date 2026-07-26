# Engineering standards

## Go toolchain

| Item | Value |
|---|---|
| Module path | `mew` (see [`docs/adr/0001-module-path.md`](adr/0001-module-path.md)) |
| Minimum Go | **1.26.5** (must match `go` directive in [`go.mod`](../go.mod)) |
| Task runner | Makefile (see [`docs/adr/0002-tool-runner.md`](adr/0002-tool-runner.md)) |

CI installs Go 1.26.x. Local development must not rely on newer language features than the `go.mod` floor.

## Quality gates

| Gate | Local | CI |
|---|---|---|
| Unit/integration tests | `make test` / `go test ./... -count=1` | all OS jobs |
| `go vet` | `make vet` | all OS jobs |
| Race detector | `make race` | Ubuntu only |
| Lint | `make lint` (pinned golangci-lint) | Ubuntu |
| Vulnerability scan | `make vuln` (pinned govulncheck) | Ubuntu |
| Fuzz smoke | `make fuzz-smoke` | optional / local |
| Dependency allowlist | `make allowlist` | Ubuntu |
| Cross-compile | — | linux/darwin/windows × amd64/arm64 build-only |

ARM64 is covered by the **cross-compile** matrix, not by hosted ARM runners for full tests (cost). Windows behavior for package-manager paths must still be validated on Windows CI jobs, not inferred from Unix-only green builds.

## Tool pins

Versions live in [`tools/versions.env`](../tools/versions.env). Do not float `latest` in CI. Bump pins in a dedicated PR with a short note in the PR body.

## Fixture checksum policy

- Fixtures under `fixtures/` are source-of-truth inputs for tests.
- Prefer small, hand-authored fixtures. When a fixture is generated, check in the exact bytes and document the generator command in the fixture `README.md`.
- Do not mutate fixtures from tests. Copy into a temp dir (`internal/testkit.CopyFixture`) before edits.
- Binary and registry blobs must appear in [`fixtures/registry/v1/manifest.json`](../fixtures/registry/v1/manifest.json) with SHA-256 digests. `testkit.LoadRegistry` verifies checksums on load.
- See [`docs/testing.md`](testing.md) for clean-home, local registry, fuzz, and conformance layout.

## Dependency allowlist

Direct and indirect modules must appear in [`tools/allowlist/modules.txt`](../tools/allowlist/modules.txt). Adding a dependency requires updating that file in the same PR. Prefer the standard library.

## License

Root [`LICENSE`](../LICENSE) is MIT. Per-file license headers are not required in this MVP; CI checks that `LICENSE` exists and mentions MIT.
