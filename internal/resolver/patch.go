package resolver

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
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
		rec := patchRecord{path: abs, hash: patchHashFromPriorGraph(s.hints.g, e.Selector)}
		s.patches.bySelector[e.Selector] = rec
	}
	return nil
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
