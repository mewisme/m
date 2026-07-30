package dlx

import "testing"

func BenchmarkRequestIdentityDigest(b *testing.B) {
	id := RequestIdentity{NormalizedSpecs: []string{"vite", "typescript"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = id.Digest()
	}
}
