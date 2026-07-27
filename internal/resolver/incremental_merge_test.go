package resolver

import (
	"testing"

	"github.com/mewisme/m/internal/graph"
)

func TestMatchingResolvedEdgeUsesFullParentKey(t *testing.T) {
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a1", "").
		Package(graph.PackageID{Name: "pkg-a", Version: "2.0.0"}, "sha256-a2", "").
		Package(graph.PackageID{Name: "shared", Version: "1.0.0"}, "sha256-s1", "").
		Package(graph.PackageID{Name: "shared", Version: "2.0.0"}, "sha256-s2", "").
		EdgeEx("pkg-a@1.0.0", "shared", "shared@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-a@2.0.0", "shared", "shared@2.0.0", graph.DepProd, "^2.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	pe := graph.Edge{From: "pkg-a@1.0.0", Name: "shared", To: "shared@1.0.0", Kind: graph.DepProd, Range: "^1.0.0"}
	got, ok := matchingResolvedEdge(pe, resolved, "")
	if !ok {
		t.Fatal("expected match for pkg-a@1.0.0 parent")
	}
	if got.To != "shared@1.0.0" {
		t.Fatalf("matched wrong target: %s", got.To)
	}
	pe2 := graph.Edge{From: "pkg-a@2.0.0", Name: "shared", To: "shared@2.0.0", Kind: graph.DepProd, Range: "^2.0.0"}
	got2, ok := matchingResolvedEdge(pe2, resolved, "")
	if !ok || got2.To != "shared@2.0.0" {
		t.Fatalf("expected distinct match for pkg-a@2.0.0, got %#v ok=%v", got2, ok)
	}
}

func TestExpandClosureForMergeMapsVersionBump(t *testing.T) {
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a1", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "").
		EdgeEx(string(graph.RootImporter), "pkg-a", "pkg-a@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-a@1.0.0", "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "2.0.0"}, "sha256-a2", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "").
		EdgeEx(string(graph.RootImporter), "pkg-a", "pkg-a@2.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-a@2.0.0", "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	closure := map[string]struct{}{"pkg-a@1.0.0": {}}
	expanded := expandClosureForMerge(prior, resolved, closure)
	if _, ok := expanded["pkg-a@2.0.0"]; !ok {
		t.Fatal("expected resolved pkg-a@2.0.0 in merge closure")
	}
	if _, ok := expanded["pkg-b@1.0.0"]; !ok {
		t.Fatal("expected pkg-b@1.0.0 in merge closure after version bump")
	}
}

func TestExpandClosureForMergeDoesNotCollapseSameNameParents(t *testing.T) {
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a1", "").
		Package(graph.PackageID{Name: "pkg-a", Version: "2.0.0"}, "sha256-a2", "").
		Package(graph.PackageID{Name: "leaf", Version: "1.0.0"}, "sha256-l1", "").
		Package(graph.PackageID{Name: "leaf", Version: "2.0.0"}, "sha256-l2", "").
		EdgeEx("pkg-a@1.0.0", "leaf", "leaf@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-a@2.0.0", "leaf", "leaf@2.0.0", graph.DepProd, "^2.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	resolved := prior
	closure := map[string]struct{}{"pkg-a@1.0.0": {}}
	expanded := expandClosureForMerge(prior, resolved, closure)
	if _, ok := expanded["leaf@2.0.0"]; ok {
		t.Fatal("pkg-a@2.0.0 sibling subtree should not enter closure for pkg-a@1.0.0 update")
	}
	if _, ok := expanded["leaf@1.0.0"]; !ok {
		t.Fatal("expected leaf@1.0.0 in merge closure")
	}
}
