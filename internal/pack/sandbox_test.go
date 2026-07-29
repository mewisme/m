package pack

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidatePackRelRejectsEscape(t *testing.T) {
	cases := []string{
		"",
		".",
		"..",
		"../secret",
		"foo/../../etc/passwd",
		"/etc/passwd",
	}
	if runtime.GOOS == "windows" {
		cases = append(cases, `C:\Windows\System32\cmd.exe`, `\\server\share\file`)
	}
	for _, rel := range cases {
		if err := validatePackRel(rel); err == nil {
			t.Fatalf("validatePackRel(%q) expected error", rel)
		}
	}
}

func TestPackRejectsSymlinkFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink("real.txt", link); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Pack(t.Context(), Options{Root: root, PackDestination: t.TempDir()})
	if err == nil {
		t.Fatal("expected pack failure for symlink file")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "reparse") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackRejectsWhitelistEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0","files":["../outside"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ListFiles(root, mustRead(t, filepath.Join(root, "package.json")))
	if err == nil {
		t.Fatal("expected list failure for escape path in files field")
	}
}

func TestPackPreservesExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix executable bits are not preserved on Windows")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "run")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0","files":["bin/run"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	res, err := Pack(t.Context(), Options{Root: root, PackDestination: dest})
	if err != nil {
		t.Fatal(err)
	}
	mode, err := tarEntryMode(res.TarballPath, "package/bin/run")
	if err != nil {
		t.Fatal(err)
	}
	if mode&0o111 == 0 {
		t.Fatalf("expected executable tar mode, got %#o", mode)
	}
}

func TestPackExcludesOutputTarballInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Pack(t.Context(), Options{Root: root, PackDestination: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range res.Files {
		if strings.HasSuffix(rel, ".tgz") {
			t.Fatalf("packed output tarball path %q", rel)
		}
	}
}

func TestPackRejectsOversizeFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"p","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(root, "big.bin")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	chunk := make([]byte, 1<<20)
	if _, err := f.Write(chunk); err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxPackFileBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = Pack(t.Context(), Options{Root: root, PackDestination: t.TempDir()})
	if err == nil {
		t.Fatal("expected oversize rejection")
	}
}

func tarEntryMode(tgzPath, wantName string) (int64, error) {
	f, err := os.Open(tgzPath)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if hdr.Name == wantName {
			return hdr.Mode, nil
		}
	}
	return 0, os.ErrNotExist
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
