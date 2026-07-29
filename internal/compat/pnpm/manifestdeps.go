package pnpm

import (
	"path/filepath"

	"github.com/mewisme/mew/internal/manifest"
)

func manifestImporterDepNames(lockRoot, importerID string) (map[string]struct{}, error) {
	if lockRoot == "" {
		return nil, nil
	}
	pkgPath := filepath.Join(lockRoot, "package.json")
	if importerID != "" && importerID != "." {
		pkgPath = filepath.Join(lockRoot, filepath.FromSlash(importerID), "package.json")
	}
	doc, err := manifest.Load(pkgPath)
	if err != nil {
		return nil, err
	}
	norm, err := manifest.ToNormalized(doc)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{})
	for _, d := range norm.Dependencies {
		names[d.Name] = struct{}{}
	}
	return names, nil
}
