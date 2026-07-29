# peer-conflict

Fixture for `m explain` on a missing strict peer dependency.

- Root depends on `react@^18.0.0`, which peers `react-dom@^18.0.0`.
- With strict peer dependencies, resolution fails until `react-dom` is installed.
- `m explain react-dom` returns a structured peer conflict explanation.
