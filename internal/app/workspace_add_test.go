package app_test

import (
	"testing"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/resolver"
)

func TestUntouchedImporterIDs(t *testing.T) {
	prior := &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/a"},
			{ID: "packages/b"},
		},
	}
	active := &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/a"},
		},
	}
	untouched := app.UntouchedImporterIDsForTest(prior, active)
	if len(untouched) != 1 || untouched[0] != "packages/b" {
		t.Fatalf("got %v", untouched)
	}
}

func TestMergeFilteredWorkspaceResolutionPreservesClosure(t *testing.T) {
	prior := &graph.Graph{
		Importers: []graph.Importer{
			{ID: graph.RootImporter},
			{ID: "packages/alpha"},
			{ID: "packages/beta"},
		},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-a", TarballURL: "http://x/pkg-a.tgz"},
			{ID: graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, Integrity: "sha256-b", TarballURL: "http://x/pkg-b.tgz"},
		},
		Edges: []graph.Edge{
			{From: "packages/alpha", Name: "pkg-a", To: "pkg-a@1.0.0"},
			{From: "packages/beta", Name: "pkg-b", To: "pkg-b@1.2.0"},
		},
	}
	active := &resolver.Resolution{
		Graph: &graph.Graph{
			Importers: []graph.Importer{
				{ID: graph.RootImporter},
				{ID: "packages/alpha"},
			},
			Packages: []graph.Package{
				{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-a", TarballURL: "http://x/pkg-a.tgz"},
			},
			Edges: []graph.Edge{
				{From: "packages/alpha", Name: "pkg-a", To: "pkg-a@1.0.0"},
			},
		},
	}
	merged, err := app.MergeFilteredWorkspaceResolutionForTest(prior, nil, active, []graph.ImporterID{"packages/beta"})
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, p := range merged.Graph.Packages {
		keys[p.ID.Key()] = true
	}
	if !keys["pkg-b@1.2.0"] {
		t.Fatalf("pkg-b dropped from merged graph: %v", keys)
	}
}
