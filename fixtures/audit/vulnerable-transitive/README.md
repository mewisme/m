# vulnerable-transitive

Integration fixture: root depends on `carrier-pkg`, which depends on `vuln-pkg@1.0.0`
(advisory CVE-2026-0001 in `testdata/advisory/fixture-osv.json`).

Registry: `fixtures/registry/audit/v1`.
