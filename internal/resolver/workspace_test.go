package resolver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/resolver"
)

func writeWorkspace(t testing.TB, layout map[string]string) string {
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

func TestResolveWorkspaceStar(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": { "pkg-a": "workspace:*" }
}`,
		"packages/a/package.json": `{
  "name": "pkg-a",
  "version": "2.3.4",
  "dependencies": { "lodash": "4.17.21" }
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-a" && p.ID.Version == "2.3.4" {
			found = true
			if p.Integrity != "" || p.TarballURL != "" {
				t.Fatalf("workspace package should have empty dist fields: %#v", p)
			}
		}
	}
	if !found {
		t.Fatalf("missing workspace package: %#v", res.Graph.Packages)
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		t.Fatal(err)
	}
	if src, ok := locals["pkg-a@2.3.4"]; !ok || src.Protocol != "workspace" || src.Path != "packages/a" {
		t.Fatalf("local extension=%#v", locals)
	}
}

func TestResolveWorkspaceCaret(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": { "pkg-a": "workspace:^" }
}`,
		"packages/a/package.json": `{
  "name": "pkg-a",
  "version": "1.0.0"
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range res.Graph.Edges {
		if e.From == string(graph.RootImporter) && strings.HasPrefix(e.To, "pkg-a@") {
			if e.Range != "workspace:^" {
				t.Fatalf("edge range=%q want workspace:^", e.Range)
			}
			return
		}
	}
	t.Fatalf("missing workspace edge: %#v", res.Graph.Edges)
}

func TestResolveWorkspaceMissingTarget(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": { "missing-pkg": "workspace:*" }
}`,
	})
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), `workspace target "missing-pkg" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWorkspaceMultiImporter(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeWorkspace(t, map[string]string{
		"package.json": `{
  "name": "root",
  "version": "1.0.0",
  "workspaces": ["packages/*"],
  "dependencies": { "shared": "workspace:*" }
}`,
		"packages/shared/package.json": `{
  "name": "shared",
  "version": "1.0.0"
}`,
		"packages/a/package.json": `{
  "name": "a",
  "version": "1.0.0",
  "dependencies": { "shared": "workspace:^" }
}`,
	})
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: testStrictPeersOff(),
	})
	if err != nil {
		t.Fatal(err)
	}
	importers := map[string]bool{}
	for _, im := range res.Graph.Importers {
		importers[string(im.ID)] = true
	}
	if !importers["."] || !importers["packages/a"] {
		t.Fatalf("importers=%v", importers)
	}
	edgeFrom := map[string]int{}
	for _, e := range res.Graph.Edges {
		if strings.HasPrefix(e.To, "shared@") {
			edgeFrom[e.From]++
		}
	}
	if edgeFrom["."] != 1 || edgeFrom["packages/a"] != 1 {
		t.Fatalf("shared edges=%v", edgeFrom)
	}
}

func testStrictPeersOff() *policy.Policy {
	return &policy.Policy{StrictPeerDependencies: false}
}
