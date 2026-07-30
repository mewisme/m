package binresolve

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/binmeta"
)

func TestBuildLaunchNodeShebang(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix launch")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "run.js")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec, err := BuildLaunchSpec(binmeta.BinCandidate{ShimPath: script, TargetPath: script, PackageDir: dir, Command: "run"}, []string{"--help"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != LaunchNode || spec.Program == "" {
		t.Fatalf("%+v", spec)
	}
}

func TestBuildLaunchWindowsCmd(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	dir := t.TempDir()
	shim := filepath.Join(dir, "tool.cmd")
	if err := os.WriteFile(shim, []byte("@echo off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := BuildLaunchSpec(binmeta.BinCandidate{ShimPath: shim, PackageDir: dir, Command: "tool"}, nil, []string{"ComSpec=C:\\Windows\\System32\\cmd.exe"}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != LaunchCmd {
		t.Fatalf("%+v", spec)
	}
}
