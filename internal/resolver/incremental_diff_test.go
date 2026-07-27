package resolver_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/resolver"
)

func TestIncrementalGraphDiffGolden(t *testing.T) {
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

	assertSubgraphGolden(t, "incremental-lodash-unchanged.json", prior, res.Graph, "lodash@4.17.21")
}

func assertSubgraphGolden(t testing.TB, name string, prior, resolved *graph.Graph, pkgKey string) {
	t.Helper()
	priorSub, err := resolver.ExtractPackageSubgraph(prior, pkgKey)
	if err != nil {
		t.Fatal(err)
	}
	gotSub, err := resolver.ExtractPackageSubgraph(resolved, pkgKey)
	if err != nil {
		t.Fatal(err)
	}
	got, err := graph.EncodeJSON(gotSub)
	if err != nil {
		t.Fatal(err)
	}
	want, err := graph.EncodeJSON(priorSub)
	if err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join("..", "..", "testdata", "resolver", "golden", "graphs", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, stripTarballHosts(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s (set UPDATE_GOLDEN=1 to create): %v", name, err)
	}
	gotNorm := stripTarballHosts(got)
	wantNorm := stripTarballHosts(want)
	if !bytes.Equal(gotNorm, wantNorm) {
		t.Fatalf("subgraph %s differs from prior\n--- got ---\n%s\n--- prior ---\n%s", pkgKey, gotNorm, wantNorm)
	}
	if !bytes.Equal(gotNorm, stripTarballHosts(golden)) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, gotNorm, stripTarballHosts(golden))
	}
}

func TestIncrementalAliasStability(t *testing.T) {
	eng, _ := engineWithPackuments(t, aliasPackuments())
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "foo": "npm:bar@^1.0.0"
  }
}`)
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "bar", Version: "1.0.0"}, "sha512-bar", "").
		EdgeEx(string(graph.RootImporter), "foo", "bar@1.0.0", graph.DepProd, "npm:bar@^1.0.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Prior: prior, Hints: prior,
		UpdateTargets: []string{"foo"}, IncrementalUpdate: true,
		PriorFingerprints: &resolver.PriorFingerprints{
			OverridesFingerprint:      resolver.OverridesFingerprint(nil),
			ResolverPolicyFingerprint: resolver.PolicyFingerprint(resolver.PolicyFromEffective(eng.Effective)),
			TargetPlatformFingerprint: resolver.TargetPlatformFingerprint(resolver.CurrentTarget()),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var aliasEdge graph.Edge
	for _, e := range res.Graph.Edges {
		if e.Name == "foo" {
			aliasEdge = e
		}
	}
	if aliasEdge.To != "bar@1.0.0" || aliasEdge.Name != "foo" {
		t.Fatalf("alias edge=%#v", aliasEdge)
	}
}

func TestIncrementalPolicyDriftReResolves(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "lodash": "^4.17.0" }
}`)
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	strict := resolver.PolicyFromEffective(eng.Effective)
	loose := &policy.Policy{StrictPeerDependencies: false}
	res, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Prior: prior, Hints: prior,
		IncrementalUpdate: true,
		Policy:            loose,
		PriorFingerprints: &resolver.PriorFingerprints{
			ResolverPolicyFingerprint: resolver.PolicyFingerprint(strict),
			TargetPlatformFingerprint: resolver.TargetPlatformFingerprint(resolver.CurrentTarget()),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Graph.Packages {
		if p.ID.Name == "lodash" && p.ID.Version == "4.17.21" {
			return
		}
	}
	t.Fatal("policy drift should still resolve lodash")
}
