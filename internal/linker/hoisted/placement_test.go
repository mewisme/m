package hoisted_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/hoisted"
)

func testGraph(t *testing.T, build func(*graph.Builder) *graph.Builder) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder().Importer(graph.RootImporter, "root")
	g, err := build(b).Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func placementMap(t *testing.T, g *graph.Graph, nmRoot string) map[string]string {
	t.Helper()
	ps, err := hoisted.Placements(g, nmRoot)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string, len(ps))
	for _, p := range ps {
		out[p.Key] = p.DestDir
	}
	return out
}

func TestPlacementsHoistsRootDeps(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "", "").
			Edge(string(graph.RootImporter), "lodash@4.17.21", graph.DepProd, "^4.17.21")
	})
	got := placementMap(t, g, nm)
	want := filepath.Join(nm, "lodash")
	if got["lodash@4.17.21"] != want {
		t.Fatalf("got %q want %q", got["lodash@4.17.21"], want)
	}
}

func TestPlacementsScopedPackage(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "@scope/pkg", Version: "1.0.0"}, "", "").
			Edge(string(graph.RootImporter), "@scope/pkg@1.0.0", graph.DepProd, "^1.0.0")
	})
	got := placementMap(t, g, nm)
	want := filepath.Join(nm, "@scope", "pkg")
	if got["@scope/pkg@1.0.0"] != want {
		t.Fatalf("got %q want %q", got["@scope/pkg@1.0.0"], want)
	}
}

func TestPlacementsTransitiveHoist(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
			Package(graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, "", "").
			Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
			Edge("pkg-a@1.0.0", "pkg-b@1.2.0", graph.DepProd, "^1.0.0")
	})
	got := placementMap(t, g, nm)
	if got["pkg-a@1.0.0"] != filepath.Join(nm, "pkg-a") {
		t.Fatalf("pkg-a: %q", got["pkg-a@1.0.0"])
	}
	if got["pkg-b@1.2.0"] != filepath.Join(nm, "pkg-b") {
		t.Fatalf("pkg-b should hoist to root, got %q", got["pkg-b@1.2.0"])
	}
}

func TestPlacementsVersionCollisionNests(t *testing.T) {
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "", "").
			Package(graph.PackageID{Name: "lodash", Version: "4.17.20"}, "", "").
			Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
			Edge(string(graph.RootImporter), "lodash@4.17.21", graph.DepProd, "^4.17.21").
			Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
			Edge("pkg-a@1.0.0", "lodash@4.17.20", graph.DepProd, "^4.17.20")
	})
	got := placementMap(t, g, nm)
	if got["lodash@4.17.21"] != filepath.Join(nm, "lodash") {
		t.Fatalf("root lodash: %q", got["lodash@4.17.21"])
	}
	wantNested := filepath.Join(nm, "pkg-a", "node_modules", "lodash")
	if got["lodash@4.17.20"] != wantNested {
		t.Fatalf("nested lodash: got %q want %q", got["lodash@4.17.20"], wantNested)
	}
}

func TestPlanApplyAndBins(t *testing.T) {
	stage := t.TempDir()
	nm := filepath.Join(stage, "node_modules")

	pkgDir := filepath.Join(stage, "extract", "pkg-cli")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"pkg-cli","version":"1.0.0","bin":{"cli":"./cli.js"}}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "pkg-cli", Version: "1.0.0"}, "", "").
			Edge(string(graph.RootImporter), "pkg-cli@1.0.0", graph.DepProd, "^1.0.0")
	})

	l := &hoisted.Linker{
		NodeModules: nm,
		ExtractDirs: map[string]string{"pkg-cli@1.0.0": pkgDir},
	}
	plan, err := l.Plan(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Ops) != 2 {
		t.Fatalf("ops=%d want 2", len(plan.Ops))
	}
	if err := l.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(nm, "pkg-cli", "cli.js")); err != nil {
		t.Fatalf("installed package: %v", err)
	}

	binName := "cli"
	if runtime.GOOS == "windows" {
		binName = "cli.cmd"
	}
	binPath := filepath.Join(nm, ".bin", binName)
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("bin shim: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "node") {
		t.Fatalf("bin shim missing node invocation: %q", body)
	}
	if !strings.Contains(body, "cli.js") {
		t.Fatalf("bin shim missing target script: %q", body)
	}
}

func TestFakeLinkerStructuredPlan(t *testing.T) {
	plan := &linker.Plan{
		NodeModules: filepath.Join(t.TempDir(), "node_modules"),
		Ops: []linker.Op{
			{Kind: linker.OpMkdir, Dest: filepath.Join(t.TempDir(), "node_modules", "a")},
		},
	}
	if plan.Ops[0].Kind != linker.OpMkdir {
		t.Fatalf("unexpected op kind %q", plan.Ops[0].Kind)
	}
}
