# cache-hit-miss

Expectations for registry metadata cache tests (MVP 0012):

1. Cold fetch stores `packument.json` + `meta.json` with etag and sha256.
2. Warm fetch sends `If-None-Match` and accepts HTTP 304 without rewriting body checksum.
3. `--offline` serves from cache; miss returns `ERR_M_NETWORK`.
