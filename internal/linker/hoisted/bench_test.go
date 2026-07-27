package hoisted_test

import (
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker/hoisted"
)

func BenchmarkHoistedPlan(b *testing.B) {
	g, err := graph.NewBuilder().
		Package(graph.PackageID{Name: "pkg-a", Version: "1.0.0"}, "", "").
		Package(graph.PackageID{Name: "pkg-b", Version: "1.2.0"}, "", "").
		Package(graph.PackageID{Name: "pkg-c", Version: "1.0.1"}, "", "").
		Edge(string(graph.RootImporter), "pkg-a@1.0.0", graph.DepProd, "^1.0.0").
		Edge("pkg-a@1.0.0", "pkg-b@1.2.0", graph.DepProd, "^1.0.0").
		Edge("pkg-b@1.2.0", "pkg-c@1.0.1", graph.DepProd, "^1.0.0").
		Build()
	if err != nil {
		b.Fatal(err)
	}
	nmRoot := filepath.Join(b.TempDir(), "node_modules")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := hoisted.Placements(g, nmRoot); err != nil {
			b.Fatal(err)
		}
	}
}
