package resolver

import (
	"github.com/mewisme/mew/internal/registry"
)

// platformSkipsOptional reports whether an optional dependency should be skipped
// for the current platform after version selection.
func platformSkipsOptional(optional bool, meta *registry.VersionMeta, target Target) bool {
	if !optional || meta == nil {
		return false
	}
	return !target.Matches(*meta)
}
