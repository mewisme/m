#!/usr/bin/env bash
# MewJS development installer shared library (POSIX). LF line endings required.
# shellcheck disable=SC2034

DEVINSTALL_PATH_START='# >>> mewjs dev installer >>>'
DEVINSTALL_PATH_END='# <<< mewjs dev installer <<<'
DEVINSTALL_COMPLETION_START='# >>> mewjs dev completion >>>'
DEVINSTALL_COMPLETION_END='# <<< mewjs dev completion <<<'

DEVINSTALL_REPO_ROOT=''
DEVINSTALL_HOST_GOOS=''
DEVINSTALL_HOST_GOARCH=''
DEVINSTALL_TARGET_GOOS=''
DEVINSTALL_TARGET_GOARCH=''
DEVINSTALL_VERSION=''
DEVINSTALL_COMMIT=''
DEVINSTALL_SHORT_COMMIT=''
DEVINSTALL_DIRTY=0
DEVINSTALL_BUILD_DATE=''
DEVINSTALL_INSTALL_DIR=''
DEVINSTALL_COMPLETION_BASE=''
DEVINSTALL_CAN_INSTALL=0
DEVINSTALL_GO_MIN='1.26.5'

devinstall_die() {
  echo "error: $*" >&2
  exit 1
}

devinstall_fail() {
  devinstall_die "$@"
}

devinstall_log_stage() {
  printf '[%s] %s\n' "$1" "$2"
}

devinstall_repo_root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  (cd "$script_dir/.." && pwd)
}

devinstall_host_os() {
  local u
  u="$(uname -s)"
  case "$u" in
    Linux*) echo linux ;;
    Darwin*) echo darwin ;;
    MINGW*|MSYS*|CYGWIN*) echo windows ;;
    *) devinstall_die "unsupported host OS: $u" ;;
  esac
}

devinstall_host_arch() {
  local a
  a="$(uname -m)"
  case "$a" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) devinstall_die "unsupported host architecture: $a" ;;
  esac
}

devinstall_normalize_goos() {
  case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
    windows|linux|darwin) echo "$1" | tr '[:upper:]' '[:lower:]' ;;
    *) devinstall_die "unsupported GOOS: $1" ;;
  esac
}

devinstall_normalize_goarch() {
  case "$(echo "$1" | tr '[:upper:]' '[:lower:]')" in
    amd64|x86_64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) devinstall_die "unsupported GOARCH: $1" ;;
  esac
}

devinstall_detect_host() {
  DEVINSTALL_HOST_GOOS="$(devinstall_host_os)"
  DEVINSTALL_HOST_GOARCH="$(devinstall_host_arch)"
}

devinstall_resolve_target() {
  local flag_goos="${1:-}" flag_goarch="${2:-}" raw_os raw_arch
  if [[ -n "$flag_goos" ]]; then raw_os="$flag_goos"
  elif [[ -n "${GOOS:-}" ]]; then raw_os="$GOOS"
  else raw_os="$DEVINSTALL_HOST_GOOS"; fi
  if [[ -n "$flag_goarch" ]]; then raw_arch="$flag_goarch"
  elif [[ -n "${GOARCH:-}" ]]; then raw_arch="$GOARCH"
  else raw_arch="$DEVINSTALL_HOST_GOARCH"; fi
  DEVINSTALL_TARGET_GOOS="$(devinstall_normalize_goos "$raw_os")"
  DEVINSTALL_TARGET_GOARCH="$(devinstall_normalize_goarch "$raw_arch")"
  case "$DEVINSTALL_TARGET_GOOS/$DEVINSTALL_TARGET_GOARCH" in
    windows/amd64|windows/arm64|linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) ;;
    *) devinstall_die "unsupported target matrix: $DEVINSTALL_TARGET_GOOS/$DEVINSTALL_TARGET_GOARCH" ;;
  esac
  if [[ "$DEVINSTALL_TARGET_GOOS" == "$DEVINSTALL_HOST_GOOS" && "$DEVINSTALL_TARGET_GOARCH" == "$DEVINSTALL_HOST_GOARCH" ]]; then
    DEVINSTALL_CAN_INSTALL=1
  else
    DEVINSTALL_CAN_INSTALL=0
  fi
}

