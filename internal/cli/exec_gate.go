package cli

import (
	"os"

	"github.com/mewisme/mew/internal/config"
)

// DirectDispatchBinsEnabled reports whether direct m <bin> shortcuts are enabled.
func DirectDispatchBinsEnabled(eff *config.Effective) bool {
	if config.Bool(eff, "runner.exec.direct_dispatch.enabled", false) {
		return true
	}
	if eff != nil && eff.Env.Initialized() {
		if v, ok := eff.Env.Lookup("MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH"); ok && v == "1" {
			return true
		}
	}
	return os.Getenv("MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH") == "1"
}
