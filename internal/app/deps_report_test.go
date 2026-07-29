package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

func testDepsGraph(t *testing.T) *graph.Graph {
	g, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "int-b", "").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, "int-c", "").
		Package(graph.PackageID{Name: "pkg-d", Version: "2.0.0"}, "int-d", "").
		Edge(".", "pkg-b@1.0.0", graph.DepProd, "^1.0.0").
		Edge("pkg-b@1.0.0", "pkg-c@1.0.0", graph.DepProd, "^1.0.0").
		Edge(".", "pkg-d@2.0.0", graph.DepDev, "^2.0.0").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestBuildDepTreeProdAndDepth(t *testing.T) {
	g := testDepsGraph(t)
	tree, err := BuildDepTree(g, DepTreeOptions{ImporterID: graph.RootImporter, ProdOnly: true, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Dependencies) != 1 {
		t.Fatalf("prod deps=%d", len(tree.Dependencies))
	}
	if tree.Dependencies[0].Name != "pkg-b" || tree.Dependencies[0].Version != "1.0.0" {
		t.Fatalf("%v", tree.Dependencies[0])
	}
	if len(tree.Dependencies[0].Dependencies) != 0 {
		t.Fatalf("depth=1 should not include grandchildren: %v", tree.Dependencies[0].Dependencies)
	}

	full, err := BuildDepTree(g, DepTreeOptions{ImporterID: graph.RootImporter, Depth: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Dependencies) != 2 {
		t.Fatalf("all deps=%d", len(full.Dependencies))
	}
	if len(full.Dependencies[0].Dependencies) != 1 {
		t.Fatalf("pkg-b child=%v", full.Dependencies[0].Dependencies)
	}
}

func TestFormatDepTreeText(t *testing.T) {
	g := testDepsGraph(t)
	tree, err := BuildDepTree(g, DepTreeOptions{ImporterID: graph.RootImporter, ProdOnly: true, Depth: -1})
	if err != nil {
		t.Fatal(err)
	}
	text := FormatDepTreeText("root", "1.0.0", tree)
	if !strings.Contains(text, "pkg-b@1.0.0") || !strings.Contains(text, "pkg-c@1.0.0") {
		t.Fatalf("text=%q", text)
	}
}

func TestCollectOutdatedRefsDirectOnly(t *testing.T) {
	g := testDepsGraph(t)
	refs := collectOutdatedRefs(g, graph.RootImporter, false, indexPackages(g), indexChildEdges(g))
	if len(refs) != 2 {
		t.Fatalf("direct refs=%d", len(refs))
	}
}

func TestOutdatedEntryForRef(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	pkgJSON := `{
  "name": "outdated-unit",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^1.0.0" }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	cfgPath := filepath.Join(dir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:                  dir,
		ProjectRoot:          dir,
		ProjectPath:          cfgPath,
		RequireProjectConfig: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		Ctx:    context.Background(),
		Config: eff,
	}
	proj, err := project.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	proj.Normalized, err = manifest.ToNormalized(proj.Doc)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	ref := outdatedRef{
		importer: graph.RootImporter,
		name:     "pkg-b",
		current:  "1.0.0",
		spec:     "^1.0.0",
		kind:     graph.DepProd,
		from:     ".",
	}
	entry, ok, err := outdatedEntryForRef(context.Background(), ac, proj, eng, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected outdated pkg-b")
	}
	if entry.Current != "1.0.0" || entry.Wanted != "1.2.0" || entry.Latest != "1.2.0" {
		t.Fatalf("%v", entry)
	}
}

func TestOutdatedOfflineWithoutCache(t *testing.T) {
	testkit.CleanEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
  "name": "offline-outdated",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^1.0.0" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		Ctx: context.Background(),
	}
	proj, err := project.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	proj.Normalized, err = manifest.ToNormalized(proj.Doc)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := config.Load(context.Background(), config.LoadOptions{
		CWD:         dir,
		ProjectRoot: dir,
		CLI:         map[string]any{"offline": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	ac.Config = eff
	eng, err := resolver.NewFromApp(ac.Config, proj)
	if err != nil {
		t.Fatal(err)
	}
	ref := outdatedRef{
		importer: graph.RootImporter,
		name:     "pkg-b",
		current:  "1.0.0",
		spec:     "^1.0.0",
		kind:     graph.DepProd,
		from:     ".",
	}
	_, _, err = outdatedEntryForRef(context.Background(), ac, proj, eng, ref)
	if err == nil {
		t.Fatal("expected offline error without cache")
	}
}
