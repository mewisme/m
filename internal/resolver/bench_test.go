package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/policy"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/testkit"
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
