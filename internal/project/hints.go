package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LockHints carries package.json identity fields for incumbent lock detection.
type LockHints struct {
	PackageManager string
	DevEnginesPM   string
}

// LockfileHints returns package.json identity fields used for incumbent lock detection.
func LockfileHints(p *Project) LockHints {
	if p == nil {
		return LockHints{}
	}
	hints := LockHints{}
	if p.Doc != nil && p.Doc.PackageManager != "" {
		hints.PackageManager = p.Doc.PackageManager
	}
	pkgPath := filepath.Join(p.Root, "package.json")
	if p.Rel != "" && p.Rel != "." {
		pkgPath = filepath.Join(p.Root, filepath.FromSlash(p.Rel), "package.json")
	}
	if hints.DevEnginesPM == "" {
		if raw, err := readDevEnginesPM(pkgPath); err == nil {
			hints.DevEnginesPM = raw
		}
	}
	return hints
}

func readDevEnginesPM(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var pkg struct {
		DevEngines struct {
			PackageManager string `json:"packageManager"`
		} `json:"devEngines"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return "", err
	}
	return pkg.DevEngines.PackageManager, nil
}
