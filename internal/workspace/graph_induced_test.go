package workspace_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/workspace"
)

func TestInducedSubgraphDAG(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":               `{"name":"root","workspaces":["packages/*"]}`,
		"packages/base/package.json": `{"name":"base","version":"1.0.0"}`,
		"packages/lib/package.json": `{
  "name":"lib","version":"1.0.0",
  "dependencies":{"base":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{"packages/base", "packages/lib"}
	edges, inDegree, err := workspace.InducedSubgraph(g, paths)
	if err != nil {
		t.Fatal(err)
	}
	if inDegree["packages/lib"] != 1 {
		t.Fatalf("lib inDegree=%d", inDegree["packages/lib"])
	}
	if len(edges["packages/base"]) != 1 {
		t.Fatalf("edges=%v", edges)
	}
}

func TestValidateSelectedCycleSelectedOnly(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json": `{"name":"root","workspaces":["packages/*"]}`,
		"packages/a/package.json": `{
  "name":"a","version":"1.0.0",
  "dependencies":{"b":"workspace:*"}
}`,
		"packages/b/package.json": `{
  "name":"b","version":"1.0.0",
  "dependencies":{"a":"workspace:*"}
}`,
		"packages/c/package.json": `{"name":"c","version":"1.0.0"}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.ValidateSelectedCycle(g, []string{"packages/a", "packages/b"}); err == nil {
		t.Fatal("expected cycle in selected set")
	}
	if err := workspace.ValidateSelectedCycle(g, []string{"packages/c"}); err != nil {
		t.Fatalf("unrelated cycle should not block c: %v", err)
	}
}

func TestReverseTopoOrder(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":               `{"name":"root","workspaces":["packages/*"]}`,
		"packages/base/package.json": `{"name":"base","version":"1.0.0"}`,
		"packages/app/package.json": `{
  "name":"app","version":"1.0.0",
  "dependencies":{"base":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{"packages/base", "packages/app"}
	rev, err := g.ReverseTopoOrder(paths)
	if err != nil {
		t.Fatal(err)
	}
	if rev[0] != "packages/app" {
		t.Fatalf("reverse topo=%v", rev)
	}
}

func TestTopoOrderForSubset(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":               `{"name":"root","workspaces":["packages/*"]}`,
		"packages/base/package.json": `{"name":"base","version":"1.0.0"}`,
		"packages/app/package.json": `{
  "name":"app","version":"1.0.0",
  "dependencies":{"base":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	order, err := g.TopoOrderFor([]string{"packages/base", "packages/app"})
	if err != nil {
		t.Fatal(err)
	}
	if order[0] != "packages/base" {
		t.Fatalf("order=%v", order)
	}
}

func TestInducedSubgraphErrors(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json": `{"name":"root","workspaces":["packages/*"]}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = workspace.InducedSubgraph(g, []string{"missing"})
	if err == nil || apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("got %v", err)
	}
}
