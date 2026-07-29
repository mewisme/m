package resolver

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/manifest"
)

// LocalExtensionKey is the m.lock extensions key for workspace/file/link/portal sources.
const LocalExtensionKey = "mew.resolver/local"

// PatchExtensionKey is the extensions key for install-time patch application.
const PatchExtensionKey = "mew.resolver/patches"

// PatchSource records a patch file for a resolved package key.
type PatchSource struct {
	Path string `json:"path"`
	Hash string `json:"hash,omitempty"`
}

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

	protocol := string(item.protocol)
	targetDir, relPath, err := resolveLocalPath(s.proj.Root, item.declarerPath, item.rng)
	if err != nil {
		return err
	}

	_, version, err := readLocalPackage(targetDir, item.name, item.protocol)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.local", item.name, err)
	}

	id, key := s.packageKeyForInstance(item, version, nil)

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
	s.b.EdgeEx(item.from, item.display, key, item.kind, edgeRange, false)
	s.recordProvides(item.from, item.display, key)

	if _, ok := s.seenPkg[key]; ok {
		return nil
	}
	if _, ok := s.resolving[basePackageKey(item.name, version)]; ok {
		return nil
	}
	if len(s.seenPkg) >= maxPackages {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution package count exceeded %d", maxPackages))
	}
	s.resolving[basePackageKey(item.name, version)] = struct{}{}
	defer delete(s.resolving, basePackageKey(item.name, version))

	s.seenPkg[key] = struct{}{}
	s.b.Package(id, "", "")
	s.localSources[key] = LocalSource{Protocol: protocol, Path: relPath}
	s.pkgEnv[key] = append([]string(nil), item.envKeys...)
	s.pkgFrom[key] = item.from

	return s.expandLocalManifest(key, relPath, item.depth, item.path, item.envKeys)
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

func (s *resolveState) expandLocalManifest(fromKey, memberPath string, depth int, namePath, envKeys []string) error {
	doc, err := manifest.Load(filepath.Join(s.proj.Root, filepath.FromSlash(memberPath), "package.json"))
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.local", memberPath, err)
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return err
	}
	nextNamePath := append(append([]string(nil), namePath...), parsePackageKey(fromKey).Name)
	nextEnv := append(append([]string(nil), envKeys...), fromKey)
	return s.seedDeps(fromKey, memberPath, norm, depth, nextNamePath, nextEnv)
}

func (s *resolveState) buildExtensions() lockfile.Extensions {
	var ext lockfile.Extensions
	if len(s.localSources) > 0 {
		raw, err := json.Marshal(s.localSources)
		if err == nil {
			ext = lockfile.Extensions{LocalExtensionKey: raw}
		}
	}
	if s.patches != nil && len(s.patches.byPkgKey) > 0 {
		raw, err := json.Marshal(s.patches.byPkgKey)
		if err == nil {
			if ext == nil {
				ext = lockfile.Extensions{}
			}
			ext[PatchExtensionKey] = raw
		}
	}
	return ext
}

// DecodePatchSources parses the mew.resolver/patches extension payload.
func DecodePatchSources(ext lockfile.Extensions) (map[string]PatchSource, error) {
	if len(ext) == 0 {
		return nil, nil
	}
	raw, ok := ext[PatchExtensionKey]
	if !ok {
		return nil, nil
	}
	var patches map[string]PatchSource
	if err := json.Unmarshal(raw, &patches); err != nil {
		return nil, err
	}
	return patches, nil
}

// HasPatchSources reports whether extensions contain patch apply metadata.
func HasPatchSources(ext lockfile.Extensions) bool {
	patches, err := DecodePatchSources(ext)
	return err == nil && len(patches) > 0
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
	if err := ValidateLocalPaths(locals); err != nil {
		return nil, err
	}
	return locals, nil
}

// ValidateLocalPaths rejects untrusted relative paths that escape a project root.
func ValidateLocalPaths(locals map[string]LocalSource) error {
	for key, src := range locals {
		p := strings.TrimSpace(src.Path)
		if p == "" {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(p))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return apperr.New(apperr.Lockfile, "resolver.local", key, "local path escapes project root")
		}
		if filepath.IsAbs(clean) {
			return apperr.New(apperr.Lockfile, "resolver.local", key, "absolute local path not allowed")
		}
	}
	return nil
}
