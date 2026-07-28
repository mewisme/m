package resolver_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/resolver"
)

func BenchmarkPeerContextResolution(b *testing.B) {
	eng, _ := engineWithPackuments(b, dualReactPluginPackuments())
	root := writeProject(b, `{
  "name": "root",
  "version": "1.0.0",
  "dependencies": {
    "host-a": "1.0.0",
    "host-b": "1.0.0"
  }
}`)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := eng.Resolve(ctx, root, resolver.ResolveOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}
