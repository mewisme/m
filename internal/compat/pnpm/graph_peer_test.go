package pnpm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/graph"
)

func peerFixtureRoot(t *testing.T) string {
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

func TestPeerContextFixtureGraph(t *testing.T) {
	root := peerFixtureRoot(t)
	cases := []struct {
		major    int
		acornVer string
	}{
		{9, "8.18.0"},
		{10, "8.18.0"},
		{11, "8.17.0"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("pnpm-%d-peer-context", tc.major), func(t *testing.T) {
			path := filepath.Join(root, "fixtures", "locks", "generated",
				fmt.Sprintf("pnpm-%d", tc.major), "peer-context", "pnpm-lock.yaml")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			g, err := ToGraph(doc)
			if err != nil {
				t.Fatal(err)
			}

			peerInstance := "acorn-jsx@5.3.2#acorn@" + tc.acornVer
			acornKey := "acorn@" + tc.acornVer

			if !hasPackage(g, peerInstance) {
				t.Fatalf("missing peer-context package %q", peerInstance)
			}
			if !hasPackage(g, acornKey) {
				t.Fatalf("missing acorn package %q", acornKey)
			}
			if !hasEdge(g, ".", "acorn-jsx", peerInstance, graph.DepProd) {
				t.Fatal("missing importer→peer-instance edge")
			}
			if !hasEdge(g, peerInstance, "acorn", acornKey, graph.DepProd) {
				t.Fatal("missing snapshot peer-instance→acorn edge")
			}
		})
	}
}

func TestPeerContextMultiPeerInstance(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      react-dom:
        specifier: 18.2.0
        version: 18.2.0(react@18.2.0,scheduler@0.23.0)
packages:
  react-dom@18.2.0:
    resolution: {integrity: sha512-dom}
  react@18.2.0:
    resolution: {integrity: sha512-react}
  scheduler@0.23.0:
    resolution: {integrity: sha512-sched}
snapshots:
  react-dom@18.2.0(react@18.2.0,scheduler@0.23.0): {}
  react@18.2.0: {}
  scheduler@0.23.0: {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "react-dom@18.2.0#react@18.2.0,scheduler@0.23.0"
	if !hasPackage(g, want) {
		t.Fatalf("missing multi-peer instance %q", want)
	}
}

func TestPeerContextScopedInstance(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      '@scope/pkg':
        specifier: 1.0.0
        version: 1.0.0(peer@2.0.0)
packages:
  '@scope/pkg@1.0.0':
    resolution: {integrity: sha512-pkg}
  peer@2.0.0:
    resolution: {integrity: sha512-peer}
snapshots:
  '@scope/pkg@1.0.0(peer@2.0.0)': {}
  peer@2.0.0: {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "@scope/pkg@1.0.0#peer@2.0.0"
	if !hasPackage(g, want) {
		t.Fatalf("missing scoped peer instance %q", want)
	}
}

func TestMissingBaseMetadataFails(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .: {}
packages:
  base@1.0.0:
    resolution: {integrity: sha512-base}
snapshots:
  orphan@9.9.9(peer@1.0.0): {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ToGraph(doc)
	if err == nil {
		t.Fatal("expected missing base metadata error")
	}
}

func TestNoPhantomBaseInstance(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      acorn-jsx:
        specifier: 5.3.2
        version: 5.3.2(acorn@8.18.0)
packages:
  acorn-jsx@5.3.2:
    resolution: {integrity: sha512-jsx}
  acorn@8.18.0:
    resolution: {integrity: sha512-acorn}
snapshots:
  acorn-jsx@5.3.2(acorn@8.18.0):
    dependencies:
      acorn: 8.18.0
  acorn@8.18.0: {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if hasPackage(g, "acorn-jsx@5.3.2") {
		t.Fatal("phantom base instance acorn-jsx@5.3.2 must not appear in graph")
	}
	if !hasPackage(g, "acorn-jsx@5.3.2#acorn@8.18.0") {
		t.Fatal("missing peer-context instance")
	}
}

func TestPeerContextRoundTripDeterministic(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .: {}
packages:
  z@1.0.0:
    resolution: {integrity: sha512-z}
  a@1.0.0:
    resolution: {integrity: sha512-a}
snapshots:
  z@1.0.0: {}
  a@1.0.0: {}
  z@1.0.0(peer@2.0.0): {}
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Packages) != 3 {
		t.Fatalf("packages=%d want 3", len(g.Packages))
	}
	prev := ""
	for _, p := range g.Packages {
		key := p.ID.Key()
		if key < prev {
			t.Fatalf("packages not sorted: %q after %q", key, prev)
		}
		prev = key
	}
}

func hasPackage(g *graph.Graph, key string) bool {
	for _, p := range g.Packages {
		if p.ID.Key() == key {
			return true
		}
	}
	return false
}

func hasEdge(g *graph.Graph, from, name, to string, kind graph.DepKind) bool {
	for _, e := range g.Edges {
		if e.From == from && e.Name == name && e.To == to && e.Kind == kind {
			return true
		}
	}
	return false
}