devinstall_is_emulated() {
  [[ "$DEVINSTALL_TARGET_GOARCH" != "$DEVINSTALL_HOST_GOARCH" ]]
}

devinstall_detect() {
  local goos="${1:-}" goarch="${2:-}"
  devinstall_detect_host
  devinstall_resolve_target "$goos" "$goarch"
  local emulated=0
  if devinstall_is_emulated; then emulated=1; fi
  echo "$DEVINSTALL_HOST_GOOS|$DEVINSTALL_HOST_GOARCH|$DEVINSTALL_TARGET_GOOS|$DEVINSTALL_TARGET_GOARCH|$emulated|$DEVINSTALL_CAN_INSTALL"
}

devinstall_go_mod_version() {
  local repo="$1" line
  line="$(grep -E '^go[[:space:]]+' "$repo/go.mod" | head -n1)" || devinstall_die 'go.mod missing go directive'
  echo "$line" | awk '{print $2}'
}

devinstall_version_ge() {
  local have="$1" want="$2"
  local IFS=.
  local -a h=($have) w=($want)
  local i hv wv
  for i in 0 1 2; do
    hv="${h[$i]:-0}"; wv="${w[$i]:-0}"
    if (( hv > wv )); then return 0; fi
    if (( hv < wv )); then return 1; fi
  done
  return 0
}

devinstall_check_go_version() {
  local required="$1" out ver
  if ! out="$(go version 2>&1)"; then devinstall_die "go not found on PATH: $out"; fi
  if [[ ! "$out" =~ go([0-9]+\.[0-9]+(\.[0-9]+)?) ]]; then devinstall_die "cannot parse go version: $out"; fi
  ver="${BASH_REMATCH[1]}"
  if ! devinstall_version_ge "$ver" "$required"; then
    devinstall_die "go $required or newer required (found $out)"
  fi
  echo "$out"
}

devinstall_check_go() {
  devinstall_check_go_version "$DEVINSTALL_GO_MIN" >/dev/null
}

devinstall_resolve_metadata() {
  local version_override="${1:-}"
  if [[ -n "$version_override" ]]; then
    DEVINSTALL_VERSION="$version_override"
  elif [[ -n "${MEW_VERSION:-}" ]]; then
    DEVINSTALL_VERSION="$MEW_VERSION"
  else
    DEVINSTALL_VERSION='dev'
  fi

  DEVINSTALL_COMMIT=''
  DEVINSTALL_SHORT_COMMIT=''
  DEVINSTALL_DIRTY=0
  DEVINSTALL_BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  local git_commit
  git_commit="$(git rev-parse HEAD 2>/dev/null)" || true
  if [[ -n "$git_commit" ]]; then
    DEVINSTALL_COMMIT="$git_commit"
    DEVINSTALL_SHORT_COMMIT="${git_commit:0:7}"
    local dirty_out
    dirty_out="$(git status --porcelain 2>/dev/null)" || true
    if [[ -n "$dirty_out" ]]; then
      DEVINSTALL_DIRTY=1
    fi
  fi
}

devinstall_check_repo() {
  [[ -d "$DEVINSTALL_REPO_ROOT/cmd/m" && -d "$DEVINSTALL_REPO_ROOT/cmd/mx" ]] || devinstall_die 'cmd/m and cmd/mx must exist under repo root'
  mkdir -p "$DEVINSTALL_REPO_ROOT/bin"
  echo ok >"$DEVINSTALL_REPO_ROOT/bin/.write-test" || devinstall_die "cannot write to bin/"
  rm -f "$DEVINSTALL_REPO_ROOT/bin/.write-test"
}

devinstall_metadata() {
  local repo="$1" version_override="$2"
  devinstall_resolve_metadata "$version_override"
  echo "$DEVINSTALL_VERSION|$DEVINSTALL_COMMIT|$DEVINSTALL_SHORT_COMMIT|$DEVINSTALL_DIRTY|$DEVINSTALL_BUILD_DATE"
}

