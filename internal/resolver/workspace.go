package resolver

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/semver"
	"github.com/mewisme/mew/internal/workspace"
)

type workspaceMember struct {
	Name    string
	Version string
	Path    string // root-relative POSIX
}

func (s *resolveState) initWorkspace() error {
	cat, err := workspace.LoadCatalog(s.proj.Root)
	if err != nil {
		return err
	}
	s.catalog = cat

	patterns, err := s.proj.Doc.WorkspacePatterns()
	if err != nil {
		return err
	}
	if len(patterns) == 0 {
		return nil
	}
	wg, err := workspace.BuildGraph(s.proj.Root)
	if err != nil {
		return err
	}
	s.wsIndex = wg.Index
	s.wsGraph = wg
	s.wsByName = make(map[string]workspaceMember, len(wg.Members))
	s.wsMemberPaths = make(map[string]struct{}, len(wg.ByPath))
	for name, mem := range wg.Members {
		s.wsByName[name] = workspaceMember{Name: mem.Name, Version: mem.Version, Path: mem.Path}
		s.wsMemberPaths[mem.Path] = struct{}{}
	}
	return nil
}

func (s *resolveState) seedWorkspaceMembers(opts ResolveOptions) error {
	if s.wsIndex == nil {
		return nil
	}
	if workspace.Enabled(s.e.Effective) {
		if len(opts.Filter) > 0 {
			return s.seedFilteredMembers(opts.Filter, opts)
		}
		if opts.Recursive {
			return s.seedAllMembers(opts)
		}
		return nil
	}
	return s.seedMembersWithDeps(opts)
}

func (s *resolveState) loadMemberManifest(memPath string, opts ResolveOptions) (*manifest.Document, error) {
	if opts.MemberManifests != nil {
		if doc, ok := opts.MemberManifests[memPath]; ok && doc != nil {
			return doc, nil
		}
	}
	return manifest.Load(filepath.Join(s.proj.Root, filepath.FromSlash(memPath), "package.json"))
}

func (s *resolveState) seedMembersWithDeps(opts ResolveOptions) error {
	paths := append([]string(nil), s.wsIndex.Members...)
	sort.Strings(paths)
	for _, memPath := range paths {
		if memPath == "." || memPath == string(graph.RootImporter) {
			continue
		}
		doc, err := s.loadMemberManifest(memPath, opts)
		if err != nil {
			return err
		}
		if !memberHasDeps(doc) {
			continue
		}
		if err := s.seedMemberImporter(memPath, doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *resolveState) seedAllMembers(opts ResolveOptions) error {
	paths := append([]string(nil), s.wsIndex.Members...)
	sort.Strings(paths)
	for _, memPath := range paths {
		if memPath == "." || memPath == string(graph.RootImporter) {
			continue
		}
		doc, err := s.loadMemberManifest(memPath, opts)
		if err != nil {
			return err
		}
		if err := s.seedMemberImporter(memPath, doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *resolveState) seedFilteredMembers(patterns []string, opts ResolveOptions) error {
	if s.wsGraph == nil {
		return apperr.New(apperr.Internal, "resolver.workspace", "", "missing workspace graph")
	}
	ids, err := workspace.ExpandFilter(s.wsGraph, patterns)
	if err != nil {
		return err
	}
	for _, id := range ids {
		memPath := string(id)
		doc, err := s.loadMemberManifest(memPath, opts)
		if err != nil {
			return err
		}
		if err := s.seedMemberImporter(memPath, doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *resolveState) seedMemberImporter(memPath string, doc *manifest.Document) error {
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return err
	}
	importerID := graph.ImporterID(memPath)
	if !s.seededImporters[importerID] {
		s.b.Importer(importerID, norm.Name)
		s.seededImporters[importerID] = true
	}
	if workspace.Enabled(s.e.Effective) {
		if err := s.ensureWorkspaceMemberPackage(memPath, doc); err != nil {
			return err
		}
	}
	return s.seedDeps(string(importerID), memPath, norm, 1, nil, nil)
}

func (s *resolveState) ensureWorkspaceMemberPackage(memPath string, doc *manifest.Document) error {
	if s.wsByName == nil || doc == nil {
		return nil
	}
	member, ok := s.wsByName[doc.Name]
	if !ok {
		return nil
	}
	key := basePackageKey(member.Name, member.Version)
	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	s.seenPkg[key] = struct{}{}
	s.b.Package(graph.PackageID{Name: member.Name, Version: member.Version}, "", "")
	s.localSources[key] = LocalSource{Protocol: "workspace", Path: member.Path}
	return nil
}

func memberHasDeps(doc *manifest.Document) bool {
	if doc == nil {
		return false
	}
	return len(doc.Dependencies) > 0 || len(doc.DevDependencies) > 0 || len(doc.OptionalDependencies) > 0
}

func (s *resolveState) processWorkspace(item workItem) error {
	if item.depth > maxDepth {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution depth exceeded %d", maxDepth))
	}
	if s.wsByName == nil {
		return apperr.New(apperr.Resolve, "resolver.workspace", item.name,
			fmt.Sprintf("workspace target %q not found", item.name))
	}

	member, ok := s.wsByName[item.name]
	if !ok {
		return apperr.New(apperr.Resolve, "resolver.workspace", item.name,
			fmt.Sprintf("workspace target %q not found", item.name))
	}

	version, edgeRange, err := workspaceVersion(member, item.rng)
	if err != nil {
		return err
	}

	id, key := s.packageKeyForInstance(item, version, nil)

	decision := ResolutionDecision{
		Package:   item.name,
		Requested: "workspace:" + item.rng,
		Selected:  version,
		Reason:    "workspace",
	}
	s.decisions = append(s.decisions, decision)

	if item.spec != "" {
		edgeRange = item.spec
	}
	s.b.EdgeEx(item.from, item.display, key, item.kind, edgeRange, false)
	s.recordProvides(item.from, item.display, key)

	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	if _, ok := s.resolving[basePackageKey(member.Name, version)]; ok {
		return nil
	}
	if len(s.seenPkg) >= maxPackages {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution package count exceeded %d", maxPackages))
	}
	s.resolving[basePackageKey(member.Name, version)] = struct{}{}
	defer delete(s.resolving, basePackageKey(member.Name, version))

	s.seenPkg[key] = struct{}{}
	s.b.Package(id, "", "")
	s.localSources[key] = LocalSource{Protocol: "workspace", Path: member.Path}
	s.pkgEnv[key] = append([]string(nil), item.envKeys...)
	s.pkgFrom[key] = item.from

	return s.expandLocalManifest(key, member.Path, item.depth, item.path, item.envKeys)
}

func workspaceVersion(member workspaceMember, rng string) (version, edgeRange string, err error) {
	switch rng {
	case "*":
		return member.Version, member.Version, nil
	case "^":
		caret := "^" + member.Version
		ok, err := semver.Satisfies(member.Version, caret)
		if err != nil {
			return "", "", apperr.Wrap(apperr.Resolve, "resolver.workspace", member.Name, err)
		}
		if !ok {
			return "", "", apperr.New(apperr.Resolve, "resolver.workspace", member.Name,
				fmt.Sprintf("workspace:^ ambiguous for %s@%s", member.Name, member.Version))
		}
		return member.Version, caret, nil
	default:
		return "", "", apperr.New(apperr.Resolve, "resolver.workspace", member.Name,
			fmt.Sprintf("unsupported workspace range %q", rng))
	}
}
