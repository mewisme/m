package graph_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

func moduleRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestPackageIDKey(t *testing.T) {
	plain := graph.PackageID{Name: "lodash", Version: "4.17.21"}
	if got := plain.Key(); got != "lodash@4.17.21" {
		t.Fatalf("got %q", got)
	}
	withPeers := graph.PackageID{
		Name:    "react",
		Version: "18.2.0",
		PeerProviderContext: graph.PeerProviderContext{
			{Name: "react-dom", Version: "18.0.0", Key: "react-dom@18.0.0"},
			{Name: "scheduler", Version: "0.23.0", Key: "scheduler@0.23.0"},
		},
	}
	withPeers.Normalize()
	want := "react@18.2.0#react-dom@18.0.0,scheduler@0.23.0"
	if got := withPeers.Key(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	root := moduleRoot(t)
	for _, name := range []string{"simple-app.json", "peers.json", "workspace.json"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, "testdata", "graph", name)
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			g, err := graph.DecodeJSON(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := graph.EncodeJSON(g)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("encode mismatch\ngot:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestSortStability(t *testing.T) {
	g, err := graph.NewBuilder().
		Importer("packages/z", "z").
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "b", Version: "1.0.0"}, "", "").
		Package(graph.PackageID{Name: "a", Version: "1.0.0"}, "", "").
		Edge("packages/z", "b@1.0.0", graph.DepProd, "*").
		Edge(".", "a@1.0.0", graph.DepProd, "*").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	first, err := graph.EncodeJSON(g)
	if err != nil {
		t.Fatal(err)
	}
	g2, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Importer("packages/z", "z").
		Package(graph.PackageID{Name: "a", Version: "1.0.0"}, "", "").
		Package(graph.PackageID{Name: "b", Version: "1.0.0"}, "", "").
		Edge(".", "a@1.0.0", graph.DepProd, "*").
		Edge("packages/z", "b@1.0.0", graph.DepProd, "*").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	second, err := graph.EncodeJSON(g2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("ordering not stable\n%s\nvs\n%s", first, second)
	}
}

func TestPeerContextCollision(t *testing.T) {
	id := graph.PackageID{Name: "pkg", Version: "1.0.0"}
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter, Path: "."}},
		Packages: []graph.Package{
			{ID: id, Integrity: "sha512-a"},
			{ID: id, Integrity: "sha512-b"},
		},
	}
	err := g.Validate()
	if err == nil {
		t.Fatal("expected collision error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code %s", apperr.CodeOf(err))
	}
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Subject != "peer-context" {
		t.Fatalf("subject want peer-context got %#v", err)
	}
}

func TestDanglingEdge(t *testing.T) {
	g := &graph.Graph{
		SchemaVersion: graph.SchemaVersion,
		Importers:     []graph.Importer{{ID: graph.RootImporter, Path: "."}},
		Packages:      []graph.Package{{ID: graph.PackageID{Name: "a", Version: "1.0.0"}}},
		Edges:         []graph.Edge{{From: ".", To: "missing@1.0.0", Kind: graph.DepProd}},
	}
	if err := g.Validate(); err == nil {
		t.Fatal("expected dangling edge error")
	}
}

func BenchmarkEncodeGraph(b *testing.B) {
	root := moduleRoot(b)
	data, err := os.ReadFile(filepath.Join(root, "testdata", "graph", "simple-app.json"))
	if err != nil {
		b.Fatal(err)
	}
	g, err := graph.DecodeJSON(data)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := graph.EncodeJSON(g); err != nil {
			b.Fatal(err)
		}
	}
}
