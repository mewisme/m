package resolver

import (
	"strings"

	"github.com/mewisme/mew/internal/manifest"
)

// matchOverride returns an npm-style override specifier for depName under parentPath.
// Longest matching dotted path wins (nearest nested override).
func matchOverride(overrides map[string]string, parentPath []string, depName string) (string, bool) {
	if len(overrides) == 0 {
		return "", false
	}
	for i := 0; i <= len(parentPath); i++ {
		parts := append(append([]string(nil), parentPath[i:]...), depName)
		key := strings.Join(parts, ".")
		if spec, ok := overrides[key]; ok {
			return spec, true
		}
	}
	return "", false
}

// rewriteSpecifier applies overrides then parses the dependency specifier.
func rewriteSpecifier(overrides map[string]string, parentPath []string, displayName, spec string) (display, target, rng string, protocol manifest.Protocol, err error) {
	if ovr, ok := matchOverride(overrides, parentPath, displayName); ok {
		spec = ovr
	}
	return parseDependencySpecifier(displayName, spec)
}
