# React peer ecosystem fixture

Hermetic peer-resolution cases for MVP 0020:

- `package.json` — depends on `react` only (strict peer failure without `react-dom`)
- `package-with-peers.json` — `react` + `react-dom` (peer context on `react`)
- `registry/` — local packuments served via `testkit.LoadRegistry`
