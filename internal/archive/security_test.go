package archive_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/archive"
)

func TestExtractRejectsWindowsReservedName(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows reserved name enforcement is platform-specific")
	}
	tgz := writeMiniTar(t, []tarEntry{{name: "package/CON", typ: tar.TypeReg, body: "x"}})
	dest := t.TempDir()
	err := archive.Extract(context.Background(), tgz, dest, archive.DefaultOptions())
	if apperr.CodeOf(err) != apperr.Integrity {
		t.Fatalf("got %v", err)
	}
}

type tarEntry struct {
	name string
	typ  byte
	body string
}

func writeMiniTar(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o644, Size: int64(len(e.body)), Typeflag: e.typ}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mini.tgz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