devinstall_check() {
  local repo="$1" version_override="$2"
  local go_ver
  go_ver="$(devinstall_check_go_version "$(devinstall_go_mod_version "$repo")")"
  devinstall_check_repo
  devinstall_resolve_metadata "$version_override"
  echo "$go_ver|$DEVINSTALL_VERSION|$DEVINSTALL_COMMIT|$DEVINSTALL_SHORT_COMMIT|$DEVINSTALL_DIRTY|$DEVINSTALL_BUILD_DATE|$repo/bin"
}

devinstall_binary_names() {
  local target_os="$1"
  if [[ "$target_os" == windows ]]; then echo 'm.exe|mx.exe'; else echo 'm|mx'; fi
}

devinstall_build() {
  local bin_dir="$DEVINSTALL_REPO_ROOT/bin" m_name mx_name ldflags dirty_str
  IFS='|' read -r m_name mx_name <<<"$(devinstall_binary_names "$DEVINSTALL_TARGET_GOOS")"
  dirty_str='false'
  if [[ "$DEVINSTALL_DIRTY" == 1 ]]; then dirty_str='true'; fi
  ldflags="-X main.version=$DEVINSTALL_VERSION -X main.commit=$DEVINSTALL_COMMIT -X main.shortCommit=$DEVINSTALL_SHORT_COMMIT -X main.dirty=$dirty_str -X main.buildDate=$DEVINSTALL_BUILD_DATE -X main.targetOS=$DEVINSTALL_TARGET_GOOS -X main.targetArch=$DEVINSTALL_TARGET_GOARCH"
  devinstall_log_stage build "CGO_ENABLED=0 go build -> bin/$m_name, bin/$mx_name"
  CGO_ENABLED=0 GOOS="$DEVINSTALL_TARGET_GOOS" GOARCH="$DEVINSTALL_TARGET_GOARCH" \
    go build -ldflags "$ldflags" -o "$bin_dir/$m_name" ./cmd/m || devinstall_die 'go build m failed'
  CGO_ENABLED=0 GOOS="$DEVINSTALL_TARGET_GOOS" GOARCH="$DEVINSTALL_TARGET_GOARCH" \
    go build -ldflags "$ldflags" -o "$bin_dir/$mx_name" ./cmd/mx || devinstall_die 'go build mx failed'

  # Byte-identical copies: os.Args[0] determines logical identity.
  local ext=''
  if [[ "$DEVINSTALL_TARGET_GOOS" == windows ]]; then ext='.exe'; fi
  cp "$bin_dir/$m_name" "$bin_dir/mew$ext"
  cp "$bin_dir/$mx_name" "$bin_dir/mewx$ext"
  chmod +x "$bin_dir/mew$ext" "$bin_dir/mewx$ext" 2>/dev/null || true
  devinstall_log_stage build "created mew$ext (copy of m$ext), mewx$ext (copy of mx$ext)"
}

devinstall_default_paths() {
  local install_dir="${1:-}"
  if [[ -n "$install_dir" ]]; then
    echo "$install_dir|$(dirname "$install_dir")/completions"
    return
  fi
  local data="${XDG_DATA_HOME:-$HOME/.local/share}"
  echo "$data/mewjs/bin|$data/mewjs/completions"
}

devinstall_default_install_dir_unix() {
  devinstall_default_paths '' | cut -d'|' -f1
}

devinstall_completion_base_from_install_dir() {
  echo "$(dirname "$1")/completions"
}

devinstall_copy_atomic() {
  local src="$1" dest="$2" tmp dir
  dir="$(dirname "$dest")"
  mkdir -p "$dir"
  tmp="${dest}.tmp.$$"
  cp "$src" "$tmp"
  mv "$tmp" "$dest"
}

