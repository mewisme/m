package resolver

import (
	"time"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/policy"
)

// PolicyFromEffective returns resolver policy from effective config only.
// Includes every graph-affecting field; excludes script trust and linker.
func PolicyFromEffective(eff *config.Effective) *policy.Policy {
	pol := policy.Policy{StrictPeerDependencies: true}
	if eff != nil {
		pol.AutoInstallPeers = config.Bool(eff, "resolve.auto_install_peers", false)
		pol.StrictPeerDependencies = config.Bool(eff, "resolve.strict_peer_dependencies", true)
		pol.RejectDeprecated = config.Bool(eff, "resolve.reject_deprecated", false)
		pol.MinimumReleaseAge = time.Duration(config.Int(eff, "resolve.minimum_release_age", 0)) * time.Millisecond
		pol.Offline = config.Bool(eff, "offline", false)
	}
	_ = pol.Normalize()
	return &pol
}
