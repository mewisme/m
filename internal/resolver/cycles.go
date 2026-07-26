package resolver

import (
	"fmt"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// cycleError builds ERR_M_RESOLVE naming the full cycle path.
func cycleError(path []string, name string) error {
	full := append(append([]string(nil), path...), name)
	return apperr.New(apperr.Resolve, "resolver.cycle", name,
		fmt.Sprintf("dependency cycle: %s", strings.Join(full, " → ")))
}

func pathContains(path []string, name string) bool {
	for _, p := range path {
		if p == name {
			return true
		}
	}
	return false
}