devinstall_install_unix() {
  local repo="$1" bin_dir="$2" m_name="$3" mx_name="$4" install_dir="$5" force="$6"
  local completion_root paths_line mew_name mewx_name ext
  paths_line="$(devinstall_default_paths "$install_dir")"
  IFS='|' read -r install_dir completion_root <<<"$paths_line"
  ext=''
  if [[ "$DEVINSTALL_TARGET_GOOS" == windows ]]; then ext='.exe'; fi
  mew_name="mew$ext"
  mewx_name="mewx$ext"
  mkdir -p "$install_dir"
  devinstall_copy_atomic "$bin_dir/$m_name" "$install_dir/m"
  devinstall_copy_atomic "$bin_dir/$mx_name" "$install_dir/mx"
  devinstall_copy_atomic "$bin_dir/$mew_name" "$install_dir/mew"
  devinstall_copy_atomic "$bin_dir/$mewx_name" "$install_dir/mewx"
  chmod +x "$install_dir/m" "$install_dir/mx" "$install_dir/mew" "$install_dir/mewx" 2>/dev/null || true
  echo "$install_dir|$completion_root"
}

devinstall_install_unix_files() {
  local install_dir="$1" force="$2"
  local bin_dir="$DEVINSTALL_REPO_ROOT/bin" m_name mx_name ext mew_name mewx_name
  IFS='|' read -r m_name mx_name <<<"$(devinstall_binary_names "$DEVINSTALL_TARGET_GOOS")"
  ext=''
  if [[ "$DEVINSTALL_TARGET_GOOS" == windows ]]; then ext='.exe'; fi
  mew_name="mew$ext"
  mewx_name="mewx$ext"
  mkdir -p "$install_dir"
  devinstall_copy_atomic "$bin_dir/$m_name" "$install_dir/m"
  devinstall_copy_atomic "$bin_dir/$mx_name" "$install_dir/mx"
  devinstall_copy_atomic "$bin_dir/$mew_name" "$install_dir/mew"
  devinstall_copy_atomic "$bin_dir/$mewx_name" "$install_dir/mewx"
  chmod +x "$install_dir/m" "$install_dir/mx" "$install_dir/mew" "$install_dir/mewx" 2>/dev/null || true
}

devinstall_upsert_managed_block() {
  local file="$1" start="$2" end="$3" body="$4"
  local content block
  if [[ -f "$file" ]]; then content="$(cat "$file")"; else content=''; fi
  block="${start}
${body}
${end}"
  if [[ "$content" == *"$start"* && "$content" == *"$end"* ]]; then
    awk -v s="$start" -v e="$end" -v b="$body" '
      BEGIN { inb=0; printed=0 }
      {
        if ($0 ~ s) { inb=1; if (!printed) { print s; print b; print e; printed=1 }; next }
        if ($0 ~ e) { inb=0; next }
        if (!inb) print
      }' <<<"$content" >"$file"
    return
  fi
  if [[ -n "$content" && "${content: -1}" != $'\n' ]]; then content+=$'\n'; fi
  printf '%s%s\n' "$content" "$block" >"$file"
}

devinstall_remove_managed_block() {
  local file="$1" start="$2" end="$3"
  [[ -f "$file" ]] || return 0
  awk -v s="$start" -v e="$end" '
    $0 ~ s { skip=1; next }
    $0 ~ e { skip=0; next }
    !skip { print }
  ' "$file" >"${file}.tmp" && mv "${file}.tmp" "$file"
}

devinstall_shell_profile() {
  local shell_name
  shell_name="$(basename "${SHELL:-bash}")"
  case "$shell_name" in
    zsh)
      [[ -f "$HOME/.zshrc" ]] && echo "$HOME/.zshrc" && return
      [[ -f "$HOME/.zprofile" ]] && echo "$HOME/.zprofile" && return
      echo "$HOME/.zshrc"
      ;;
    fish)
      echo "${XDG_CONFIG_HOME:-$HOME/.config}/fish/config.fish"
      ;;
    *)
      [[ -f "$HOME/.bashrc" ]] && echo "$HOME/.bashrc" && return
      [[ -f "$HOME/.bash_profile" ]] && echo "$HOME/.bash_profile" && return
      [[ -f "$HOME/.profile" ]] && echo "$HOME/.profile" && return
      echo "$HOME/.bashrc"
      ;;
  esac
}

