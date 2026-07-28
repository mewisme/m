package registry

import (
	"fmt"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/semver"
)

// SelectMaxSatisfying resolves a version or dist-tag, else the highest
// semver-compatible version for an npm range.
func SelectMaxSatisfying(p *Packument, spec string) (*VersionMeta, error) {
	if p == nil {
		return nil, apperr.New(apperr.Network, "registry.select", "", "nil packument")
	}
	if meta, err := p.SelectVersion(spec); err == nil {
		return meta, nil
	}
	vers := p.SortedVersionsSemver()
	best, err := semver.MaxSatisfying(vers, spec)
	if err != nil {
		return nil, apperr.New(apperr.Resolve, "registry.select", p.Name+"@"+spec,
			fmt.Sprintf("no version satisfies %q", spec))
	}
	meta, ok := p.Versions[best]
	if !ok {
		return nil, apperr.New(apperr.NotFound, "registry.select", p.Name+"@"+best,
			fmt.Sprintf("version %q not in packument", best))
	}
	return &meta, nil
}

// SortedVersionsSemver returns version keys ordered by semver ascending.
// Non-parseable versions sort after valid ones, then lexicographically.
func (p *Packument) SortedVersionsSemver() []string {
	keys := make([]string, 0, len(p.Versions))
	for k := range p.Versions {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		cmp, err := semver.Compare(keys[i], keys[j])
		if err != nil {
			return keys[i] < keys[j]
		}
		return cmp < 0
	})
	return keys
}

// AbsoluteTarballURL joins a registry base with a relative tarball path.
func AbsoluteTarballURL(base, name, tarball string) string {
	return absoluteTarballURL(base, name, tarball)
}
