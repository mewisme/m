package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/store"
)

const storeManifestSchemaVersion = 1

// StoreManifest lists integrity keys for packages linked in the current install.
type StoreManifest struct {
	SchemaVersion int      `json:"schemaVersion"`
	Packages      []string `json:"packages"`
}

func storeManifestPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".mew", "store-manifest.json")
}

func writeStagedStoreManifest(stageDir string, g *graph.Graph) error {
	return writeStoreManifestAt(filepath.Join(stageDir, ".mew", "store-manifest.json"), g)
}

func writeStoreManifestAt(path string, g *graph.Graph) error {
	if g == nil {
		return nil
	}
	if err := guardStoreManifestPath(path); err != nil {
		return err
	}
	keys := make([]string, 0, len(g.Packages))
	seen := map[string]struct{}{}
	for _, pkg := range g.Packages {
		if pkg.Integrity == "" {
			continue
		}
		if _, ok := seen[pkg.Integrity]; ok {
			continue
		}
		seen[pkg.Integrity] = struct{}{}
		keys = append(keys, pkg.Integrity)
	}
	sort.Strings(keys)
	doc := StoreManifest{SchemaVersion: storeManifestSchemaVersion, Packages: keys}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", path, err)
	}
	raw = append(raw, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", dir, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", path, err)
	}
	return nil
}

// ReadStoreManifest loads .mew/store-manifest.json when present.
func ReadStoreManifest(projectRoot string) (*StoreManifest, error) {
	path := storeManifestPath(projectRoot)
	if err := guardStoreManifestPath(path); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &StoreManifest{SchemaVersion: storeManifestSchemaVersion, Packages: []string{}}, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.store-manifest", path, err)
	}
	var doc StoreManifest
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, apperr.Wrap(apperr.Store, "app.store-manifest", path, err)
	}
	if doc.Packages == nil {
		doc.Packages = []string{}
	}
	return &doc, nil
}

// CollectReferencedIntegrities scans manifest files and active txn journals under roots.
func CollectReferencedIntegrities(roots []string) (map[string]struct{}, error) {
	return store.CollectReferencedIntegrities(roots)
}

func guardStoreManifestPath(path string) error {
	dir := filepath.Dir(path)
	projectRoot := filepath.Dir(dir)
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", projectRoot, err)
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", path, err)
	}
	return fsx.GuardAncestors(absRoot, target)
}
