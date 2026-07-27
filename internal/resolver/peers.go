package resolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/registry"
	"github.com/mewisme/m/internal/semver"
)

// PeerConflict describes an unsatisfied peer dependency.
type PeerConflict struct {
	Package string `json:"package"`
	Peer    string `json:"peer"`
	Range   string `json:"range"`
}

type peerConflictErr struct {
	PeerConflict
}

func (e *peerConflictErr) Error() string {
	return e.PeerConflict.Error()
}

func (c PeerConflict) Error() string {
	return fmt.Sprintf("missing peer %s@%q required by %s", c.Peer, c.Range, c.Package)
}

func peerConflictError(pkg, peer, rng string) error {
	conflict := PeerConflict{Package: pkg, Peer: peer, Range: rng}
	return apperr.Wrap(apperr.Resolve, "resolver.peer", pkg, &peerConflictErr{conflict})
}

func peerContextFromMeta(meta *registry.VersionMeta) graph.PeerContext {
	if meta == nil || len(meta.PeerDependencies) == 0 {
		return nil
	}
	pc := make(graph.PeerContext, 0, len(meta.PeerDependencies))
	for name, rng := range meta.PeerDependencies {
		pc = append(pc, graph.PeerRef{Name: name, Range: rng})
	}
	pc.Sort()
	return pc
}

func peerOptional(meta *registry.VersionMeta, peerName string) bool {
	if meta == nil || meta.PeerDependenciesMeta == nil {
		return false
	}
	entry, ok := meta.PeerDependenciesMeta[peerName]
	return ok && entry.Optional
}

func (s *resolveState) peerSatisfied(peerName, rng string) bool {
	_, ok := s.findResolvedPeer(peerName, rng)
	return ok
}

func (s *resolveState) findResolvedPeer(peerName, rng string) (string, bool) {
	var best string
	for key := range s.seenPkg {
		id := parsePackageKey(key)
		if id.Name != peerName {
			continue
		}
		ok, err := semver.Satisfies(id.Version, rng)
		if err != nil || !ok {
			continue
		}
		if best == "" || versionGT(id.Version, best) {
			best = id.Version
		}
	}
	return best, best != ""
}

func (s *resolveState) peerPending(peerName string) bool {
	for _, item := range s.queue {
		if item.name == peerName {
			return true
		}
	}
	for key := range s.queuedEdge {
		parts := strings.Split(key, "\x00")
		if len(parts) >= 3 && parts[2] == peerName {
			return true
		}
	}
	return false
}

func (s *resolveState) resolvePeers(pkgName string, meta *registry.VersionMeta, pol *policy.Policy) error {
	if meta == nil || len(meta.PeerDependencies) == 0 {
		return nil
	}
	for _, peer := range sortedPeerDeps(meta.PeerDependencies) {
		if peerOptional(meta, peer.name) {
			continue
		}
		if s.peerSatisfied(peer.name, peer.rng) || s.peerPending(peer.name) {
			continue
		}
		if pol.AutoInstallPeers {
			if err := s.enqueue(string(graph.RootImporter), ".", peer.name, peer.rng, graph.DepProd, 1, nil, false); err != nil {
				return err
			}
			continue
		}
		if pol.StrictPeerDependencies {
			return peerConflictError(pkgName, peer.name, peer.rng)
		}
	}
	return nil
}

func (s *resolveState) validatePeers(pol *policy.Policy) error {
	for key, peers := range s.pkgPeers {
		id := parsePackageKey(key)
		opt := s.pkgPeerOpt[key]
		for peerName, peerRange := range peers {
			if opt != nil && opt[peerName] {
				continue
			}
			if s.peerSatisfied(peerName, peerRange) {
				continue
			}
			if pol.AutoInstallPeers {
				continue
			}
			if pol.StrictPeerDependencies {
				return peerConflictError(id.Name, peerName, peerRange)
			}
		}
	}
	return nil
}

type peerDep struct {
	name, rng string
}

func sortedPeerDeps(m map[string]string) []peerDep {
	if len(m) == 0 {
		return nil
	}
	out := make([]peerDep, 0, len(m))
	for k, v := range m {
		out = append(out, peerDep{name: k, rng: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func parsePackageKey(key string) graph.PackageID {
	var pc graph.PeerContext
	base := key
	if i := strings.IndexByte(key, '#'); i >= 0 {
		base = key[:i]
		for _, part := range strings.Split(key[i+1:], ",") {
			if part == "" {
				continue
			}
			at := strings.LastIndexByte(part, '@')
			if at <= 0 {
				continue
			}
			pc = append(pc, graph.PeerRef{Name: part[:at], Range: part[at+1:]})
		}
		pc.Sort()
	}
	name, ver := splitNameVersion(base)
	return graph.PackageID{Name: name, Version: ver, PeerContext: pc}
}

func splitNameVersion(base string) (name, version string) {
	if strings.HasPrefix(base, "@") {
		slash := strings.IndexByte(base, '/')
		if slash < 0 {
			return base, ""
		}
		at := strings.LastIndexByte(base[slash:], '@')
		if at < 0 {
			return base, ""
		}
		return base[:slash+at], base[slash+at+1:]
	}
	at := strings.LastIndexByte(base, '@')
	if at < 0 {
		return base, ""
	}
	return base[:at], base[at+1:]
}

func versionGT(a, b string) bool {
	cmp, err := semver.Compare(a, b)
	if err != nil {
		return a > b
	}
	return cmp > 0
}
