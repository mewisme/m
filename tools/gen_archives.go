//go:build ignore

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen_archives <repo-root>")
		os.Exit(2)
	}
	root := os.Args[1]
	archDir := filepath.Join(root, "fixtures", "archives")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		panic(err)
	}
	if err := writeTraversal(filepath.Join(archDir, "traversal-attack.tgz")); err != nil {
		panic(err)
	}
	if err := writeMinimal(filepath.Join(archDir, "corrupt-hash.tgz")); err != nil {
		panic(err)
	}
	fmt.Println("wrote fixtures/archives")
}

func writeTraversal(path string) error {
	members := []struct{ name, body string }{
		{"../../etc/passwd", "evil"},
		{"package/../../../tmp/evil", "evil"},
		{`C:\Windows\System32\evil.dll`, "evil"},
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, m := range members {
		if err := tw.WriteHeader(&tar.Header{
			Name: m.name, Mode: 0o644, Size: int64(len(m.body)), Typeflag: tar.TypeReg,
		}); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(m.body)); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func writeMinimal(path string) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := "valid but wrong hash"
	if err := tw.WriteHeader(&tar.Header{
		Name: "package/index.js", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
	}); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
