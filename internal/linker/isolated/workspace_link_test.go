package isolated_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/linker/isolated"
)

func TestWorkspaceDualRootEdgesApplyWithCopyAliases(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	pkgADir := t.TempDir()
	writePkgJSON(t, pkgADir, "pkg-a", "1.0.0")
	msDir := t.TempDir()
	writePkgJSON(t, msDir, "ms", "2.1.3")

	g := &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/pkg-a", Name: "pkg-a"},
		},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "ms", Version: "2.1.3"}},
		},
		Edges: []graph.Edge{
			{From: ".", Name: "pkg-a", To: "pkg-a@1.0.0", Kind: graph.DepProd},
			{From: ".", Name: "pkg-a", To: "link:packages/pkg-a", Kind: graph.DepProd},
			{From: ".", Name: "ms", To: "ms@2.1.3", Kind: graph.DepProd},
			{From: "packages/pkg-a", Name: "ms", To: "ms@2.1.3", Kind: graph.DepProd},
		},
	}

	extracts := map[string]string{
		"pkg-a@1.0.0":         pkgADir,
		"ms@2.1.3":            msDir,
		"link:packages/pkg-a": pkgADir,
	}
	l := &isolated.Linker{
		NodeModules: nm,
		ExtractDirs: extracts,
	}
	plan, err := l.Plan(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	aliasDest := filepath.Join(nm, "pkg-a")
	if aliasOpCount(plan.Ops, aliasDest) > 1 {
		t.Fatalf("duplicate ops for %s: %d", aliasDest, aliasOpCount(plan.Ops, aliasDest))
	}
	if err := l.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
}

func aliasOpCount(ops []linker.Op, dest string) int {
	dest = filepath.Clean(dest)
	n := 0
	for _, op := range ops {
		if filepath.Clean(op.Dest) == dest {
			n++
		}
	}
	return n
}
