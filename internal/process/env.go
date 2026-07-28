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
