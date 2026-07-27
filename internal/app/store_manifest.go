package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
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

func writeStoreManifest(projectRoot string, g *graph.Graph) error {
	if g == nil {
		return nil
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
		return apperr.Wrap(apperr.IO, "app.store-manifest", projectRoot, err)
	}
	raw = append(raw, '\n')
	dir := filepath.Dir(storeManifestPath(projectRoot))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.store-manifest", dir, err)
	}
	path := storeManifestPath(projectRoot)
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

// CollectReferencedIntegrities scans manifest files under roots for prune.
func CollectReferencedIntegrities(roots []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return err
			}
			if filepath.Base(path) != "store-manifest.json" {
				return nil
			}
			doc, err := ReadStoreManifest(filepath.Dir(filepath.Dir(path)))
			if err != nil {
				return nil
			}
			for _, k := range doc.Packages {
				out[k] = struct{}{}
			}
			return nil
		})
	}
	return out, nil
}
