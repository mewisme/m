package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

func BenchmarkResolveWorkspaceProtocol(b *testing.B) {
	eng, _ := testEngine(b)
	projDir := b.TempDir()
	testkit.CopyFixture(b, "projects/workspace-protocol", projDir)
	opts := resolver.ResolveOptions{Policy: &policy.Policy{StrictPeerDependencies: false}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Resolve(context.Background(), projDir, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// ponytail: no large monorepo fixture yet; workspace-protocol is the largest checked-in
// resolver graph. Revisit when fixtures/projects gains a 1k+ package workspace corpus.

func BenchmarkTargetedIncrementalUpdate(b *testing.B) {
	eng, _ := testEngine(b)
	root := writeProject(b, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)
	prior, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "sha256-a", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.0.0"}, "sha256-b", "").
		Package(graph.PackageID{Name: "lodash", Version: "4.17.21"}, "sha256-l", "").
		EdgeEx(string(graph.RootImporter), "pkg-a", "pkg-a@1.0.0", graph.DepProd, "^1.0.0", false).
		EdgeEx(string(graph.RootImporter), "lodash", "lodash@4.17.21", graph.DepProd, "^4.17.0", false).
		EdgeEx("pkg-a@1.0.0", "pkg-b", "pkg-b@1.0.0", graph.DepProd, "^1.0.0", false).
		Build()
	if err != nil {
		b.Fatal(err)
	}
	opts := resolver.ResolveOptions{
		Prior: prior, Hints: prior,
		UpdateTargets: []string{"pkg-a"}, IncrementalUpdate: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Resolve(context.Background(), root, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeGraphResolve(b *testing.B) {
	eng, _ := testEngine(b)
	root := writeProject(b, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)
	opts := resolver.ResolveOptions{Policy: &policy.Policy{StrictPeerDependencies: false}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Resolve(context.Background(), root, opts); err != nil {
			b.Fatal(err)
		}
	}
}
