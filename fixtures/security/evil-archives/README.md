# Evil archives

Known-bad path members for archive parser fail-closed tests.
Never extract these into production paths. The `path-traversal-members.txt`
lists hostile member names; a real crafted `.tgz` may be added when
`internal/archive` ships extraction.
