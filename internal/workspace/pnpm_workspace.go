package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// PNPMWorkspacePatterns reads packages globs from pnpm-workspace.yaml when present.
func PNPMWorkspacePatterns(root string) ([]string, error) {
	path := filepath.Join(root, "pnpm-workspace.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.Packages))
	for _, p := range doc.Packages {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}
