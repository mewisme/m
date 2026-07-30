package binresolve

import "testing"

func BenchmarkCheapVerifiedMiss(b *testing.B) {
	root := b.TempDir()
	opts := Options{ProjectRoot: root, PackageDir: root, ImporterRel: ".", Command: "definitely-missing-bin"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = CheapVerifiedHint(opts)
	}
}
