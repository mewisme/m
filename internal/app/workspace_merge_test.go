package app_test

import (
	"encoding/json"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/resolver"
)

func TestMergeFilteredWorkspaceResolutionPreservesBetaOnly(t *testing.T) {
	prior := buildAlphaBetaPriorGraph()
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{
				{ID: graph.RootImporter},
				{ID: "packages/alpha"},
			},
			Packages: []graph.Package{
				{ID: graph.PackageID{Name: "shared", Version: "1.0.0"}, Integrity: "sha256-s", TarballURL: "http://x/shared.tgz"},
				{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-a", TarballURL: "http://x/pkg-a.tgz"},
			},
			Edges: []graph.Edge{
				{From: "packages/alpha", Name: "shared", To: "shared@1.0.0"},
				{From: "packages/alpha", Name: "pkg-a", To: "pkg-a@1.0.0"},
			},
		},
	}
	merged, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, []graph.ImporterID{"packages/beta"})
	if err != nil {
		t.Fatal(err)
	}
	keys := packageKeys(merged.Graph)
	for _, want := range []string{"shared@1.0.0", "pkg-b@1.2.0", "pkg-c@1.0.0", "beta-only@2.0.0"} {
		if !keys[want] {
			t.Fatalf("missing %s in merged graph: %v", want, keys)
		}
	}
	if !hasMergedEdge(merged.Graph, "pkg-b@1.2.0", "pkg-c", "pkg-c@1.0.0") {
		t.Fatal("missing preserved package-to-package edge pkg-b -> pkg-c")
	}
}

func TestMergePreservesPackageToPackageForUntouchedBeta(t *testing.T) {
	prior := buildAlphaBetaPriorGraph()
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/alpha"}},
			Packages: []graph.Package{
				{ID: graph.PackageID{Name: "shared", Version: "1.0.0"}, Integrity: "sha256-s", TarballURL: "http://x/shared.tgz"},
			},
			Edges: []graph.Edge{{From: "packages/alpha", Name: "shared", To: "shared@1.0.0"}},
		},
	}
	merged, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, []graph.ImporterID{"packages/beta"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct{ from, name, to string }{
		{"packages/beta", "pkg-b", "pkg-b@1.2.0"},
		{"pkg-b@1.2.0", "pkg-c", "pkg-c@1.0.0"},
	} {
		if !hasMergedEdge(merged.Graph, want.from, want.name, want.to) {
			t.Fatalf("missing edge %s %s -> %s", want.from, want.name, want.to)
		}
	}
}

func TestMergePriorMalformedEdgeFails(t *testing.T) {
	prior := &graph.Graph{
		Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/beta"}},
		Packages:  []graph.Package{{ID: graph.PackageID{Name: "pkg-b", Version: "1.2.0"}}},
		Edges:     []graph.Edge{{From: "packages/beta", Name: "ghost", To: "ghost@9.9.9"}},
	}
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/alpha"}},
		},
	}
	_, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, []graph.ImporterID{"packages/beta"})
	if err == nil {
		t.Fatal("expected merge validation error for malformed prior edge")
	}
}

func TestMergeFilteredWorkspaceResolutionDeterministic(t *testing.T) {
	prior := buildAlphaBetaPriorGraph()
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/alpha"}},
			Packages: []graph.Package{
				{ID: graph.PackageID{Name: "shared", Version: "1.0.0"}, Integrity: "sha256-s", TarballURL: "http://x/shared.tgz"},
			},
			Edges: []graph.Edge{
				{From: "packages/alpha", Name: "shared", To: "shared@1.0.0"},
			},
		},
	}
	untouched := []graph.ImporterID{"packages/beta"}
	a, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, untouched)
	if err != nil {
		t.Fatal(err)
	}
	b, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, untouched)
	if err != nil {
		t.Fatal(err)
	}
	ja, err := json.Marshal(a.Graph)
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.Marshal(b.Graph)
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Fatalf("non-deterministic merge:\n%s\n---\n%s", ja, jb)
	}
}

func TestMergeFilteredWorkspaceResolutionDanglingEdgeFails(t *testing.T) {
	prior := &graph.Graph{
		Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/beta"}},
		Edges:     []graph.Edge{{From: "packages/beta", Name: "missing", To: "ghost@9.9.9"}},
	}
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/alpha"}},
		},
	}
	_, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, []graph.ImporterID{"packages/beta"})
	if err == nil {
		t.Fatal("expected merge validation error for dangling prior edge")
	}
}

func TestMergeFilteredWorkspaceResolutionPreservesLocalExtension(t *testing.T) {
	prior := buildAlphaBetaPriorGraph()
	priorExt := lockfile.Extensions{
		resolver.LocalExtensionKey: []byte(`{"pkg-b@1.2.0":{"path":"packages/lib"}}`),
	}
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{{ID: graph.RootImporter}, {ID: "packages/alpha"}},
			Packages: []graph.Package{
				{ID: graph.PackageID{Name: "shared", Version: "1.0.0"}, Integrity: "sha256-s", TarballURL: "http://x/shared.tgz"},
			},
			Edges: []graph.Edge{{From: "packages/alpha", Name: "shared", To: "shared@1.0.0"}},
		},
	}
	merged, err := app.MergeFilteredWorkspaceResolutionForTest(prior, priorExt, active, []graph.ImporterID{"packages/beta"})
	if err != nil {
		t.Fatal(err)
	}
	locals, err := resolver.DecodeLocalSources(merged.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := locals["pkg-b@1.2.0"]; !ok {
		t.Fatalf("local extension for pkg-b not preserved: %v", locals)
	}
}

func buildAlphaBetaPriorGraph() *graph.Graph {
	return &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/alpha"},
			{ID: "packages/beta"},
		},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "shared", Version: "1.0.0"}, Integrity: "sha256-s", TarballURL: "http://x/shared.tgz"},
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-a", TarballURL: "http://x/pkg-a.tgz"},
			{ID: graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, Integrity: "sha256-b", TarballURL: "http://x/pkg-b.tgz"},
			{ID: graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, Integrity: "sha256-c", TarballURL: "http://x/pkg-c.tgz"},
			{ID: graph.PackageID{Name: "beta-only", Version: "2.0.0"}, Integrity: "sha256-bo", TarballURL: "http://x/beta-only.tgz"},
		},
		Edges: []graph.Edge{
			{From: "packages/alpha", Name: "shared", To: "shared@1.0.0"},
			{From: "packages/alpha", Name: "pkg-a", To: "pkg-a@1.0.0"},
			{From: "packages/beta", Name: "shared", To: "shared@1.0.0"},
			{From: "packages/beta", Name: "pkg-b", To: "pkg-b@1.2.0"},
			{From: "packages/beta", Name: "beta-only", To: "beta-only@2.0.0"},
			{From: "pkg-b@1.2.0", Name: "pkg-c", To: "pkg-c@1.0.0"},
		},
	}
}

func packageKeys(g *graph.Graph) map[string]bool {
	out := map[string]bool{}
	for _, p := range g.Packages {
		out[p.ID.Key()] = true
	}
	return out
}

func hasMergedEdge(g *graph.Graph, from, name, to string) bool {
	for _, e := range g.Edges {
		if e.From == from && e.Name == name && e.To == to {
			return true
		}
	}
	return false
}
