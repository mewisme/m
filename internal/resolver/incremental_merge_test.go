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
	got, ok := matchingResolvedEdge(pe, resolved, "", nil)
	if !ok {
		t.Fatal("expected match for pkg-a@1.0.0 parent")
	}
	if got.To != "shared@1.0.0" {
		t.Fatalf("matched wrong target: %s", got.To)
	}
	pe2 := graph.Edge{From: "pkg-a@2.0.0", Name: "shared", To: "shared@2.0.0", Kind: graph.DepProd, Range: "^2.0.0"}
	got2, ok := matchingResolvedEdge(pe2, resolved, "", nil)
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

func TestMatchingResolvedEdgeAllowsRangeChangeInClosure(t *testing.T) {
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "foo", Version: "2.0.0"}, "sha256-f2", "").
		EdgeEx(string(graph.RootImporter), "foo", "foo@2.0.0", graph.DepProd, "^2.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	pe := graph.Edge{
		From: string(graph.RootImporter), Name: "foo", To: "foo@1.0.0",
		Kind: graph.DepProd, Range: "^1.0.0",
	}
	closure := map[string]struct{}{"foo@1.0.0": {}}
	got, ok := matchingResolvedEdge(pe, resolved, string(graph.RootImporter), closure)
	if !ok || got.To != "foo@2.0.0" || got.Range != "^2.0.0" {
		t.Fatalf("got %#v ok=%v", got, ok)
	}
}

func TestMatchingResolvedEdgeRequiresRangeOutsideClosure(t *testing.T) {
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	pe := graph.Edge{
		From: string(graph.RootImporter), Name: "lodash", To: "lodash@4.17.21",
		Kind: graph.DepProd, Range: "^4.17.21",
	}
	_, ok := matchingResolvedEdge(pe, resolved, string(graph.RootImporter), nil)
	if ok {
		t.Fatal("unaffected edges must match range exactly")
	}
}

func TestExpandClosureForMergeRangeBump(t *testing.T) {
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "foo", Version: "1.0.0"}, "sha256-f1", "").
		EdgeEx(string(graph.RootImporter), "foo", "foo@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "foo", Version: "2.0.0"}, "sha256-f2", "").
		EdgeEx(string(graph.RootImporter), "foo", "foo@2.0.0", graph.DepProd, "^2.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	closure := map[string]struct{}{"foo@1.0.0": {}}
	expanded := expandClosureForMerge(prior, resolved, closure)
	if _, ok := expanded["foo@2.0.0"]; !ok {
		t.Fatal("expected resolved foo@2.0.0 in merge closure after range bump")
	}
}

func TestMatchingResolvedEdgeProdDevDistinct(t *testing.T) {
	resolved, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "dual", Version: "1.0.0"}, "sha256-d", "").
		EdgeEx(string(graph.RootImporter), "dual", "dual@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx(string(graph.RootImporter), "dual", "dual@1.0.0", graph.DepDev, "^2.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	pe := graph.Edge{
		From: string(graph.RootImporter), Name: "dual", To: "dual@1.0.0",
		Kind: graph.DepDev, Range: "^1.0.0",
	}
	closure := map[string]struct{}{"dual@1.0.0": {}}
	got, ok := matchingResolvedEdge(pe, resolved, string(graph.RootImporter), closure)
	if !ok || got.Kind != graph.DepDev || got.Range != "^2.0.0" {
		t.Fatalf("got %#v ok=%v", got, ok)
	}
}