devinstall_path_unix() {
  local install_dir="$1"
  devinstall_upsert_unix_path "$install_dir"
}

devinstall_upsert_unix_path() {
  local install_dir="$1" profile body shell_name
  profile="$(devinstall_shell_profile)"
  shell_name="$(basename "${SHELL:-bash}")"
  mkdir -p "$(dirname "$profile")"
  if [[ "$shell_name" == fish ]]; then
    body="fish_add_path \"$install_dir\""
  else
    body="export PATH=\"$install_dir:\$PATH\""
  fi
  devinstall_upsert_managed_block "$profile" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END" "$body"
}

devinstall_write_utf8() {
  local path="$1" content="$2"
  printf '%s' "$content" >"$path"
}

devinstall_generate_completions_unix() {
  local install_dir="$1" completion_root="$2"
  local bash_dir zsh_dir fish_dir profile body shell_name m_bin mx_bin
  bash_dir="$completion_root/bash"
  zsh_dir="$completion_root/zsh"
  fish_dir="$completion_root/fish"
  m_bin="$install_dir/m"
  mx_bin="$install_dir/mx"
  mkdir -p "$bash_dir" "$zsh_dir" "$fish_dir"
  for shell in bash zsh fish; do
    for pair in m:"$m_bin" mx:"$mx_bin"; do
      local pair_base pair_bin out file
      pair_base="${pair%%:*}"
      pair_bin="${pair##*:}"
      out="$("$pair_bin" completion "$shell")" || devinstall_die "completion failed for $pair_base $shell"
      [[ -n "$out" ]] || devinstall_die "empty completion output for $pair_base $shell"
      case "$shell" in
        bash) file="$bash_dir/$pair_base" ;;
        zsh) file="$zsh_dir/_$pair_base" ;;
        fish) file="$fish_dir/$pair_base.fish" ;;
      esac
      devinstall_write_utf8 "$file" "$out"
    done
  done
  DEVINSTALL_COMPLETION_BASE="$completion_root"
}

devinstall_upsert_unix_completion_profile() {
  local completion_root="$1"
  local bash_dir zsh_dir fish_dir profile body shell_name
  bash_dir="$completion_root/bash"
  zsh_dir="$completion_root/zsh"
  fish_dir="$completion_root/fish"
  profile="$(devinstall_shell_profile)"
  shell_name="$(basename "${SHELL:-bash}")"
  case "$shell_name" in
    zsh)
      body="fpath=($zsh_dir \$fpath)
autoload -Uz compinit
if ! grep -q compinit \"$profile\" 2>/dev/null; then compinit; fi
compdef _m mew
compdef _mx mewx"
      ;;
    fish)
      mkdir -p "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions"
      ln -sf "$fish_dir/m.fish" "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/m.fish"
      ln -sf "$fish_dir/mx.fish" "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/mx.fish"
      printf '%s\n' "complete -c mew -w m" >"$fish_dir/mew.fish"
      printf '%s\n' "complete -c mewx -w mx" >"$fish_dir/mewx.fish"
      body="# fish completions installed under $fish_dir"
      ;;
    *)
      body="[[ -r $bash_dir/m ]] && source $bash_dir/m
[[ -r $bash_dir/mx ]] && source $bash_dir/mx
complete -o default -F _m mew
complete -o default -F _mx mewx"
      ;;
  esac
  devinstall_upsert_managed_block "$profile" "$DEVINSTALL_COMPLETION_START" "$DEVINSTALL_COMPLETION_END" "$body"
}

devinstall_completion_unix() {
  devinstall_generate_completions_unix "$1" "$2"
  devinstall_upsert_unix_completion_profile "$2"
}

