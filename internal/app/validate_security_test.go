package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
)

func TestValidateDetectsSymlinkIdentitySwap(t *testing.T) {
	stage := t.TempDir()
	nm := filepath.Join(stage, "node_modules")
	pkgDir := filepath.Join(nm, "pkg-a")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := `{"name":"pkg-a","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
		Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	plan := &linker.Plan{
		Placements: []linker.Placement{{
			Key:     "pkg-a@1.0.0",
			DestDir: pkgDir,
		}},
	}

	if err := validateStaged(stage, plan, g, "hoisted"); err != nil {
		t.Fatalf("initial validate: %v", err)
	}

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "package.json"), []byte(`{"name":"evil","version":"9.9.9"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(pkgDir, "package.json"))
	if err := os.Symlink(filepath.Join(outside, "package.json"), filepath.Join(pkgDir, "package.json")); err != nil {
		t.Skip("symlink not supported:", err)
	}

	err = validateStaged(stage, plan, g, "hoisted")
	if err == nil {
		t.Fatal("expected identity mismatch after symlink swap")
	}
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}
