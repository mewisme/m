package isolated_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/linker/isolated"
	"github.com/mewisme/mew/internal/linker/planner"
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

func TestWorkspaceUpdatePlanWithPollutedLinkSource(t *testing.T) {
	pkgADir := t.TempDir()
	writePkgJSON(t, pkgADir, "pkg-a", "1.0.0")
	pollutedNM := filepath.Join(pkgADir, "node_modules")
	if err := os.MkdirAll(pollutedNM, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pollutedNM, "ms"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ms212 := t.TempDir()
	writePkgJSON(t, ms212, "ms", "2.1.2")
	ms213 := t.TempDir()
	writePkgJSON(t, ms213, "ms", "2.1.3")

	g := &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/pkg-a", Name: "pkg-a"},
		},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "ms", Version: "2.1.2"}},
			{ID: graph.PackageID{Name: "ms", Version: "2.1.3"}},
		},
		Edges: []graph.Edge{
			{From: ".", Name: "pkg-a", To: "link:packages/pkg-a", Kind: graph.DepProd},
			{From: ".", Name: "pkg-a", To: "pkg-a@1.0.0", Kind: graph.DepProd},
			{From: ".", Name: "ms", To: "ms@2.1.2", Kind: graph.DepProd},
			{From: "packages/pkg-a", Name: "ms", To: "ms@2.1.3", Kind: graph.DepProd},
			{From: "pkg-a@1.0.0", Name: "ms", To: "ms@2.1.3", Kind: graph.DepProd},
			{From: "link:packages/pkg-a", Name: "ms", To: "ms@2.1.3", Kind: graph.DepProd},
		},
	}

	pkgAVirtual := pkgADir // install uses live workspace path (pnpm may leave node_modules)

	nm := filepath.Join(t.TempDir(), "node_modules")
	extractDir := filepath.Join(t.TempDir(), "extract")
	stageNM := nm
	if err := os.MkdirAll(stageNM, 0o755); err != nil {
		t.Fatal(err)
	}
	caps, _ := planner.ProbeCached("", extractDir, stageNM)

	l := &isolated.Linker{
		NodeModules: nm,
		ExtractDirs: map[string]string{
			"pkg-a@1.0.0":         pkgAVirtual,
			"link:packages/pkg-a": pkgADir,
			"ms@2.1.2":            ms212,
			"ms@2.1.3":            ms213,
		},
		Capabilities: caps,
	}
	plan, err := l.Plan(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	for i, op := range plan.Ops {
		t.Logf("op[%d] kind=%s src=%s dest=%s", i, op.Kind, op.Src, op.Dest)
	}
	if err := l.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply: %v", err)
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
