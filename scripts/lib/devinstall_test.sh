#!/usr/bin/env bash
# Logic tests for devinstall.sh helpers.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=devinstall.sh
source "$SCRIPT_DIR/devinstall.sh"

passed=0
failed=0

ok() { passed=$((passed + 1)); echo "ok: $1"; }
fail() { failed=$((failed + 1)); echo "FAIL: $1"; }

assert_eq() {
  local name="$1" expected="$2" actual="$3"
  if [[ "$expected" == "$actual" ]]; then ok "$name"; else fail "$name (expected '$expected', got '$actual')"; fi
}

assert_contains() {
  local name="$1" needle="$2" hay="$3"
  if [[ "$hay" == *"$needle"* ]]; then ok "$name"; else fail "$name"; fi
}

assert_not_contains() {
  local name="$1" needle="$2" hay="$3"
  if [[ "$hay" != *"$needle"* ]]; then ok "$name"; else fail "$name"; fi
}

# 1. OS normalization
assert_eq 'goos normalize linux' 'linux' "$(devinstall_normalize_goos linux)"
assert_eq 'goos normalize darwin' 'darwin' "$(devinstall_normalize_goos DARWIN)"
if ( devinstall_normalize_goos freebsd >/dev/null 2>&1 ); then fail 'unsupported goos'; else ok 'unsupported goos'; fi

# 2. Arch normalization
assert_eq 'goarch normalize amd64' 'amd64' "$(devinstall_normalize_goarch x86_64)"
assert_eq 'goarch normalize arm64' 'arm64' "$(devinstall_normalize_goarch aarch64)"
if ( devinstall_normalize_goarch riscv64 >/dev/null 2>&1 ); then fail 'unsupported goarch'; else ok 'unsupported goarch'; fi

# 3. Detect native vs cross
DETECT="$(devinstall_detect)"
IFS='|' read -r HOST_OS HOST_ARCH TARGET_OS TARGET_ARCH EMULATED CAN_INSTALL <<<"$DETECT"
assert_eq 'native target os' "$HOST_OS" "$TARGET_OS"
assert_eq 'native canInstall' '1' "$CAN_INSTALL"
CROSS="$(devinstall_detect linux amd64)"
IFS='|' read -r _ _ TOS TARCH _ CI <<<"$CROSS"
if [[ "$HOST_OS" == linux && "$HOST_ARCH" == amd64 ]]; then
  assert_eq 'cross same still installable' '1' "$CI"
else
  assert_eq 'cross canInstall false' '0' "$CI"
fi

# 4. PATH managed block insert/replace/remove
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
printf 'echo hello\n' >"$TMP"
devinstall_upsert_managed_block "$TMP" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END" 'export PATH="/a/bin:$PATH"'
CONTENT="$(cat "$TMP")"
assert_contains 'path block insert' '/a/bin' "$CONTENT"
devinstall_upsert_managed_block "$TMP" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END" 'export PATH="/b/bin:$PATH"'
CONTENT="$(cat "$TMP")"
assert_contains 'path block replace' '/b/bin' "$CONTENT"
assert_eq 'path block single marker' '1' "$(grep -c '>>> mewjs dev installer' "$TMP" || true)"
devinstall_remove_managed_block "$TMP" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END"
CONTENT="$(cat "$TMP")"
assert_not_contains 'path block removed' '/b/bin' "$CONTENT"
assert_contains 'unrelated preserved' 'echo hello' "$CONTENT"

# 5. Completion block
devinstall_upsert_managed_block "$TMP" "$DEVINSTALL_COMPLETION_START" "$DEVINSTALL_COMPLETION_END" 'source /tmp/m'
assert_contains 'completion block insert' 'source /tmp/m' "$(cat "$TMP")"
devinstall_remove_managed_block "$TMP" "$DEVINSTALL_COMPLETION_START" "$DEVINSTALL_COMPLETION_END"
assert_not_contains 'completion block remove' 'source /tmp/m' "$(cat "$TMP")"

# 6. Default paths
PATHS="$(devinstall_default_paths '/tmp/custom mew/bin')"
IFS='|' read -r IDIR CDIR <<<"$PATHS"
assert_eq 'custom install dir' '/tmp/custom mew/bin' "$IDIR"
assert_contains 'custom completion root' 'completions' "$CDIR"

