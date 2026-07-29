package pnpm_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

func TestPatchFixtureSurvivesExtraDepEncode(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "fixtures", "locks", "generated", "pnpm-9", "patch", "pnpm-lock.yaml")
	prior, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pnpm.Decode(prior)
	if err != nil {
		t.Fatal(err)
	}
	g, err := pnpm.ToGraph(doc)
	if err != nil {
		t.Fatal(err)
	}
	g.Packages = append(g.Packages, graph.Package{ID: graph.PackageID{Name: "left-pad", Version: "1.0.2"}})
	g.Edges = append(g.Edges,
		graph.Edge{From: ".", Name: "left-pad", To: "left-pad@1.0.2", Kind: graph.DepProd, Range: "1.0.2"},
	)
	det, err := lockfile.DetectPnpmWithMajor(prior, 9)
	if err != nil {
		t.Fatal(err)
	}
	det.ExplicitMajor = true
	res, err := pnpm.Adapter{}.EncodePreserving(context.Background(), path, g, prior, doc.Extensions, det)
	if err != nil {
		t.Fatal(err)
	}
	if res.Unchanged {
		t.Fatal("expected lock change")
	}
	outDoc, err := pnpm.Decode(res.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	const patchSnap = "ms@2.1.3(patch_hash=ts3vzsn6djz7ihcowyzjb4qjla)"
	if _, ok := outDoc.Snapshots[patchSnap]; !ok {
		t.Fatalf("missing patch snapshot; keys=%v", sortedSnapKeys(outDoc.Snapshots))
	}
	if _, err := pnpm.ToGraph(outDoc); err != nil {
		t.Fatal(err)
	}
}

func sortedSnapKeys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
