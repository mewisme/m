package resolver

import (
	"fmt"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/semver"
)

// selectVersion chooses a version for name@rng against packument + policy + hints.
func selectVersion(
	p *registry.Packument,
	name, rng string,
	pol *policy.Policy,
	hints *graphHints,
	pin pinContext,
) (meta *registry.VersionMeta, decision ResolutionDecision, err error) {
	decision = ResolutionDecision{
		Package:    name,
		Requested:  rng,
		Candidates: p.SortedVersionsSemver(),
	}

	eligible := make([]string, 0, len(decision.Candidates))
	for _, ver := range decision.Candidates {
		m := p.Versions[ver]
		if reason := policyRejectReason(&m, pol); reason != "" {
			decision.Rejected = append(decision.Rejected, ver+": "+reason)
			continue
		}
		eligible = append(eligible, ver)
	}

	if hints != nil {
		if hv, ok := hints.reusedVersion(pin); ok && containsStr(eligible, hv) {
			m := p.Versions[hv]
			decision.Selected = hv
			decision.Reason = "reuse-key"
			return &m, decision, nil
		}
		if hv := hints.version(name, rng); hv != "" && !hints.incremental {
			if containsStr(eligible, hv) {
				m := p.Versions[hv]
				decision.Selected = hv
				decision.Reason = "hint"
				return &m, decision, nil
			}
			if pkg, ok := hints.pkg(name, hv); ok {
				if reason := policyRejectReason(&registry.VersionMeta{
					Version: hv, Deprecated: "", Time: "",
				}, pol); reason == "" {
					decision.Selected = hv
					decision.Reason = "hint"
					return &registry.VersionMeta{
						Name: name, Version: hv,
						Dist: registry.Dist{Integrity: pkg.Integrity, Tarball: pkg.TarballURL},
					}, decision, nil
				}
			}
		}
	}

	// Exact version or dist-tag on eligible set.
	if exact, selErr := p.SelectVersion(rng); selErr == nil {
		if containsStr(eligible, exact.Version) {
			decision.Selected = exact.Version
			decision.Reason = "tag-or-exact"
			return exact, decision, nil
		}
		if reason := policyRejectReason(exact, pol); reason != "" {
			decision.Rejected = appendUnique(decision.Rejected, exact.Version+": "+reason)
		}
	}

	best, err := semver.MaxSatisfying(eligible, rng)
	if err != nil {
		return nil, decision, apperr.New(apperr.Resolve, "resolver.select", name+"@"+rng,
			fmt.Sprintf("no version satisfies %q", rng))
	}
	m := p.Versions[best]
	decision.Selected = best
	decision.Reason = "max-satisfying"
	return &m, decision, nil
}

func policyRejectReason(m *registry.VersionMeta, pol *policy.Policy) string {
	if pol == nil || m == nil {
		return ""
	}
	if pol.RejectDeprecated && m.Deprecated != "" {
		return "deprecated"
	}
	if pol.MinimumReleaseAge > 0 && m.Time != "" {
		t, err := time.Parse(time.RFC3339, m.Time)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, m.Time)
		}
		if err == nil && time.Since(t) < pol.MinimumReleaseAge {
			return "minimum-release-age"
		}
	}
	return ""
}

func containsStr(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func appendUnique(ss []string, s string) []string {
	if containsStr(ss, s) {
		return ss
	}
	return append(ss, s)
}
