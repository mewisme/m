package resolver

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
)

func TestConsolidateDuplicateNames(t *testing.T) {
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha-a", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, "sha-b", "").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha-c", "").
		Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
		Edge(string(graph.RootImporter), "pkg-b@1.0.0", graph.DepProd, "^1.0.0").
		Edge("pkg-a@1.0.0", "pkg-b@1.2.0", graph.DepProd, "^1.0.0").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha-a", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, "sha-b", "").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha-c", "").
		Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
		Edge(string(graph.RootImporter), "pkg-b@1.0.0", graph.DepProd, "^1.0.0").
		Edge("pkg-a@1.0.0", "pkg-b@1.2.0", graph.DepProd, "^1.0.0").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	out, err := consolidateDuplicateNames(resolved, prior)
	if err != nil {
		t.Fatal(err)
	}
	versions := map[string]int{}
	for _, p := range out.Packages {
		if p.ID.Name == "pkg-b" {
			versions[p.ID.Version]++
		}
	}
	if len(versions) != 1 {
		t.Fatalf("expected one pkg-b version, got %v", versions)
	}
	if versions["1.2.0"] != 1 {
		t.Fatalf("expected pkg-b@1.2.0, got %v", versions)
	}
}
