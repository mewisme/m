package resolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/testkit"
)

// BenchmarkExplainPackageLargeGraph measures ExplainPackage on the largest
// checked-in resolver fixture (registry v1 transitive graph: pkg-a → pkg-b → pkg-c
// plus lodash). Target: complete in <1s per call on CI-class hardware (GitHub
// Actions ubuntu-latest / windows-latest runners, CGO_ENABLED=0).
func BenchmarkExplainPackageLargeGraph(b *testing.B) {
	eng, _ := testEngine(b)
	root := writeProject(b, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)
	opts := resolver.ResolveOptions{Policy: &policy.Policy{StrictPeerDependencies: false}}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.ExplainPackage(ctx, root, "pkg-b", opts); err != nil {
			b.Fatal(err)
		}
	}
}

func TestExplainPackageLargeGraphUnderOneSecond(t *testing.T) {
	eng, _ := testEngine(t)
	root := writeProject(t, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": { "pkg-a": "^1.0.0", "lodash": "^4.17.0" }
}`)
	opts := resolver.ResolveOptions{Policy: &policy.Policy{StrictPeerDependencies: false}}
	start := time.Now()
	if _, err := eng.ExplainPackage(context.Background(), root, "pkg-b", opts); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("ExplainPackage took %v; target <1s on CI-class hardware", elapsed)
	}
}

// ponytail: workspace-protocol is the largest workspace graph today; registry v1
// transitive chain is the default large-graph proxy until a 1k+ package corpus lands.

func BenchmarkExplainPackageWorkspace(b *testing.B) {
	eng, _ := testEngine(b)
	projDir := b.TempDir()
	testkit.CopyFixture(b, "projects/workspace-protocol", projDir)
	opts := resolver.ResolveOptions{Policy: &policy.Policy{StrictPeerDependencies: false}}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.ExplainPackage(ctx, projDir, "lib", opts); err != nil {
			b.Fatal(err)
		}
	}
}
