package resolver

import (
	"time"

	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/policy"
)

// PolicyFromEffective returns resolver policy from effective config only.
// Includes every graph-affecting field; excludes script trust and linker.
func PolicyFromEffective(eff *config.Effective) *policy.Policy {
	pol := policy.Policy{StrictPeerDependencies: true}
	if eff != nil {
		pol.AutoInstallPeers = config.Bool(eff, "resolve.autoInstallPeers", false)
		pol.StrictPeerDependencies = config.Bool(eff, "resolve.strictPeerDependencies", true)
		pol.RejectDeprecated = config.Bool(eff, "resolve.rejectDeprecated", false)
		pol.MinimumReleaseAge = time.Duration(config.Int(eff, "resolve.minimumReleaseAge", 0)) * time.Millisecond
		pol.Offline = config.Bool(eff, "offline", false)
	}
	_ = pol.Normalize()
	return &pol
}
