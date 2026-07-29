package resolver

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/manifest"
)

type patchState struct {
	bySelector map[string]patchRecord
	byPkgKey   map[string]patchRecord
}

type patchRecord struct {
	path string
	hash string
}

func (s *resolveState) initPatches() error {
	entries, err := manifest.PatchedDependencies(s.proj.Doc)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	s.patches = &patchState{
		bySelector: make(map[string]patchRecord, len(entries)),
		byPkgKey:   map[string]patchRecord{},
	}
	for _, e := range entries {
		abs, err := manifest.ResolvePatchPath(s.proj.Root, e.Path)
		if err != nil {
			return apperr.Wrap(apperr.Resolve, "resolver.patch", e.Selector, err)
		}
		hash, err := s.patchHashForFile(abs, e.Selector)
		if err != nil {
			return err
		}
		s.patches.bySelector[e.Selector] = patchRecord{path: abs, hash: hash}
	}
	return nil
}

func (s *resolveState) patchHashForFile(abs, selector string) (string, error) {
	data, err := manifest.ReadPatchFile(abs)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "resolver.patch", abs, err)
	}
	major := s.pnpmMajor
	if major == 0 {
		major = 10
	}
	computed, err := pnpm.ComputePatchHash(major, data)
	if err != nil {
		return "", apperr.Wrap(apperr.Resolve, "resolver.patch", selector, err)
	}
	if prior := patchHashFromPriorGraph(s.hints.g, selector); prior != "" && prior != computed {
		return computed, apperr.New(apperr.Resolve, "resolver.patch", selector,
			fmt.Sprintf("patch bytes hash %q does not match prior lock hash %q", computed, prior))
	}
	return computed, nil
}

func patchHashFromPriorGraph(g *graph.Graph, selector string) string {
	if g == nil {
		return ""
	}
	name, ver, ok := strings.Cut(selector, "@")
	if !ok {
		return ""
	}
	for _, p := range g.Packages {
		if p.ID.Name != name {
			continue
		}
		if !strings.HasPrefix(p.ID.Version, ver+"(patch_hash=") {
			continue
		}
		if h := extractPatchHash(p.ID.Version); h != "" {
			return h
		}
	}
	return ""
}

func extractPatchHash(version string) string {
	const prefix = "(patch_hash="
	if !strings.Contains(version, prefix) {
		return ""
	}
	i := strings.Index(version, prefix)
	if i < 0 {
		return ""
	}
	rest := version[i+len(prefix):]
	if j := strings.IndexByte(rest, ')'); j >= 0 {
		return rest[:j]
	}
	return ""
}

func (s *resolveState) applyPatchVersion(name, version string) string {
	suffix := s.patchSuffix(name, version)
	if suffix == "" {
		return version
	}
	return stripPatchParenthetical(version) + suffix
}

func (s *resolveState) patchSuffix(name, version string) string {
	base := stripPatchParenthetical(version)
	selector := name + "@" + base
	if s.patches != nil {
		if rec, ok := s.patches.bySelector[selector]; ok {
			if rec.hash != "" {
				return "(patch_hash=" + rec.hash + ")"
			}
		}
	}
	if s.hints.g != nil {
		for _, p := range s.hints.g.Packages {
			if p.ID.Name != name {
				continue
			}
			if strings.HasPrefix(p.ID.Version, base+"(patch_hash=") {
				if i := strings.IndexByte(p.ID.Version, '('); i >= 0 {
					return p.ID.Version[i:]
				}
			}
		}
	}
	return ""
}

func stripPatchParenthetical(version string) string {
	if i := strings.IndexByte(version, '('); i >= 0 {
		return version[:i]
	}
	return version
}

func (s *resolveState) finalizePatchTargets(g *graph.Graph) error {
	if s.patches == nil || g == nil {
		return nil
	}
	matched := make(map[string]bool, len(s.patches.bySelector))
	remap := map[string]string{}
	for i := range g.Packages {
		p := &g.Packages[i]
		selector := p.ID.Name + "@" + stripPatchParenthetical(p.ID.Version)
		rec, ok := s.patches.bySelector[selector]
		if !ok || rec.hash == "" {
			continue
		}
		matched[selector] = true
		patchedVer := stripPatchParenthetical(p.ID.Version) + "(patch_hash=" + rec.hash + ")"
		if p.ID.Version != patchedVer {
			oldKey := p.ID.Key()
			p.ID = graph.PackageID{
				Name:                p.ID.Name,
				Version:             patchedVer,
				PeerProviderContext: p.ID.PeerProviderContext,
			}
			p.ID.Normalize()
			remap[oldKey] = p.ID.Key()
		}
		s.patches.byPkgKey[p.ID.Key()] = rec
	}
	for i := range g.Edges {
		if to, ok := remap[g.Edges[i].To]; ok {
			g.Edges[i].To = to
		}
	}
	for selector := range s.patches.bySelector {
		if !matched[selector] {
			return apperr.New(apperr.Resolve, "resolver.patch", selector,
				"patch declared in manifest but no matching resolved package")
		}
	}
	return nil
}

func (s *resolveState) recordPatchTarget(pkgKey, name, version string) {
	if s.patches == nil {
		return
	}
	selector := name + "@" + stripPatchParenthetical(version)
	rec, ok := s.patches.bySelector[selector]
	if !ok {
		return
	}
	s.patches.byPkgKey[pkgKey] = rec
}
