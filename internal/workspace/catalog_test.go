package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/workspace"
)

func TestLoadCatalogMergeYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "name":"root",
  "catalog": {"react":"^17.0.0"}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	yaml := `packages:
  - "packages/*"
catalog:
  lodash: 4.17.21
`
	if err := os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cat, err := workspace.LoadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cat["react"] != "^17.0.0" || cat["lodash"] != "4.17.21" {
		t.Fatalf("catalog=%v", cat)
	}
}
