package linker_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/manifest"
)

func TestWriteBinsUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix shim test")
	}
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	pkgDir := filepath.Join(nm, "pkg-cli")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte("// cli"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := linker.WriteBins(nm, []linker.BinSource{{
		Cmd:        "cli",
		Target:     "./cli.js",
		PackageDir: pkgDir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(nm, ".bin", "cli"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.HasPrefix(body, "#!/bin/sh") {
		t.Fatalf("want shell script, got %q", body)
	}
	if !strings.Contains(body, "../pkg-cli/cli.js") {
		t.Fatalf("relative target missing: %q", body)
	}
}

func TestWriteBinsWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows shim test")
	}
	root := t.TempDir()
	nm := filepath.Join(root, "node_modules")
	pkgDir := filepath.Join(nm, "pkg-cli")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte("// cli"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := linker.WriteBins(nm, []linker.BinSource{{
		Cmd:        "cli",
		Target:     "./cli.js",
		PackageDir: pkgDir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(nm, ".bin", "cli.cmd"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "node") || !strings.Contains(body, "cli.js") {
		t.Fatalf("cmd wrapper: %q", body)
	}
}

func TestBinCommandsStringForm(t *testing.T) {
	doc := &manifest.Document{
		Name: "my-tool",
		Bin:  []byte(`"./bin.js"`),
	}
	cmds, err := linker.BinCommands(doc)
	if err != nil {
		t.Fatal(err)
	}
	if cmds["my-tool"] != "./bin.js" {
		t.Fatalf("got %#v", cmds)
	}
}

func TestWriteBinsDuplicateCmd(t *testing.T) {
	nm := filepath.Join(t.TempDir(), "node_modules")
	err := linker.WriteBins(nm, []linker.BinSource{
		{Cmd: "cli", Target: "./a.js", PackageDir: nm},
		{Cmd: "cli", Target: "./b.js", PackageDir: nm},
	})
	if err == nil {
		t.Fatal("expected duplicate bin error")
	}
}
