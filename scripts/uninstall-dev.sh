#!/usr/bin/env bash
# MewJS development uninstaller (Unix). LF line endings required.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# shellcheck source=lib/devinstall.sh
source "$SCRIPT_DIR/lib/devinstall.sh"

stage() { printf '[%s] %s\n' "$1" "$2"; }

INSTALL_DIR=''
KEEP_PATH=0
KEEP_COMPLETION=0

usage() {
  cat <<'EOF'
Usage: uninstall-dev.sh [options]

  --install-dir <dir>   Install directory (default: XDG mewjs/bin)
  --keep-path           Do not remove PATH profile block
  --keep-completion     Do not remove completion files or profile block
  -h, --help            Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --keep-path) KEEP_PATH=1; shift ;;
    --keep-completion) KEEP_COMPLETION=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) devinstall_die "unknown argument: $1" ;;
  esac
done

cd "$REPO_ROOT"

PATHS="$(devinstall_default_paths "$INSTALL_DIR")"
IFS='|' read -r INSTALL_DIR COMPLETION_ROOT <<<"$PATHS"

stage uninstall "removing installer-owned files from ${INSTALL_DIR}"
devinstall_uninstall_unix "$INSTALL_DIR" "$COMPLETION_ROOT" "$KEEP_PATH" "$KEEP_COMPLETION"

stage done 'development uninstall complete'
