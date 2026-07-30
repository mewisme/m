#!/usr/bin/env sh
set -eu
target="${1:-core-cert}"
exec python3 "$(dirname "$0")/run_core_cert.py" "$target"
