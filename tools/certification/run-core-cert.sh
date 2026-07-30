#!/usr/bin/env sh
set -eu
target="${1:-core-cert}"
exec pwsh -NoProfile -File "$(dirname "$0")/run-core-cert.ps1" -Target "$target"
