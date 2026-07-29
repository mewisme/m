package npm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/npm"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/testkit"
)

func moduleRoot(t testing.TB) string {
	t.Helper()
	return testkit.ModuleRoot(t)
}

func TestDecodeRejectV1(t *testing.T) {
	raw := []byte(`{"lockfileVersion":1,"dependencies":{"lodash":{"version":"4.17.21"}}}`)
	_, err := npm.Decode(raw)
	if err == nil {
		t.Fatal("expected v1 rejection")
	}
}

func TestDecodeRejectUnknownMajor(t *testing.T) {
	raw := []byte(`{"lockfileVersion":4,"packages":{}}`)
	_, err := npm.Decode(raw)
	if err == nil {
		t.Fatal("expected unsupported major rejection")
	}
}

func TestFixtureV2BasicToGraph(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "fixtures", "locks", "npm", "v2-basic")
	data, err := os.ReadFile(filepath.Join(dir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := npm.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := npm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) != 1 {
		t.Fatalf("packages=%d want 1", len(g.Packages))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("edges=%d want 1", len(g.Edges))
	}
}

func TestFixtureV3WorkspacesToGraph(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "fixtures", "locks", "npm", "v3-workspaces")
	data, err := os.ReadFile(filepath.Join(dir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := npm.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := npm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Importers) < 2 {
		t.Fatalf("importers=%d want >=2", len(g.Importers))
	}
}

func TestRoundTripSemantic(t *testing.T) {
	cases := []string{"v2-basic", "v3-workspaces"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(moduleRoot(t), "testdata", "lockfile", "npm-roundtrip", name, "package-lock.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := npm.Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			g, err := npm.ToGraph(doc)
			if err != nil {
				t.Fatal(err)
			}
			outDoc, err := npm.FromGraph(g, doc, npm.DetectFromDocument(doc))
			if err != nil {
				t.Fatal(err)
			}
			out, err := npm.Encode(outDoc)
			if err != nil {
				t.Fatal(err)
			}
			redoc, err := npm.Decode(out)
			if err != nil {
				t.Fatal(err)
			}
			g2, err := npm.ToGraph(redoc)
			if err != nil {
				t.Fatal(err)
			}
			eq, err := lockfile.GraphsEqual(g, g2)
			if err != nil {
				t.Fatal(err)
			}
			if !eq {
				t.Fatal("semantic round-trip graph mismatch")
			}
		})
	}
}

func TestEncodePreservingNoOp(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "fixtures", "locks", "npm", "v2-basic", "package-lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := npm.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	g, err := npm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	res, err := npm.Adapter{}.EncodePreserving(t.Context(), path, g, data, nil, npm.DetectFromDocument(doc))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Unchanged {
		t.Fatal("expected unchanged encode for identical graph")
	}
}
