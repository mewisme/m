package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PatchedDependency maps a package selector (name@version) to a patch file path.
type PatchedDependency struct {
	Selector string
	Path     string
}

// PatchedDependencies reads pnpm.patchedDependencies from package.json source bytes.
func PatchedDependencies(doc *Document) ([]PatchedDependency, error) {
	if doc == nil || len(doc.Source) == 0 {
		return nil, nil
	}
	var root struct {
		Pnpm struct {
			PatchedDependencies map[string]string `json:"patchedDependencies"`
		} `json:"pnpm"`
	}
	if err := json.Unmarshal(doc.Source, &root); err != nil {
		return nil, fmt.Errorf("decode pnpm.patchedDependencies: %w", err)
	}
	if len(root.Pnpm.PatchedDependencies) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(root.Pnpm.PatchedDependencies))
	for k := range root.Pnpm.PatchedDependencies {
		names = append(names, k)
	}
	sortStrings(names)
	out := make([]PatchedDependency, 0, len(names))
	for _, sel := range names {
		out = append(out, PatchedDependency{Selector: sel, Path: root.Pnpm.PatchedDependencies[sel]})
	}
	return out, nil
}

func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}

// ResolvePatchPath resolves a patch file path relative to projectRoot.
func ResolvePatchPath(projectRoot, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty patch path")
	}
	if filepath.IsAbs(rel) {
		return rel, nil
	}
	return filepath.Join(projectRoot, filepath.FromSlash(rel)), nil
}

// SetPatchedDependency records selector→patchPath under pnpm.patchedDependencies.
func (d *Document) SetPatchedDependency(selector, patchPath string) error {
	if d == nil {
		return fmt.Errorf("nil document")
	}
	selector = strings.TrimSpace(selector)
	patchPath = strings.TrimSpace(patchPath)
	if selector == "" || patchPath == "" {
		return fmt.Errorf("empty patch selector or path")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(d.Source, &root); err != nil {
		return fmt.Errorf("decode package.json: %w", err)
	}
	pnpm := map[string]any{}
	if raw, ok := root["pnpm"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &pnpm); err != nil {
			return fmt.Errorf("decode pnpm: %w", err)
		}
	}
	patches := map[string]string{}
	if existing, ok := pnpm["patchedDependencies"]; ok {
		switch m := existing.(type) {
		case map[string]string:
			for k, v := range m {
				patches[k] = v
			}
		case map[string]any:
			for k, v := range m {
				patches[k] = fmt.Sprint(v)
			}
		}
	}
	patches[selector] = patchPath
	pnpm["patchedDependencies"] = patches
	raw, err := json.Marshal(pnpm)
	if err != nil {
		return fmt.Errorf("encode pnpm: %w", err)
	}
	return d.spliceTopLevel("pnpm", raw)
}

// ReadPatchFile reads and normalizes a unified diff patch (CRLF→LF).
func ReadPatchFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n")), nil
}
