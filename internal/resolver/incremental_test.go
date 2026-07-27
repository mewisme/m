package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/manifest"
	"github.com/mewisme/m/internal/resolver"
)

func TestIncrementalUpdatePreservesUnrelated(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)

	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a", "http://example/pkg-a.tgz").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "http://example/pkg-b.tgz").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, "sha256-c", "http://example/pkg-c.tgz").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "http://example/lodash.tgz").
		EdgeEx(string(graph.RootImporter), "pkg-a", "pkg-a@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		EdgeEx("pkg-a@1.0.0", "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-b@1.0.0", "pkg-c", "pkg-c@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Prior:             prior,
		Hints:             prior,
		UpdateTargets:     []string{"pkg-a"},
		IncrementalUpdate: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	versions := map[string]string{}
	for _, p := range res.Graph.Packages {
		versions[p.ID.Name] = p.ID.Version
	}
	if versions["pkg-b"] != "1.2.0" {
		t.Fatalf("pkg-b=%q want 1.2.0", versions["pkg-b"])
	}
	if versions["pkg-c"] != "1.0.1" {
		t.Fatalf("pkg-c=%q want 1.0.1", versions["pkg-c"])
	}
	if versions["lodash"] != "4.17.21" {
		t.Fatalf("lodash=%q want pinned 4.17.21", versions["lodash"])
	}
}

func TestIncrementalDirectDepsDefaultClosure(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-b": "^1.0.0" }
}`)

	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "http://example/pkg-b.tgz").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.0"}, "sha256-c", "http://example/pkg-c.tgz").
		EdgeEx(string(graph.RootImporter), "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx("pkg-b@1.0.0", "pkg-c", "pkg-c@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Prior:             prior,
		Hints:             prior,
		IncrementalUpdate: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "pkg-b" && p.ID.Version != "1.2.0" {
			t.Fatalf("pkg-b=%s want 1.2.0 with empty update targets", p.ID.Version)
		}
	}
}

func TestBuildUpdateClosure(t *testing.T) {
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "", "").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "", "").
		EdgeEx(string(graph.RootImporter), "pkg-a", "pkg-a@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		EdgeEx("pkg-a@1.0.0", "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	m := testManifest(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)
	closure := resolver.BuildUpdateClosure([]string{"pkg-a"}, prior, m)
	if _, ok := closure["pkg-a@1.0.0"]; !ok {
		t.Fatal("pkg-a@1.0.0 missing from closure")
	}
	if _, ok := closure["pkg-b@1.0.0"]; !ok {
		t.Fatal("pkg-b@1.0.0 missing from closure")
	}
	if _, ok := closure["lodash@4.17.21"]; ok {
		t.Fatal("lodash should not be in pkg-a update closure")
	}
}

func testManifest(t *testing.T, raw string) *manifest.Manifest {
	t.Helper()
	doc, err := manifest.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.ToNormalized(doc)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
