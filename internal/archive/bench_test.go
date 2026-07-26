package archive_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/archive"
	"github.com/mewisme/m/internal/testkit"
)

func BenchmarkExtract(b *testing.B) {
	tgz := filepath.Join(testkit.FixtureDir(b, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dest := b.TempDir()
		if err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions()); err != nil {
			b.Fatal(err)
		}
	}
}
