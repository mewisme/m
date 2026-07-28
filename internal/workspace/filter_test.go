package workspace_test

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/workspace"
)

func TestExpandFilterNameAndNegation(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":            `{"name":"root","workspaces":["packages/*"]}`,
		"packages/a/package.json": `{"name":"a","version":"1.0.0"}`,
		"packages/b/package.json": `{"name":"b","version":"1.0.0"}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := workspace.ExpandFilter(g, []string{"a", "!b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != graph.ImporterID("packages/a") {
		t.Fatalf("ids=%v", ids)
	}
}

func TestExpandFilterDepsClosure(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":               `{"name":"root","workspaces":["packages/*"]}`,
		"packages/base/package.json": `{"name":"base","version":"1.0.0"}`,
		"packages/app/package.json": `{
  "name":"app","version":"1.0.0",
  "dependencies": {"base":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := workspace.ExpandFilter(g, []string{"...app"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[graph.ImporterID]bool{
		"packages/app":  true,
		"packages/base": true,
	}
	for _, id := range ids {
		if !want[id] {
			t.Fatalf("unexpected id %s in %v", id, ids)
		}
		delete(want, id)
	}
	if len(want) != 0 {
		t.Fatalf("missing ids %v", want)
	}
}
