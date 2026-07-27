package lifecycle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lifecycle"
	"github.com/mewisme/m/internal/linker"
)

func writePkg(t *testing.T, dir, name, scriptsJSON string) {
	t.Helper()
	body := `{"name":"` + name + `","version":"1.0.0"`
	if scriptsJSON != "" {
		body += `,"scripts":` + scriptsJSON
	}
	body += "}"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverSkipsMissingScripts(t *testing.T) {
	root := t.TempDir()
	aDir := filepath.Join(root, "node_modules", "pkg-a")
	writePkg(t, aDir, "pkg-a", "")
	g := &graph.Graph{
		Packages: []graph.Package{{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}}},
	}
	plan := &linker.Plan{
		Placements: []linker.Placement{{Key: "pkg-a@1.0.0", DestDir: aDir}},
	}
	got, err := lifecycle.Discover(g, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scripts) != 0 {
		t.Fatalf("want no scripts, got %+v", got.Scripts)
	}
}

func TestDiscoverTopoOrder(t *testing.T) {
	root := t.TempDir()
	aDir := filepath.Join(root, "a")
	bDir := filepath.Join(root, "b")
	writePkg(t, aDir, "pkg-a", `{"postinstall":"echo a"}`)
	writePkg(t, bDir, "pkg-b", `{"postinstall":"echo b"}`)
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "pkg-b", Version: "1.0.0"}},
		},
		Edges: []graph.Edge{{From: "pkg-a@1.0.0", To: "pkg-b@1.0.0", Kind: graph.DepProd}},
	}
	plan := &linker.Plan{
		Placements: []linker.Placement{
			{Key: "pkg-a@1.0.0", DestDir: aDir},
			{Key: "pkg-b@1.0.0", DestDir: bDir},
		},
	}
	got, err := lifecycle.Discover(g, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scripts) != 2 {
		t.Fatalf("want 2 scripts, got %d", len(got.Scripts))
	}
	if got.Scripts[0].PackageName != "pkg-b" || got.Scripts[1].PackageName != "pkg-a" {
		t.Fatalf("dependency first: got %+v", got.Scripts)
	}
}

func TestEnabledEnvGate(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_LIFECYCLE", "1")
	if !lifecycle.Enabled(nil) {
		t.Fatal("expected enabled with env gate")
	}
}
