package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListFilesMinimalFixture(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "pack", "minimal-package")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(abs, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ListFiles(abs, data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"index.js", "lib/util.js", "package.json"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestTarballFileNameScoped(t *testing.T) {
	if got := TarballFileName("@scope/pkg", "1.0.0"); got != "scope-pkg-1.0.0.tgz" {
		t.Fatalf("%q", got)
	}
}

func TestPackDeterministicHash(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "pack", "minimal-package")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	r1, err := Pack(t.Context(), Options{Root: abs, PackDestination: dest})
	if err != nil {
		t.Fatal(err)
	}
	b1, err := os.ReadFile(r1.TarballPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(r1.TarballPath)
	r2, err := Pack(t.Context(), Options{Root: abs, PackDestination: dest})
	if err != nil {
		t.Fatal(err)
	}
	b2, err := os.ReadFile(r2.TarballPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatal("tarball bytes differ between runs")
	}
}
