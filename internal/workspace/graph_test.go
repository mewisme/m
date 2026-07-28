package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/workspace"
)

func writeWS(t *testing.T, layout map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range layout {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBuildGraphDuplicateNames(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":            `{"name":"root","workspaces":["packages/*"]}`,
		"packages/a/package.json": `{"name":"dup","version":"1.0.0"}`,
		"packages/b/package.json": `{"name":"dup","version":"2.0.0"}`,
	})
	_, err := workspace.BuildGraph(root)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestTopoOrderDependenciesFirst(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json":               `{"name":"root","workspaces":["packages/*"]}`,
		"packages/base/package.json": `{"name":"base","version":"1.0.0"}`,
		"packages/app/package.json": `{
  "name":"app",
  "version":"1.0.0",
  "dependencies": {"base":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	order, err := g.TopoOrder()
	if err != nil {
		t.Fatal(err)
	}
	baseIdx, appIdx := -1, -1
	for i, p := range order {
		switch p {
		case "packages/base":
			baseIdx = i
		case "packages/app":
			appIdx = i
		}
	}
	if baseIdx < 0 || appIdx < 0 {
		t.Fatalf("order=%v", order)
	}
	if baseIdx >= appIdx {
		t.Fatalf("base should precede app: %v", order)
	}
}

func TestTopoOrderCycle(t *testing.T) {
	root := writeWS(t, map[string]string{
		"package.json": `{"name":"root","workspaces":["packages/*"]}`,
		"packages/a/package.json": `{
  "name":"a","version":"1.0.0",
  "dependencies": {"b":"workspace:*"}
}`,
		"packages/b/package.json": `{
  "name":"b","version":"1.0.0",
  "dependencies": {"a":"workspace:*"}
}`,
	})
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.TopoOrder()
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
