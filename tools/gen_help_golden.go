//go:build ignore

package main

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/cli"
)

func main() {
	outDir := filepath.Join("testdata", "cli", "help-golden")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	info := cli.BuildInfo{Version: "0.0.0-test", Commit: "deadbeef", BuildDate: "2026-01-01"}

	root := cli.NewMRoot(info)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "m-root.txt"), bytes.ReplaceAll(buf.Bytes(), []byte("\r\n"), []byte("\n")), 0o644); err != nil {
		panic(err)
	}

	root2 := cli.NewMXRoot(info)
	buf.Reset()
	root2.SetOut(&buf)
	root2.SetErr(&buf)
	root2.SetArgs([]string{"--help"})
	if err := root2.Execute(); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "mx-root.txt"), bytes.ReplaceAll(buf.Bytes(), []byte("\r\n"), []byte("\n")), 0o644); err != nil {
		panic(err)
	}
}
