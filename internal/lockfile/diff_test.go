package lockfile_test

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

func TestGraphsEqualIdentical(t *testing.T) {
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "a", Version: "1.0.0"}, Integrity: "sha512-abc"},
		},
		Edges: []graph.Edge{
			{From: ".", Name: "a", To: "a@1.0.0", Kind: graph.DepProd, Range: "1.0.0"},
		},
	}
	eq, err := lockfile.GraphsEqual(g, g)
	if err != nil {
		t.Fatal(err)
	}
	if !eq {
		t.Fatal("expected equal graphs")
	}
}

func TestDiffGraphsPackageChange(t *testing.T) {
	base := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "a", Version: "1.0.0"}},
		},
	}
	changed := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "b", Version: "2.0.0"}},
		},
	}
	diff, err := lockfile.DiffGraphs(base, changed)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.PackagesAdded) != 1 || diff.PackagesAdded[0] != "b@2.0.0" {
		t.Fatalf("packages added: %v", diff.PackagesAdded)
	}
	data, err := lockfile.EncodeDiffJSON(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected encoded diff")
	}
}

func TestDiffGraphsSpecifierChange(t *testing.T) {
	before := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "left-pad", Version: "1.3.0"}},
		},
		Edges: []graph.Edge{
			{From: ".", Name: "left-pad", To: "left-pad@1.3.0", Kind: graph.DepProd, Range: "1.3.0"},
		},
	}
	after := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "left-pad", Version: "1.3.0"}},
		},
		Edges: []graph.Edge{
			{From: ".", Name: "left-pad", To: "left-pad@1.3.0", Kind: graph.DepProd, Range: "^1.3.0"},
		},
	}
	diff, err := lockfile.DiffGraphs(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Specifiers) != 1 {
		t.Fatalf("specifiers: %v", diff.Specifiers)
	}
	if diff.Specifiers[0].Before != "1.3.0" || diff.Specifiers[0].After != "^1.3.0" {
		t.Fatalf("specifier diff: %+v", diff.Specifiers[0])
	}
}
