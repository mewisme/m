#!/usr/bin/env bash
# Fuzz smoke: run short fuzz on packages that declare Fuzz* tests.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
found=0
while IFS= read -r pkg; do
  dir="$(go list -f '{{.Dir}}' "$pkg")"
  if ! grep -Rql '^func Fuzz' "$dir"/*_test.go 2>/dev/null; then
    continue
  fi
  found=1
  echo "fuzz-smoke: $pkg"
  go test "$pkg" -fuzz=. -fuzztime=1s -count=1
done < <(go list ./...)
if [[ "$found" -eq 0 ]]; then
  echo "fuzz-smoke: no Fuzz* targets; ok"
fi
