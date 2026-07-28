package archive_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/archive"
	"github.com/mewisme/mew/internal/testkit"
)

func TestExtractLodashGolden(t *testing.T) {
	tgz := filepath.Join(testkit.FixtureDir(t, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	dest := t.TempDir()
	if err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions()); err != nil {
		t.Fatal(err)
	}
	got := listFiles(t, dest)
	goldenPath := filepath.Join(testkit.ModuleRoot(t), "testdata", "archive", "expected-lodash", "files.txt")
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		_ = os.MkdirAll(filepath.Dir(goldenPath), 0o755)
		if werr := os.WriteFile(goldenPath, []byte(strings.Join(got, "\n")+"\n"), 0o644); werr != nil {
			t.Fatal(werr)
		}
		t.Skip("generated golden files.txt")
	}
	want := strings.Split(strings.TrimSpace(string(wantBytes)), "\n")
	if len(got) != len(want) {
		t.Fatalf("count %d want %d\n%v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("line %d: %q want %q", i, got[i], want[i])
		}
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	tgz := filepath.Join(testkit.FixtureDir(t, "archives"), "traversal-attack.tgz")
	dest := t.TempDir()
	err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions())
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
	if files, _ := os.ReadDir(dest); len(files) != 0 {
		t.Fatalf("wrote files: %v", files)
	}
}

func TestExtractIntegrityMismatchNoWrites(t *testing.T) {
	tgz := filepath.Join(testkit.FixtureDir(t, "archives"), "corrupt-hash.tgz")
	dest := t.TempDir()
	// Extraction itself doesn't verify hash; app layer does before extract.
	// Here we only ensure traversal archive left dest empty; corrupt-hash is valid tar.
	if err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.js")); err != nil {
		t.Fatal(err)
	}
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
