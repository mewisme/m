# offline full-cache-project

Frozen `m.lock` with pkg-a → pkg-b → pkg-c integrity pins. Integration tests seed
registry metadata and tarball blobs into an isolated cache, then run `m install --offline`.
