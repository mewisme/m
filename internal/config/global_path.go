package config

import (
	"path/filepath"
	"strings"
)

// GlobalConfigPathFromEnv resolves the user config.jsonc path from a frozen env snapshot.
func GlobalConfigPathFromEnv(env []string, goos string) string {
	m := envMap(env)
	if d := m["MEW_CONFIG_DIR"]; d != "" {
		return absJoin(d, "config.jsonc")
	}
	if home := m["MEW_HOME"]; home != "" {
		return absJoin(home, "config", "config.jsonc")
	}
	if goos == "windows" {
		base := m["AppData"]
		if base == "" {
			base = filepath.Join(m["USERPROFILE"], "AppData", "Roaming")
		}
		return absJoin(base, "mew", "config.jsonc")
	}
	cfg := m["XDG_CONFIG_HOME"]
	if cfg == "" {
		cfg = filepath.Join(userHomeFromEnv(m), ".config")
	}
	return absJoin(cfg, "mew", "config.jsonc")
}

func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		key, val, ok := strings.Cut(e, "=")
		if !ok {
			m[key] = ""
			continue
		}
		m[key] = val
	}
	return m
}

func userHomeFromEnv(m map[string]string) string {
	if h := m["HOME"]; h != "" {
		return h
	}
	return m["USERPROFILE"]
}

func absJoin(elem ...string) string {
	p := filepath.Join(elem...)
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
