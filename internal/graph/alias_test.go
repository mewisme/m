package graph_test

import (
	"testing"

	"github.com/mewisme/mew/internal/graph"
)

func TestEdgeNameAliasRoundTrip(t *testing.T) {
	g, err := graph.NewBuilder().
		Importer(graph.RootImporter, "app").
		Package(graph.PackageID{Name: "bar", Version: "1.0.0"}, "sha512-bar", "").
		EdgeEx(".", "foo", "bar@1.0.0", graph.DepProd, "npm:bar@^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("edges=%d", len(g.Edges))
	}
	e := g.Edges[0]
	if e.Name != "foo" || e.To != "bar@1.0.0" || e.Range != "npm:bar@^1.0.0" {
		t.Fatalf("edge=%#v", e)
	}
}

func TestMigrateV2InfersEdgeName(t *testing.T) {
	g := &graph.Graph{
		SchemaVersion: 2,
		Importers:     []graph.Importer{{ID: graph.RootImporter, Path: "."}},
		Packages:      []graph.Package{{ID: graph.PackageID{Name: "lodash", Version: "4.17.21"}}},
		Edges:         []graph.Edge{{From: ".", To: "lodash@4.17.21", Kind: graph.DepProd, Range: "^4.17.0"}},
	}
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	if g.SchemaVersion != graph.SchemaVersion {
		t.Fatalf("schemaVersion=%d", g.SchemaVersion)
	}
	if g.Edges[0].Name != "lodash" {
		t.Fatalf("name=%q", g.Edges[0].Name)
	}
}

func TestTargetNameFromKey(t *testing.T) {
	cases := map[string]string{
		"lodash@4.17.21":                "lodash",
		"@scope/pkg@1.0.0":              "@scope/pkg",
		"react@18.2.0#react-dom@18.2.0": "react",
		"plugin@1.0.0#provisional:inst": "plugin",
	}
	for key, want := range cases {
		if got := graph.TargetNameFromKey(key); got != want {
			t.Fatalf("TargetNameFromKey(%q)=%q want %q", key, got, want)
		}
	}
}
