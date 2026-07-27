package resolver

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile"
	"github.com/mewisme/m/internal/manifest"
)

// LocalExtensionKey is the m.lock extensions key for workspace/file/link/portal sources.
const LocalExtensionKey = "mew.resolver/local"

// LocalSource records a non-registry package location resolved into the lock graph.
type LocalSource struct {
	Protocol string `json:"protocol"`
	Path     string `json:"path"` // root-relative POSIX path
}

func (s *resolveState) processLocal(item workItem) error {
	if item.depth > maxDepth {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution depth exceeded %d", maxDepth))
	}
	if pathContains(item.path, item.name) {
		return cycleError(item.path, item.name)
	}

	protocol := string(item.protocol)
	targetDir, relPath, err := resolveLocalPath(s.proj.Root, item.declarerPath, item.rng)
	if err != nil {
		return err
	}

	name, version, err := readLocalPackage(targetDir, item.name, item.protocol)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.local", item.name, err)
	}

	id := graph.PackageID{Name: name, Version: version}
	id.Normalize()
	key := id.Key()

	decision := ResolutionDecision{
		Package:   item.name,
		Requested: protocol + ":" + item.rng,
		Selected:  version,
		Reason:    protocol,
	}
	s.decisions = append(s.decisions, decision)

	edgeRange := item.spec
	if edgeRange == "" {
		edgeRange = protocol + ":" + item.rng
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
	s.localSources[key] = LocalSource{Protocol: protocol, Path: relPath}

	return s.expandLocalManifest(key, relPath, item.depth, item.path)
}

func resolveLocalPath(root, declarerPath, rel string) (absDir, relToRoot string, err error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", "", apperr.New(apperr.Resolve, "resolver.local", rel, "empty local path")
	}
	base := root
	if declarerPath != "" && declarerPath != "." {
		base = filepath.Join(root, filepath.FromSlash(declarerPath))
	}
	var target string
	if filepath.IsAbs(rel) {
		target = filepath.Clean(rel)
	} else {
		target = filepath.Clean(filepath.Join(base, filepath.FromSlash(rel)))
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", apperr.Wrap(apperr.IO, "resolver.local", root, err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", "", apperr.Wrap(apperr.IO, "resolver.local", rel, err)
	}
	relRoot, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", "", apperr.Wrap(apperr.Resolve, "resolver.local", rel, err)
	}
	if relRoot == ".." || strings.HasPrefix(relRoot, ".."+string(filepath.Separator)) {
		return "", "", apperr.New(apperr.Resolve, "resolver.local", rel, "local path escapes project root")
	}
	return targetAbs, filepath.ToSlash(relRoot), nil
}

func readLocalPackage(targetDir, fallbackName string, protocol manifest.Protocol) (name, version string, err error) {
	doc, err := manifest.Load(filepath.Join(targetDir, "package.json"))
	if err != nil {
		if protocol == manifest.ProtocolLink {
			return fallbackName, "0.0.0", nil
		}
		return "", "", err
	}
	name = doc.Name
	if name == "" {
		name = fallbackName
	}
	version = doc.Version
	if version == "" {
		version = "0.0.0"
	}
	return name, version, nil
}

func (s *resolveState) expandLocalManifest(fromKey, memberPath string, depth int, path []string) error {
	doc, err := manifest.Load(filepath.Join(s.proj.Root, filepath.FromSlash(memberPath), "package.json"))
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.local", memberPath, err)
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return err
	}
	nextPath := append(append([]string(nil), path...), parsePackageKey(fromKey).Name)
	return s.seedDeps(fromKey, memberPath, norm, depth, nextPath)
}

func (s *resolveState) buildExtensions() lockfile.Extensions {
	if len(s.localSources) == 0 {
		return nil
	}
	raw, err := json.Marshal(s.localSources)
	if err != nil {
		return nil
	}
	return lockfile.Extensions{LocalExtensionKey: raw}
}

// HasLocalSources reports whether extensions contain resolved local/workspace packages.
func HasLocalSources(ext lockfile.Extensions) bool {
	if len(ext) == 0 {
		return false
	}
	raw, ok := ext[LocalExtensionKey]
	if !ok || len(raw) == 0 {
		return false
	}
	var locals map[string]LocalSource
	return json.Unmarshal(raw, &locals) == nil && len(locals) > 0
}

// DecodeLocalSources parses the mew.resolver/local extension payload.
func DecodeLocalSources(ext lockfile.Extensions) (map[string]LocalSource, error) {
	if len(ext) == 0 {
		return nil, nil
	}
	raw, ok := ext[LocalExtensionKey]
	if !ok {
		return nil, nil
	}
	var locals map[string]LocalSource
	if err := json.Unmarshal(raw, &locals); err != nil {
		return nil, err
	}
	return locals, nil
}
