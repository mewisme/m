package binmeta

import "path/filepath"

const fileName = "bins.v1.json"

// Dir returns node_modules/.mew for the given node_modules root.
func Dir(nodeModules string) string {
	return filepath.Join(nodeModules, ".mew")
}

// Path returns the bins metadata file path for a node_modules root.
func Path(nodeModules string) string {
	return filepath.Join(Dir(nodeModules), fileName)
}
