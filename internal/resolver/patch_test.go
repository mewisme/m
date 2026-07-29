package resolver

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
)

func TestFinalizePatchTargetsOrphanPatch(t *testing.T) {
	s := &resolveState{
		patches: &patchState{
			bySelector: map[string]patchRecord{
				"ghost@1.0.0": {path: "/tmp/x.patch", hash: "abc"},
			},
			byPkgKey: map[string]patchRecord{},
		},
	}
	g := &graph.Graph{
		Packages: []graph.Package{{ID: graph.PackageID{Name: "lodash", Version: "4.17.21"}}},
	}
	if err := s.finalizePatchTargets(g); err == nil {
		t.Fatal("expected orphan patch error")
	}
}

func TestFinalizePatchTargetsMatchesPackage(t *testing.T) {
	s := &resolveState{
		patches: &patchState{
			bySelector: map[string]patchRecord{
				"ms@2.1.3": {path: "/tmp/ms.patch", hash: "abc123"},
			},
			byPkgKey: map[string]patchRecord{},
		},
	}
	g := &graph.Graph{
		Packages: []graph.Package{{ID: graph.PackageID{Name: "ms", Version: "2.1.3"}}},
	}
	if err := s.finalizePatchTargets(g); err != nil {
		t.Fatal(err)
	}
	want := "ms@2.1.3(patch_hash=abc123)"
	if g.Packages[0].ID.Key() != want {
		t.Fatalf("key=%q want %q", g.Packages[0].ID.Key(), want)
	}
}
