package lifecycle

import (
	"os"

	"github.com/mewisme/m/internal/config"
)

// Enabled reports whether lifecycle execution is turned on for this invocation.
func Enabled(eff *config.Effective) bool {
	if config.Bool(eff, "lifecycle.enabled", false) {
		return true
	}
	if eff != nil && eff.Env.Initialized() {
		if v, ok := eff.Env.Lookup("MEW_EXPERIMENTAL_LIFECYCLE"); ok && v == "1" {
			return true
		}
	}
	return os.Getenv("MEW_EXPERIMENTAL_LIFECYCLE") == "1"
}
