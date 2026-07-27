# Peer resolution fixtures (MVP 0020 / Phase 4)

Registry-backed scenarios for importer-local peer provider identity.

| Case | Test |
|---|---|
| dual-importer providers | `TestPeerDualImporterProviders` |
| optional absent | `TestPeerOptionalAbsent` |
| scoped peers | `TestPeerScopedProviders` |
| non-root auto-install | `TestPeerAutoInstallAtImporter` |
| react ecosystem | `react-ecosystem/` + golden |

Run: `go test ./internal/resolver/... -run Peer -count=1`
