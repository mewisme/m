package binresolve

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/linker"
)

func writeFixtureBin(t *testing.T, root, cmd string) {
	t.Helper()
	nm := filepath.Join(root, "node_modules")
	binDir := filepath.Join(nm, ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(nm, "pkg-a")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(pkgDir, "cli.js")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(binDir, cmd)
	if runtime.GOOS == "windows" {
		shim = shim + ".cmd"
		if err := os.WriteFile(shim, []byte("@echo off\nnode ..\\pkg-a\\cli.js %*\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := os.WriteFile(shim, []byte("#!/bin/sh\nexec node ../pkg-a/cli.js \"$@\"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := binmeta.Publish(binmeta.PublishInput{
		NodeModules: nm, GenerationID: "g1", ImporterIdentity: ".", LayoutMode: binmeta.LayoutHoisted,
		Sources: []linker.BinSource{{Cmd: cmd, Target: "cli.js", PackageDir: pkgDir}},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveVerifiedHit(t *testing.T) {
	root := t.TempDir()
	writeFixtureBin(t, root, "hello")
	res, err := Resolve(Options{ProjectRoot: root, PackageDir: root, ImporterRel: ".", Command: "hello", AllowUnowned: true, RequireVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "hello" || !res.OwnershipVerified {
		t.Fatalf("%+v", res)
	}
}

func TestResolveAmbiguity(t *testing.T) {
	root := t.TempDir()
	writeFixtureBin(t, root, "dup")
	nm := filepath.Join(root, "node_modules")
	doc, _ := binmeta.Read(nm)
	doc.Records = append(doc.Records, binmeta.Record{
		DependencyName: "pkg-b", ResolvedPackage: "pkg-b", PackageDir: filepath.Join(nm, "pkg-b"),
		DeclaredBin: "dup", MaterializedShim: filepath.Join(nm, ".bin", "dup"), OwnershipVerified: true,
	})
	_ = binmeta.Write(nm, doc)
	_, err := Resolve(Options{ProjectRoot: root, PackageDir: root, ImporterRel: ".", Command: "dup", RequireVerified: true})
	if apperr.CodeOf(err) != apperr.Usage {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveMiss(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "node_modules"), 0o755)
	_, err := Resolve(Options{ProjectRoot: root, PackageDir: root, ImporterRel: ".", Command: "missing"})
	if apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("err=%v", err)
	}
}

func TestComSpecRelativeRejected(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	_, err := validateComSpec("relative\\cmd.exe")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPnPRecursionGuard(t *testing.T) {
	if PnPRecursionActive() {
		t.Fatal("should start false")
	}
}
