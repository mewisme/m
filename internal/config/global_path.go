package config

import "path/filepath"

// GlobalConfigPathFromEnv resolves the user config.jsonc path from a frozen env snapshot.
func GlobalConfigPathFromEnv(snap EnvSnapshot) string {
	if d, ok := snap.Lookup("MEW_CONFIG_DIR"); ok && d != "" {
		return absJoin(d, fileUserConfig)
	}
	if home, ok := snap.Lookup("MEW_HOME"); ok && home != "" {
		return absJoin(home, dirConfig, fileUserConfig)
	}
	if snap.GOOS() == "windows" {
		return absJoin(productDir(windowsRoamingAppData(snap), fileUserConfig))
	}
	cfg, ok := snap.Lookup("XDG_CONFIG_HOME")
	if !ok || cfg == "" {
		cfg = filepath.Join(userHomeFromSnap(snap), ".config")
	}
	return absJoin(productDir(cfg, fileUserConfig))
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
