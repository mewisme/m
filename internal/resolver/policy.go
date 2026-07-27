package resolver

import (
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/policy"
)

// PolicyFromEffective returns resolver policy from effective config only.
func PolicyFromEffective(eff *config.Effective) *policy.Policy {
	pol := policy.Policy{StrictPeerDependencies: true}
	if eff != nil {
		pol.AutoInstallPeers = config.Bool(eff, "resolve.autoInstallPeers", false)
		pol.StrictPeerDependencies = config.Bool(eff, "resolve.strictPeerDependencies", true)
	}
	_ = pol.Normalize()
	return &pol
}
