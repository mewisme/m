package resolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
)

func (s *resolveState) processTarball(item workItem) error {
	if item.depth > maxDepth {
		return apperr.New(apperr.Resolve, "resolver.limit", item.name,
			fmt.Sprintf("resolution depth exceeded %d", maxDepth))
	}
	absPath, relPath, err := resolveTarballPath(s.proj.Root, item.declarerPath, item.rng)
	if err != nil {
		return err
	}
	_, version, err := readTarballPackage(absPath, item.name)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.tarball", item.name, err)
	}
	id, key := s.packageKeyForInstance(item, version, nil)

	decision := ResolutionDecision{
		Package:   item.name,
		Requested: "tarball:" + item.rng,
		Selected:  version,
		Reason:    "tarball",
	}
	s.decisions = append(s.decisions, decision)

	edgeRange := item.spec
	if edgeRange == "" {
		edgeRange = "tarball:" + item.rng
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
	s.localSources[key] = LocalSource{Protocol: "tarball", Path: relPath}
	s.pkgEnv[key] = append([]string(nil), item.envKeys...)
	s.pkgFrom[key] = item.from

	stageDir, err := extractTarballPeek(s.ctx, absPath)
	if err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.tarball", item.name, err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()
	return s.expandLocalManifestAbs(key, stageDir, item.depth, item.path, item.envKeys)
}

func resolveTarballPath(root, declarerPath, rel string) (absFile, relToRoot string, err error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", "", apperr.New(apperr.Resolve, "resolver.tarball", rel, "empty tarball path")
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
		return "", "", apperr.Wrap(apperr.IO, "resolver.tarball", root, err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", "", apperr.Wrap(apperr.IO, "resolver.tarball", rel, err)
	}
	if err := fsx.GuardAncestors(rootAbs, targetAbs); err != nil {
		return "", "", apperr.Wrap(apperr.Resolve, "resolver.tarball", rel, err)
	}
	relRoot, err := relPathToRoot(rootAbs, targetAbs)
	if err != nil {
		return "", "", apperr.Wrap(apperr.Resolve, "resolver.tarball", rel, err)
	}
	if st, err := os.Stat(targetAbs); err != nil {
		return "", "", apperr.Wrap(apperr.Resolve, "resolver.tarball", rel, err)
	} else if st.IsDir() {
		return "", "", apperr.New(apperr.Resolve, "resolver.tarball", rel, "tarball path is a directory")
	}
	return targetAbs, relRoot, nil
}

func relPathToRoot(rootAbs, targetAbs string) (string, error) {
	relRoot, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if relRoot == ".." || strings.HasPrefix(relRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root")
	}
	return filepath.ToSlash(relRoot), nil
}
