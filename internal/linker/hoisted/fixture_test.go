package hoisted_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker/hoisted"
	"github.com/mewisme/m/internal/testkit"
)

type linkerFixture struct {
	Packages []struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		Key           string `json:"key"`
		PeerProviders []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Key     string `json:"key"`
		} `json:"peerProviders,omitempty"`
	} `json:"packages"`
	Edges []struct {
		From string `json:"from"`
		Name string `json:"name,omitempty"`
		To   string `json:"to"`
		Kind string `json:"kind"`
	} `json:"edges"`
	Expect        map[string]string `json:"expect"`
	AlsoExpect    map[string]string `json:"alsoExpect,omitempty"`
	MaxPlacements int               `json:"maxPlacements,omitempty"`
}

func TestHoistedLinkerFixtures(t *testing.T) {
	fixtures := []string{
		"multi-version-x",
		"shared-child-blocked",
		"scoped-conflict",
		"nested-bins",
		"cyclic-graph",
		"peer-context-instances",
		"alias-target",
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			runLinkerFixture(t, testkit.FixtureDir(t, filepath.Join("linker", "hoisted", name, "graph.json")))
		})
	}
}

func runLinkerFixture(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx linkerFixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	g, err := graphFromFixture(&fx)
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	nm := filepath.Join(stage, "node_modules")
	got := placementsByKey(t, g, nm)
	for key, wantRel := range fx.Expect {
		want := filepath.Join(stage, filepath.FromSlash(wantRel))
		if !hasDest(got[key], want) {
			t.Fatalf("%s: got %v want %q", key, got[key], want)
		}
	}
	for key, wantRel := range fx.AlsoExpect {
		want := filepath.Join(stage, filepath.FromSlash(wantRel))
		if !hasDest(got[key], want) {
			t.Fatalf("alsoExpect %s: got %v want %q", key, got[key], want)
		}
	}
	if fx.MaxPlacements > 0 {
		ps, err := hoisted.Placements(g, nm)
		if err != nil {
			t.Fatal(err)
		}
		if len(ps) > fx.MaxPlacements {
			t.Fatalf("placements=%d want <= %d", len(ps), fx.MaxPlacements)
		}
	}
}

func placementsByKey(t *testing.T, g *graph.Graph, nmRoot string) map[string][]string {
	t.Helper()
	ps, err := hoisted.Placements(g, nmRoot)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string][]string)
	for _, p := range ps {
		out[p.Key] = append(out[p.Key], p.DestDir)
	}
	return out
}

func hasDest(dests []string, want string) bool {
	for _, dest := range dests {
		if filepath.Clean(dest) == filepath.Clean(want) {
			return true
		}
	}
	return false
}

func graphFromFixture(fx *linkerFixture) (*graph.Graph, error) {
	b := graph.NewBuilder().Importer(graph.RootImporter, "root")
	for _, p := range fx.Packages {
		id := graph.PackageID{Name: p.Name, Version: p.Version}
		for _, peer := range p.PeerProviders {
			id.PeerProviderContext = append(id.PeerProviderContext, graph.PeerProvider{
				Name: peer.Name, Version: peer.Version, Key: peer.Key,
			})
		}
		id.Normalize()
		b = b.Package(id, "", "")
	}
	for _, e := range fx.Edges {
		name := e.Name
		if name == "" {
			name = graph.TargetNameFromKey(e.To)
		}
		b = b.EdgeEx(e.From, name, e.To, graph.DepKind(e.Kind), "", false)
	}
	return b.Build()
}

func TestPlacementIDDeterministicSort(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "", "").
			Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
			Edge(string(graph.RootImporter), "pkg-b@1.0.0", graph.DepProd, "").
			Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "")
	})
	ps, err := hoisted.Placements(g, nm)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(ps); i++ {
		if ps[i].ID.Compare(ps[i-1].ID) < 0 {
			t.Fatalf("not sorted at %d", i)
		}
	}
}

func TestNestedBinsRootDirectOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("covered by fixture expectations")
	}
	stage := t.TempDir()
	nm := filepath.Join(stage, "node_modules")
	pkgDir := filepath.Join(stage, "extract", "pkg-cli")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"pkg-cli","version":"1.0.0","bin":{"cli":"./cli.js"}}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	g := testGraph(t, func(b *graph.Builder) *graph.Builder {
		return b.
			Package(graph.PackageID{Name: "pkg-cli", Version: "1.0.0"}, "", "").
			Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "", "").
			Edge(string(graph.RootImporter), "pkg-cli@1.0.0", graph.DepProd, "").
			Edge(string(graph.RootImporter), "lodash@4.17.21", graph.DepProd, "")
	})
	l := &hoisted.Linker{
		NodeModules: nm,
		ExtractDirs: map[string]string{
			"pkg-cli@1.0.0":  pkgDir,
			"lodash@4.17.21": pkgDir,
		},
	}
	plan, err := l.Plan(t.Context(), g)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Bins) != 1 || plan.Bins[0].Cmd != "cli" {
		t.Fatalf("bins=%+v", plan.Bins)
	}
	if filepath.Clean(plan.Bins[0].NodeModules) != filepath.Clean(nm) {
		t.Fatalf("root bin nm got %q", plan.Bins[0].NodeModules)
	}
}
