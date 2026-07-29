package app_test

import (
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/graph"
)

func TestBuildMutationPlanDelta(t *testing.T) {
	prior := map[string]string{
		"pkg-a@1.0.0":   "1.0.0",
		"pkg-b@1.0.0":   "1.0.0",
		"old-pkg@1.0.0": "1.0.0",
	}
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-aaa"},
			{ID: graph.PackageID{Name: "pkg-b", Version: "2.0.0"}, Integrity: "sha256-bbb", TarballURL: "https://registry.example/pkg-b-2.0.0.tgz"},
			{ID: graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, Integrity: "sha256-ccc", TarballURL: "https://registry.example/pkg-c-1.0.0.tgz"},
		},
	}
	p, err := app.BuildMutationPlan(app.MutationPlanInput{PriorKeys: prior, Graph: g})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Desired) != 3 {
		t.Fatalf("desired=%d want 3", len(p.Desired))
	}
	var unlink, fetch, link int
	for _, op := range p.Operations {
		switch op.Op {
		case "unlink":
			unlink++
		case "fetch":
			fetch++
		case "link":
			link++
		}
	}
	if unlink != 2 {
		t.Fatalf("unlink=%d want 2 (old-pkg removed, pkg-b version key changed)", unlink)
	}
	if link != 2 {
		t.Fatalf("link=%d want 2 (pkg-b changed, pkg-c added)", link)
	}
	if fetch != 2 {
		t.Fatalf("fetch=%d want 2", fetch)
	}
}

func TestBuildMutationPlanNoOpWhenUnchanged(t *testing.T) {
	prior := map[string]string{"lodash@4.17.21": "4.17.21"}
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "lodash", Version: "4.17.21"}, Integrity: "sha256-aaa"},
		},
	}
	p, err := app.BuildMutationPlan(app.MutationPlanInput{PriorKeys: prior, Graph: g})
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range p.Operations {
		if op.Op == "link" || op.Op == "fetch" || op.Op == "unlink" {
			t.Fatalf("unexpected op %s on unchanged graph", op.Op)
		}
	}
}

func TestBuildMutationPlanIgnoreScripts(t *testing.T) {
	g := &graph.Graph{
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, Integrity: "sha256-aaa", TarballURL: "https://example/pkg-a.tgz"},
		},
	}
	p, err := app.BuildMutationPlan(app.MutationPlanInput{
		Graph:         g,
		IgnoreScripts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range p.Operations {
		if op.Op == "script" {
			t.Fatal("expected no script ops when ignore-scripts is set")
		}
	}
}
