package config

import "path/filepath"

// GlobalConfigPathFromEnv resolves the user config.jsonc path from a frozen env snapshot.
func GlobalConfigPathFromEnv(snap EnvSnapshot) string {
	if d, ok := snap.Lookup("MEW_CONFIG_DIR"); ok && d != "" {
		return absJoin(d, "config.jsonc")
	}
	if home, ok := snap.Lookup("MEW_HOME"); ok && home != "" {
		return absJoin(home, "config", "config.jsonc")
	}
	if snap.GOOS() == "windows" {
		base, _ := snap.Lookup("APPDATA")
		if base == "" {
			profile, _ := snap.Lookup("USERPROFILE")
			base = filepath.Join(profile, "AppData", "Roaming")
		}
		return absJoin(base, "mew", "config.jsonc")
	}
	cfg, ok := snap.Lookup("XDG_CONFIG_HOME")
	if !ok || cfg == "" {
		cfg = filepath.Join(userHomeFromSnap(snap), ".config")
	}
	return absJoin(cfg, "mew", "config.jsonc")
}

func userHomeFromSnap(snap EnvSnapshot) string {
	if h, ok := snap.Lookup("HOME"); ok && h != "" {
		return h
	}
	h, _ := snap.Lookup("USERPROFILE")
	return h
}

func absJoin(elem ...string) string {
	p := filepath.Join(elem...)
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
