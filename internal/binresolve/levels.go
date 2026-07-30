package binresolve

import (
	"os"
	"path/filepath"
)

type importerLevel struct {
	PackageDir  string
	NodeModules string
	ImporterRel string
}

// importerLevels returns nearest-first node_modules levels from importer to project root.
func importerLevels(projectRoot, packageDir, importerRel string) []importerLevel {
	projectRoot = filepath.Clean(projectRoot)
	packageDir = filepath.Clean(packageDir)
	if projectRoot == "" || packageDir == "" {
		return nil
	}
	var levels []importerLevel
	dir := packageDir
	rel := importerRel
	for {
		nm := filepath.Join(dir, "node_modules")
		if st, err := os.Stat(nm); err == nil && st.IsDir() {
			levels = append(levels, importerLevel{
				PackageDir:  dir,
				NodeModules: nm,
				ImporterRel: rel,
			})
		}
		if dir == projectRoot {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		if rel != "." {
			rel = filepath.ToSlash(filepath.Join("..", rel))
			for len(rel) > 2 && rel[:3] == "../" {
				rel = rel[3:]
			}
			if rel == ".." {
				rel = "."
			}
		}
		dir = parent
	}
	return levels
}
