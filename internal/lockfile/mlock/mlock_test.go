package mlock_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/lockfile/mlock"
)

func moduleRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func loadGraph(t testing.TB, name string) *graph.Graph {
	t.Helper()
	root := moduleRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "testdata", "graph", name))
	if err != nil {
		t.Fatal(err)
	}
	g, err := graph.DecodeJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func specifiersFromEdges(g *graph.Graph) map[graph.ImporterID][]mlock.Specifier {
	return mlock.SpecifiersFromGraph(g)
}

func TestGoldenRoundTrip(t *testing.T) {
	root := moduleRoot(t)
	cases := []struct {
		graph    string
		golden   string
		settings mlock.Settings
	}{
		{"simple-app.json", "basic", mlock.DefaultSettings()},
		{"workspace.json", "workspace", mlock.DefaultSettings()},
	}
	for _, tc := range cases {
		t.Run(tc.golden, func(t *testing.T) {
			g := loadGraph(t, tc.graph)
			specs := specifiersFromEdges(g)
			doc, err := mlock.FromGraph(g, specs, tc.settings)
			if err != nil {
				t.Fatal(err)
			}
			got, err := mlock.Encode(doc)
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join(root, "testdata", "lockfile", "mlock", "golden", tc.golden, "m.lock")
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.Getenv("UPDATE_GOLDEN") == "1" {
					if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
						t.Fatal(err)
					}
					t.Logf("updated golden %s", goldenPath)
					return
				}
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
			back, err := mlock.Decode(got)
			if err != nil {
				t.Fatal(err)
			}
			g2, err := mlock.ToGraph(back)
			if err != nil {
				t.Fatal(err)
			}
			reenc, err := graph.EncodeJSON(g2)
			if err != nil {
				t.Fatal(err)
			}
			orig, err := graph.EncodeJSON(g)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(reenc, orig) {
				t.Fatalf("graph round-trip mismatch\n%s\nvs\n%s", reenc, orig)
			}
		})
	}
}

func TestChecksumMismatch(t *testing.T) {
	root := moduleRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "testdata", "lockfile", "mlock", "corrupt", "bad-checksum.m.lock"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = mlock.Decode(data)
	if err == nil {
		t.Fatal("expected checksum error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestDuplicatePackageRejected(t *testing.T) {
	doc := &mlock.Document{
		LockfileVersion: mlock.LockfileVersion,
		Settings:        mlock.DefaultSettings(),
		Importers:       []mlock.ImporterSection{{ID: graph.RootImporter, Path: "."}},
		Packages: []graph.Package{
			{ID: graph.PackageID{Name: "a", Version: "1.0.0"}},
			{ID: graph.PackageID{Name: "a", Version: "1.0.0"}},
		},
	}
	_, err := mlock.Encode(doc)
	if err == nil {
		t.Fatal("expected duplicate package error")
	}
}

func TestMigrateUnknownVersion(t *testing.T) {
	doc := &mlock.Document{LockfileVersion: 99}
	err := mlock.Migrate(doc)
	if err == nil {
		t.Fatal("expected migrate error")
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestFrozenDrift(t *testing.T) {
	lock := []mlock.ImporterSection{{
		ID: graph.RootImporter,
		Specifiers: []mlock.Specifier{
			{Name: "lodash", Range: "^4.17.21", Kind: graph.DepProd},
		},
	}}
	manifest := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: {
			{Name: "lodash", Range: "^4.18.0", Kind: graph.DepProd},
		},
	}
	drift := mlock.CompareSpecifiers(lock, manifest)
	if len(drift) != 1 || drift[0].Kind != mlock.DriftChanged {
		t.Fatalf("drift=%v", drift)
	}
}

func TestCorruptFixtures(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "testdata", "lockfile", "mlock", "corrupt")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			_, err = mlock.Decode(data)
			if err == nil {
				t.Fatal("expected decode error")
			}
			if apperr.CodeOf(err) != apperr.Lockfile && apperr.CodeOf(err) != apperr.IO {
				t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	g := loadGraph(b, "simple-app.json")
	doc, err := mlock.FromGraph(g, mlock.SpecifiersFromGraph(g), mlock.DefaultSettings())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mlock.Encode(doc); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode(b *testing.B) {
	g := loadGraph(b, "simple-app.json")
	doc, err := mlock.FromGraph(g, mlock.SpecifiersFromGraph(g), mlock.DefaultSettings())
	if err != nil {
		b.Fatal(err)
	}
	data, err := mlock.Encode(doc)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mlock.Decode(data); err != nil {
			b.Fatal(err)
		}
	}
}
