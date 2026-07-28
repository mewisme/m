package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

// LoadCatalog loads the merged default catalog from root package.json and optional pnpm-workspace.yaml.
func LoadCatalog(root string) (manifest.Catalog, error) {
	doc, err := manifest.LoadCached(root)
	if err != nil {
		return nil, err
	}
	cat, err := manifest.ParseCatalog(doc.Catalog)
	if err != nil {
		return nil, err
	}
	yamlPath := filepath.Join(root, "pnpm-workspace.yaml")
	if overlay, err := parsePNPMWorkspaceCatalog(yamlPath); err != nil {
		return nil, err
	} else if len(overlay) > 0 {
		if cat == nil {
			cat = manifest.Catalog{}
		}
		for k, v := range overlay {
			cat[k] = v
		}
	}
	return cat, nil
}

// parsePNPMWorkspaceCatalog reads catalog entries from pnpm-workspace.yaml when present.
func parsePNPMWorkspaceCatalog(path string) (manifest.Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "workspace.catalog", path, err)
	}
	return parseYAMLCatalogBlock(string(data))
}

func parseYAMLCatalogBlock(text string) (manifest.Catalog, error) {
	lines := strings.Split(text, "\n")
	inCatalog := false
	catIndent := -1
	out := manifest.Catalog{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := leadingSpaces(line)
		trim := strings.TrimSpace(line)
		if !inCatalog {
			if trim == "catalog:" || strings.HasPrefix(trim, "catalog:") {
				inCatalog = true
				catIndent = indent
				if idx := strings.Index(trim, ":"); idx >= 0 && len(trim) > idx+1 {
					val := strings.TrimSpace(trim[idx+1:])
					if val != "" && val != "{" {
						return nil, apperr.New(apperr.Manifest, "workspace.catalog", "catalog", "inline catalog not supported")
					}
				}
			}
			continue
		}
		if indent <= catIndent && trim != "" {
			break
		}
		if idx := strings.Index(trim, ":"); idx > 0 {
			key := strings.TrimSpace(trim[:idx])
			val := strings.TrimSpace(trim[idx+1:])
			val = strings.Trim(val, `"'`)
			if key != "" && val != "" {
				out[key] = val
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}
