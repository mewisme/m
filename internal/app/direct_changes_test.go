package app

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
)

// pkgKey builds the graph key form used by PackageID.Key().
func testGraph(importers []graph.Importer, pkgs []graph.Package, edges []graph.Edge) *graph.Graph {
	return &graph.Graph{SchemaVersion: 1, Importers: importers, Packages: pkgs, Edges: edges}
}

func pkg(name, version string) graph.Package {
	return graph.Package{ID: graph.PackageID{Name: name, Version: version}}
}

// A direct removal exists only in the prior graph. Deriving direct keys from
// the new graph alone drops it; the union must keep it.
func TestDirectRemovalSurvivesUnion(t *testing.T) {
	removed := pkg("ms", "2.1.3")
	prior := testGraph(
		nil,
		[]graph.Package{removed},
		[]graph.Edge{{From: string(graph.RootImporter), To: removed.ID.Key()}},
	)
	next := testGraph(nil, nil, nil)

	fromNextOnly := directPackageKeys(next)
	if fromNextOnly[removed.ID.Key()] {
		t.Fatal("new graph alone should not know the removed key")
	}

	union := directPackageKeysAcross(prior, next)
	if !union[removed.ID.Key()] {
		t.Fatalf("union must include the direct removal: %v", union)
	}
}

// A direct addition exists only in the new graph.
func TestDirectAdditionSurvivesUnion(t *testing.T) {
	added := pkg("ms", "2.1.3")
	prior := testGraph(nil, nil, nil)
	next := testGraph(
		nil,
		[]graph.Package{added},
		[]graph.Edge{{From: string(graph.RootImporter), To: added.ID.Key()}},
	)
	union := directPackageKeysAcross(prior, next)
	if !union[added.ID.Key()] {
		t.Fatalf("union must include the direct addition: %v", union)
	}
}

// Transitive-only packages must never be reported as direct.
func TestTransitiveNotDirect(t *testing.T) {
	direct := pkg("a", "1.0.0")
	trans := pkg("b", "1.0.0")
	g := testGraph(
		nil,
		[]graph.Package{direct, trans},
		[]graph.Edge{
			{From: string(graph.RootImporter), To: direct.ID.Key()},
			{From: direct.ID.Key(), To: trans.ID.Key()},
		},
	)
	keys := directPackageKeys(g)
	if !keys[direct.ID.Key()] {
		t.Fatal("direct dependency missing")
	}
	if keys[trans.ID.Key()] {
		t.Fatal("transitive dependency reported as direct")
	}
}

// A workspace filter must scope direct changes to the targeted importer.
func TestDirectKeysRespectWorkspaceFilter(t *testing.T) {
	aDep := pkg("dep-a", "1.0.0")
	bDep := pkg("dep-b", "1.0.0")
	rootDep := pkg("dep-root", "1.0.0")
	g := testGraph(
		[]graph.Importer{
			{ID: "packages/a", Name: "pkg-a", Path: "packages/a"},
			{ID: "packages/b", Name: "pkg-b", Path: "packages/b"},
		},
		[]graph.Package{aDep, bDep, rootDep},
		[]graph.Edge{
			{From: "packages/a", To: aDep.ID.Key()},
			{From: "packages/b", To: bDep.ID.Key()},
			{From: string(graph.RootImporter), To: rootDep.ID.Key()},
		},
	)

	all := directPackageKeys(g)
	for _, want := range []string{aDep.ID.Key(), bDep.ID.Key(), rootDep.ID.Key()} {
		if !all[want] {
			t.Fatalf("unfiltered set missing %s: %v", want, all)
		}
	}

	// Filter by importer ID and by package name; both must scope the result.
	for _, filter := range []string{"packages/a", "pkg-a"} {
		got := directPackageKeys(g, filter)
		if !got[aDep.ID.Key()] {
			t.Fatalf("filter %q lost the targeted importer's dep: %v", filter, got)
		}
		if got[bDep.ID.Key()] {
			t.Fatalf("filter %q leaked a sibling importer's dep: %v", filter, got)
		}
		if got[rootDep.ID.Key()] {
			t.Fatalf("filter %q leaked the root importer's dep: %v", filter, got)
		}
	}
}

// A version update of a direct dependency must be reported as direct, and the
// output must keep diffKeys' deterministic ordering.
func TestDirectUpdateReportedAndOrdered(t *testing.T) {
	oldMs := pkg("ms", "2.1.2")
	newMs := pkg("ms", "2.1.3")
	oldDebug := pkg("debug", "4.3.0")
	newDebug := pkg("debug", "4.3.4")

	prior := testGraph(nil, []graph.Package{oldMs, oldDebug}, []graph.Edge{
		{From: string(graph.RootImporter), To: oldMs.ID.Key()},
		{From: string(graph.RootImporter), To: oldDebug.ID.Key()},
	})
	next := testGraph(nil, []graph.Package{newMs, newDebug}, []graph.Edge{
		{From: string(graph.RootImporter), To: newMs.ID.Key()},
		{From: string(graph.RootImporter), To: newDebug.ID.Key()},
	})

	res := diffKeys(packageKeysFromGraph(prior), packageKeysFromGraph(next))
	directs := filterDirectChanges(res.PackageChanges, directPackageKeysAcross(prior, next))
	if len(directs) == 0 {
		t.Fatal("direct version updates not reported")
	}
	// filterDirectChanges preserves diffKeys' order, so the same input must
	// produce the same output order every time.
	again := filterDirectChanges(res.PackageChanges, directPackageKeysAcross(prior, next))
	for i := range directs {
		if directs[i] != again[i] {
			t.Fatalf("ordering not deterministic at %d: %+v vs %+v", i, directs[i], again[i])
		}
	}
}
