package lockfile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/lockfile"
)

func moduleRoot(t *testing.T) string {
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

func TestLossReportGolden(t *testing.T) {
	path := filepath.Join(moduleRoot(t), "testdata", "graph", "loss-report.json")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := lockfile.DecodeLossJSON(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := lockfile.EncodeLossJSON(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}
