package pnpm_test

import (
	"testing"

	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
)

func TestSnapshotTopologyTransitive(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      a:
        specifier: ^1.0.0
        version: 1.0.0
packages:
  a@1.0.0:
    resolution: {integrity: sha512-a}
  b@1.0.0:
    resolution: {integrity: sha512-b}
snapshots:
  a@1.0.0:
    dependencies:
      b: b@1.0.0
  b@1.0.0: {}
`
	doc, err := pnpm.Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(g, ".", "a", "a@1.0.0", graph.DepProd) {
		t.Fatal("missing root→a edge")
	}
	if !hasEdge(g, "a@1.0.0", "b", "b@1.0.0", graph.DepProd) {
		t.Fatal("missing snapshot a→b edge")
	}
}

func TestSnapshotOptionalAndPeerEdges(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .:
    optionalDependencies:
      opt:
        specifier: ^1.0.0
        version: 1.0.0
packages:
  root-pkg@1.0.0:
    resolution: {integrity: sha512-root}
  opt@1.0.0:
    resolution: {integrity: sha512-opt}
  peer@2.0.0:
    resolution: {integrity: sha512-peer}
snapshots:
  root-pkg@1.0.0:
    optionalDependencies:
      opt: opt@1.0.0
    peerDependencies:
      peer: peer@2.0.0
  opt@1.0.0: {}
  peer@2.0.0: {}
`
	doc, err := pnpm.Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(g, ".", "opt", "opt@1.0.0", graph.DepOptional) {
		t.Fatal("missing optional importer edge")
	}
	if !hasEdge(g, "root-pkg@1.0.0", "opt", "opt@1.0.0", graph.DepOptional) {
		t.Fatal("missing optional snapshot edge")
	}
	if !hasEdge(g, "root-pkg@1.0.0", "peer", "peer@2.0.0", graph.DepPeer) {
		t.Fatal("missing peer snapshot edge")
	}
}

func TestSnapshotDanglingTargetFails(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
importers:
  .: {}
packages:
  a@1.0.0:
    resolution: {integrity: sha512-a}
snapshots:
  a@1.0.0:
    dependencies:
      missing: ghost@9.9.9
`
	doc, err := pnpm.Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pnpm.ToGraph(doc)
	if err == nil {
		t.Fatal("expected dangling snapshot target error")
	}
}

func hasEdge(g *graph.Graph, from, name, to string, kind graph.DepKind) bool {
	for _, e := range g.Edges {
		if e.From == from && e.Name == name && e.To == to && e.Kind == kind {
			return true
		}
	}
	return false
}
