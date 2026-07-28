package isolated_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/linker/isolated"
)

func TestLayoutTransitive(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	extractA := t.TempDir()
	extractB := t.TempDir()
	writePkgJSON(t, extractA, "pkg-a", "1.0.0")
	writePkgJSON(t, extractB, "pkg-b", "1.2.0")
	g := graphFromKeys(t,
		[]string{"pkg-a@1.0.0", "pkg-b@1.2.0"},
		[]graph.Edge{
			{From: ".", To: "pkg-a@1.0.0", Kind: graph.DepProd},
			{From: "pkg-a@1.0.0", To: "pkg-b@1.2.0", Kind: graph.DepProd},
		},
	)
	l := &isolated.Linker{
		NodeModules: nm,
		ExtractDirs: map[string]string{
			"pkg-a@1.0.0": extractA,
			"pkg-b@1.2.0": extractB,
		},
	}
	plan, err := l.Plan(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	if plan.LayoutMode != "isolated" {
		t.Fatalf("layout mode %q", plan.LayoutMode)
	}
	alias := filepath.Join(nm, "pkg-a")
	if !hasDest(plan.Ops, alias) {
		t.Fatalf("missing root alias %s in %d ops", alias, len(plan.Ops))
	}
	virtual := filepath.Join(nm, ".pnpm", "pkg-b@1.2.0", "node_modules", "pkg-b")
	if !hasDest(plan.Ops, virtual) {
		t.Fatalf("missing virtual pkg-b at %s", virtual)
	}
}

func graphFromKeys(t *testing.T, keys []string, edges []graph.Edge) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder().Importer(graph.RootImporter, "root")
	for _, key := range keys {
		name, ver := splitKey(key)
		b = b.Package(graph.PackageID{Name: name, Version: ver}, "", "")
	}
	for _, e := range edges {
		b = b.Edge(e.From, e.To, e.Kind, "")
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func splitKey(key string) (string, string) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '@' && i > 0 {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}

func hasDest(ops []linker.Op, dest string) bool {
	dest = filepath.Clean(dest)
	for _, op := range ops {
		if filepath.Clean(op.Dest) == dest {
			return true
		}
	}
	return false
}

func writePkgJSON(t *testing.T, dir, name, version string) {
	t.Helper()
	body := []byte(`{"name":"` + name + `","version":"` + version + `"}`)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
}