# 7. Source-build metadata resolves Git commit from working tree
unset MEW_VERSION
devinstall_resolve_metadata ''
assert_eq 'source build dev version' 'dev' "$DEVINSTALL_VERSION"
# When run from a Git working tree, commit should be resolved.
if git rev-parse HEAD >/dev/null 2>&1; then
  assert_eq 'source build commit set from git' "$(git rev-parse HEAD)" "$DEVINSTALL_COMMIT"
  assert_eq 'source build short set' "$(git rev-parse --short HEAD 2>/dev/null || git rev-parse HEAD | head -c7)" "$DEVINSTALL_SHORT_COMMIT"
fi
devinstall_resolve_metadata '1.2.3-local'
assert_eq 'source build version override' '1.2.3-local' "$DEVINSTALL_VERSION"
# commit still resolved from git even with version override
if git rev-parse HEAD >/dev/null 2>&1; then
  assert_eq 'source build override commit still set' "$(git rev-parse HEAD)" "$DEVINSTALL_COMMIT"
fi

# 8. Managed-block idempotency
IDEM="$(mktemp)"
trap 'rm -f "$TMP" "$IDEM"' EXIT
devinstall_upsert_managed_block "$IDEM" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END" 'export PATH="/x:$PATH"'
devinstall_upsert_managed_block "$IDEM" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END" 'export PATH="/x:$PATH"'
assert_eq 'idempotent markers' '1' "$(grep -c '>>> mewjs dev installer' "$IDEM")"

# 9. Alias completion body shape (bash)
ALIAS_TMP="$(mktemp)"
devinstall_upsert_managed_block "$ALIAS_TMP" "$DEVINSTALL_COMPLETION_START" "$DEVINSTALL_COMPLETION_END" $'complete -o default -F _m mew\ncomplete -o default -F _mx mewx'
ALIAS_CONTENT="$(cat "$ALIAS_TMP")"
assert_contains 'alias bash mew' 'mew' "$ALIAS_CONTENT"
assert_contains 'alias bash mewx' 'mewx' "$ALIAS_CONTENT"
rm -f "$ALIAS_TMP"

# 10. Install unix atomic copies (isolated temp)
INSTALL_TMP="$(mktemp -d)"
BIN_TMP="$(mktemp -d)"
trap 'rm -rf "$TMP" "$IDEM" "$INSTALL_TMP" "$BIN_TMP"' EXIT
mkdir -p "$BIN_TMP"
printf '#!/bin/sh\necho v\n' >"$BIN_TMP/m"
printf '#!/bin/sh\necho v\n' >"$BIN_TMP/mx"
printf '#!/bin/sh\necho v\n' >"$BIN_TMP/mew"
printf '#!/bin/sh\necho v\n' >"$BIN_TMP/mewx"
chmod +x "$BIN_TMP/m" "$BIN_TMP/mx" "$BIN_TMP/mew" "$BIN_TMP/mewx"
LINE="$(devinstall_install_unix "$PWD" "$BIN_TMP" m mx "$INSTALL_TMP" 0)"
if [[ -x "$INSTALL_TMP/m" && -x "$INSTALL_TMP/mx" && -x "$INSTALL_TMP/mew" && -x "$INSTALL_TMP/mewx" ]]; then
  ok 'install unix copies all four binaries'
else
  fail 'install unix copies all four binaries'
fi
# Verify mew and mewx are regular files (copies), not symlinks.
if [[ -f "$INSTALL_TMP/mew" && ! -L "$INSTALL_TMP/mew" ]]; then
  ok 'mew is a regular file (copy)'
else
  fail 'mew is a regular file (copy)'
fi
if [[ -f "$INSTALL_TMP/mewx" && ! -L "$INSTALL_TMP/mewx" ]]; then
  ok 'mewx is a regular file (copy)'
else
  fail 'mewx is a regular file (copy)'
fi

# 11. Uninstall cleans owned files
IFS='|' read -r IDIR CDIR <<<"$LINE"
devinstall_uninstall_unix "$IDIR" "$CDIR" 0 0
if [[ ! -e "$IDIR/m" && ! -e "$IDIR/mew" ]]; then ok 'uninstall removes owned files'; else fail 'uninstall removes owned files'; fi

echo ''
echo "results: $passed passed, $failed failed"
if [[ "$failed" -gt 0 ]]; then exit 1; fi
echo 'ok: all devinstall logic tests passed'
