package cli

import (
	"path/filepath"
	"strings"
)

// BuildInfo is injected at link time for version output.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// InvokedBinary returns the logical binary name from os.Args[0] (m|mew|mx|mewx).
func InvokedBinary(argv0, fallback string) string {
	base := filepath.Base(argv0)
	base = strings.TrimSuffix(base, ".exe")
	base = strings.TrimSuffix(base, ".EXE")
	switch strings.ToLower(base) {
	case "m", "mew", "mx", "mewx":
		return strings.ToLower(base)
	default:
		return fallback
	}
}

// DisplayName returns the Use string for help (preserves github.com/mewisme/m/mewx aliases).
func DisplayName(invoked string) string {
	switch invoked {
	case "mew":
		return "mew"
	case "mewx":
		return "mewx"
	case "mx":
		return "mx"
	default:
		return "m"
	}
}
