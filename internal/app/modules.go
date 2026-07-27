package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker/isolated"
)

const modulesFileName = "modules.v1.json"

// ModulesDocument is node_modules/.mew/modules.v1.json for isolated layouts.
type ModulesDocument struct {
	SchemaVersion int            `json:"schemaVersion"`
	Linker        string         `json:"linker"`
	Packages      []ModulesEntry `json:"packages"`
}

// ModulesEntry maps one resolved package to its virtual store id.
type ModulesEntry struct {
	Key       string `json:"key"`
	StoreID   string `json:"storeID"`
	Integrity string `json:"integrity,omitempty"`
}

func writeModulesMetadata(nmRoot string, g *graph.Graph) error {
	if g == nil {
		return nil
	}
	doc := ModulesDocument{SchemaVersion: 1, Linker: "isolated"}
	byKey := map[string]string{}
	for _, p := range g.Packages {
		byKey[p.ID.Key()] = p.Integrity
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		doc.Packages = append(doc.Packages, ModulesEntry{
			Key: key, StoreID: isolated.StoreIDFromKey(key), Integrity: byKey[key],
		})
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return apperr.Wrap(apperr.Internal, "app.modules", nmRoot, err)
	}
	dir := filepath.Join(nmRoot, ".mew")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "app.modules", dir, err)
	}
	path := filepath.Join(dir, modulesFileName)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
