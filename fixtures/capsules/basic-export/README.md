# basic-export capsule fixture

Small registry-backed project used to exercise `m capsule create` and restore
round-trips.

## Layout after export

A capsule archive contains:

- `capsule.json` — manifest with lock bytes, root `package.json`, graph digest,
  platform, and blob references
- `blobs/<algo>/<hex>` — one verified tarball blob per unique package integrity

## Usage

Copy `package.json` into a clean-home project, run `m install`, then
`m capsule create --output basic-export.capsule`.