devinstall_verify_unix() {
  local install_dir="$1" completion_root="$2"
  local expected_target="$DEVINSTALL_TARGET_GOOS/$DEVINSTALL_TARGET_GOARCH"
  for cmd in m mx mew mewx; do
    local bin="$install_dir/$cmd"
    if [[ ! -x "$bin" ]]; then
      devinstall_die "verify failed: missing $bin"
    fi
    local json_out
    json_out="$("$bin" version --json 2>&1)" || devinstall_die "verify failed: $cmd version --json (exit $?)"
    local parsed
    parsed="$(python3 -c "
import json, sys
try:
    d = json.loads(sys.argv[1])
except Exception as e:
    sys.exit('invalid JSON: ' + str(e))
binary = d.get('binary', '')
version = d.get('version', '')
commit = d.get('commit', '')
target = d.get('target_os', '') + '/' + d.get('target_arch', '')
print('|'.join([binary, version, commit, target]))
" "$json_out")" || devinstall_die "verify failed: invalid JSON from $cmd version --json"

    IFS='|' read -r json_binary json_version json_commit json_target <<<"$parsed"
    if [[ "$json_binary" != "$cmd" ]]; then
      devinstall_die "identity mismatch for ${cmd}: expected $cmd, got $json_binary"
    fi
    if [[ "$json_version" != "$DEVINSTALL_VERSION" ]]; then
      devinstall_die "version mismatch for ${cmd}: expected $DEVINSTALL_VERSION, got $json_version"
    fi
    if [[ -n "$DEVINSTALL_COMMIT" && "$json_commit" != "$DEVINSTALL_COMMIT" ]]; then
      devinstall_die "commit mismatch for ${cmd}: expected $DEVINSTALL_COMMIT, got $json_commit"
    fi
    if [[ "$json_target" != "$expected_target" ]]; then
      devinstall_die "target mismatch for ${cmd}: expected $expected_target, got $json_target"
    fi
  done

  devinstall_log_stage verify 'm, mew, mx, and mewx identities ok'

  if [[ -n "$completion_root" ]]; then
    for f in bash/m bash/mx zsh/_m zsh/_mx fish/m.fish fish/mx.fish; do
      if [[ ! -s "$completion_root/$f" ]]; then
        devinstall_die "verify failed: missing completion $completion_root/$f"
      fi
    done
  fi

  if ! command -v m >/dev/null 2>&1; then
    devinstall_log_stage verify 'warning: m not on current shell PATH (restart terminal)'
  fi
}

devinstall_uninstall_unix() {
  local install_dir="$1" completion_root="$2" keep_path="$3" keep_completion="$4"
  local owned f profile
  owned=(m mx mew mewx)
  if [[ -d "$install_dir" ]]; then
    for f in "${owned[@]}"; do rm -f "$install_dir/$f"; done
    rmdir "$install_dir" 2>/dev/null || true
  fi
  if [[ "$keep_completion" != 1 && -n "$completion_root" && -d "$completion_root" ]]; then
    rm -rf "$completion_root"
    profile="$(devinstall_shell_profile)"
    devinstall_remove_managed_block "$profile" "$DEVINSTALL_COMPLETION_START" "$DEVINSTALL_COMPLETION_END"
  fi
  if [[ "$keep_path" != 1 ]]; then
    profile="$(devinstall_shell_profile)"
    devinstall_remove_managed_block "$profile" "$DEVINSTALL_PATH_START" "$DEVINSTALL_PATH_END"
  fi
}

devinstall_print_summary() {
  local dirty_label='false'
  if [[ "$DEVINSTALL_DIRTY" == 1 ]]; then dirty_label='true'; fi
  cat <<EOF

MewJS development install summary
  repo:        $DEVINSTALL_REPO_ROOT
  source:      working tree
  target:      $DEVINSTALL_TARGET_GOOS/$DEVINSTALL_TARGET_GOARCH
  version:     $DEVINSTALL_VERSION
  commit:      $DEVINSTALL_COMMIT
  short:       $DEVINSTALL_SHORT_COMMIT
  dirty:       $dirty_label
  build date:  $DEVINSTALL_BUILD_DATE
  install dir: ${DEVINSTALL_INSTALL_DIR:-<build-only>}
  completion:  ${DEVINSTALL_COMPLETION_BASE:-<skipped>}
EOF
}

devinstall_summary() {
  devinstall_print_summary
}
