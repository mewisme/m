# npm alias fixture

Hermetic alias resolution for stabilization pass 2:

- `package.json` — depends on `foo: npm:bar@^1.0.0`
- `registry/` — local packument for `bar`

Expected graph edge: `name=foo`, `to=bar@1.0.0`, `range=npm:bar@^1.0.0`.
