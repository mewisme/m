package process

import (
	"os"
	"runtime"
	"strings"
)

// EnvSource describes explicit vs unset environment intent for RestrictedEnv.
type EnvSource struct {
	Vars     []string // nil = unset; non-nil (incl. empty) = explicit
	Explicit bool
}

// RestrictedEnv copies base env, prepends binDir to PATH, and strips secrets.
// When Explicit is true and Vars is empty, output contains only controlled PATH/Path.
// When Explicit is false, base falls back to os.Environ() for compatibility.
func RestrictedEnv(src EnvSource, binDir string) []string {
	base := src.Vars
	if !src.Explicit {
		base = os.Environ()
	}
	out := make([]string, 0, len(base)+1)
	for _, kv := range base {
		key := envKey(kv)
		if shouldStripEnv(key) {
			continue
		}
		if strings.EqualFold(key, "PATH") {
			continue
		}
		out = append(out, kv)
	}
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	pathVal := binDir
	if old, ok := lookupEnv(base, pathKey); ok && old != "" {
		sep := string(os.PathListSeparator)
		pathVal = binDir + sep + old
	}
	out = append(out, pathKey+"="+pathVal)
	return out
}

// StripGitWorktreeEnv removes parent-repository git metadata inherited from CI
// checkouts so nested git subprocesses operate only on their own -C directory.
func StripGitWorktreeEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		switch strings.ToUpper(envKey(kv)) {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_PREFIX", "GIT_COMMON_DIR":
			continue
		}
		out = append(out, kv)
	}
	return out
}

// GitSubprocessEnv returns a restricted environment for nested git commands.
func GitSubprocessEnv(binDir string) []string {
	return StripGitWorktreeEnv(append(RestrictedEnv(EnvSource{}, binDir),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0=*",
		"GIT_CONFIG_KEY_1=protocol.file.allow",
		"GIT_CONFIG_VALUE_1=always",
	))
}
