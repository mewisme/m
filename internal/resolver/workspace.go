package resolver

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/semver"
	"github.com/mewisme/m/internal/workspace"
)

type workspaceMember struct {
	Name    string
	Version string
	Path    string // root-relative POSIX
}

func (s *resolveState) initWorkspace() error {
	patterns, err := s.proj.Doc.WorkspacePatterns()
	if err != nil {
		return err
	}
	if len(patterns) == 0 {
		return nil
	}
	idx, err := workspace.BuildIndex(s.proj.Root)
	if err != nil {
		return err
	}
	s.wsIndex = idx
	s.wsByName = make(map[string]workspaceMember, len(idx.Members))
	s.wsMemberPaths = make(map[string]struct{}, len(idx.Members))
	for _, memPath := range idx.Members {
		doc, err := manifest.Load(filepath.Join(s.proj.Root, filepath.FromSlash(memPath), "package.json"))
		if err != nil {
			return apperr.Wrap(apperr.Manifest, "resolver.workspace", memPath, err)
		}
		name := doc.Name
		if name == "" {
			return apperr.New(apperr.Manifest, "resolver.workspace", memPath, "workspace member missing name")
		}
		if prev, ok := s.wsByName[name]; ok {
			return apperr.New(apperr.Resolve, "resolver.workspace", name,
				fmt.Sprintf("ambiguous workspace target %q: members %q and %q", name, prev.Path, memPath))
		}
		ver := doc.Version
		if ver == "" {
			ver = "0.0.0"
		}
		s.wsByName[name] = workspaceMember{Name: name, Version: ver, Path: memPath}
		s.wsMemberPaths[memPath] = struct{}{}
	}
	return nil
}

func (s *resolveState) seedWorkspaceMembers() error {
	if s.wsIndex == nil {
		return nil
	}
	paths := append([]string(nil), s.wsIndex.Members...)
	sort.Strings(paths)
	for _, memPath := range paths {
		if memPath == "." || memPath == string(graph.RootImporter) {
			continue
		}
		doc, err := manifest.Load(filepath.Join(s.proj.Root, filepath.FromSlash(memPath), "package.json"))
		if err != nil {
			return err
		}
		if !memberHasDeps(doc) {
			continue
		}
		norm, err := manifest.ToNormalized(doc)
		if err != nil {
			return err
		}
		importerID := graph.ImporterID(memPath)
		if !s.seededImporters[importerID] {
			s.b.Importer(importerID, norm.Name)
			s.seededImporters[importerID] = true
		}
		if err := s.seedDeps(string(importerID), memPath, norm, 1, nil); err != nil {
			return err
		}
	}
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
	if pathContains(item.path, item.name) {
		return cycleError(item.path, item.name)
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

	id := graph.PackageID{Name: member.Name, Version: version}
	id.Normalize()
	key := id.Key()

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
	s.b.EdgeEx(item.from, key, item.kind, edgeRange, false)

	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	if len(s.seenPkg) >= maxPackages {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution package count exceeded %d", maxPackages))
	}
	s.seenPkg[key] = struct{}{}
	s.b.Package(id, "", "")
	s.localSources[key] = LocalSource{Protocol: "workspace", Path: member.Path}

	return s.expandLocalManifest(key, member.Path, item.depth, item.path)
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
