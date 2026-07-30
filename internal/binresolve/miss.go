package binresolve

import "fmt"

// MissMessage returns the dependency-aware miss diagnostic (never guesses package from command).
func MissMessage(command string) string {
	return fmt.Sprintf(`No local binary named %q is visible from this importer.

Install the package that provides it, or specify an importer-visible dependency with:
  m exec --package <dependency> %s`, command, command)
}

// AmbiguityMessage formats same-level metadata ambiguity with sorted --package hints.
func AmbiguityMessage(command string, deps []string) string {
	msg := fmt.Sprintf("ambiguous local binary %q: multiple importer-visible dependencies provide it", command)
	if len(deps) == 0 {
		return msg
	}
	msg += "\n\nSpecify one with --package:"
	for _, dep := range deps {
		msg += fmt.Sprintf("\n  m exec --package %s %s", dep, command)
	}
	return msg
}
