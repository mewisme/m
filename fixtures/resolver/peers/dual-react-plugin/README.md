# Dual React peer-context fixture

Hermetic multi-instance peer resolution:

- `package.json` — depends on `host-a` and `host-b`
- Each host pins a different `react` major and pulls `plugin@1.0.0`
- Expect two distinct `plugin@1.0.0#react@…` package nodes in the resolved graph
