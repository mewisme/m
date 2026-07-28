package workspace

import (
	"os"

	"github.com/mewisme/mew/internal/config"
)

// Enabled reports whether workspace install/filter features are turned on.
func Enabled(eff *config.Effective) bool {
	if config.Bool(eff, "workspaces.enabled", false) {
		return true
	}
	if eff != nil && eff.Env.Initialized() {
		if v, ok := eff.Env.Lookup("MEW_EXPERIMENTAL_WORKSPACES"); ok && v == "1" {
			return true
		}
	}
	return os.Getenv("MEW_EXPERIMENTAL_WORKSPACES") == "1"
}
