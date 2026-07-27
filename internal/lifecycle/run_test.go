package lifecycle_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mewisme/m/internal/lifecycle"
	"github.com/mewisme/m/internal/process"
)

func TestRunScriptBenign(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		bat := filepath.Join(dir, "run.cmd")
		if err := os.WriteFile(bat, []byte("@echo ok>marker.txt\r\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	script := lifecycle.Script{
		PackageName: "demo",
		PackageDir:  dir,
		Name:        "postinstall",
	}
	if runtime.GOOS == "windows" {
		script.Command = "run.cmd"
	} else {
		script.Command = "touch marker.txt"
	}
	code, err := lifecycle.RunScript(context.Background(), process.NewExecSupervisor(), lifecycle.RunSpec{
		Script:      script,
		NodeModules: filepath.Join(dir, "node_modules"),
		Env:         os.Environ(),
	})
	if err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "marker.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestRunScriptFailure(t *testing.T) {
	dir := t.TempDir()
	script := lifecycle.Script{
		PackageName: "demo",
		PackageDir:  dir,
		Name:        "postinstall",
	}
	if runtime.GOOS == "windows" {
		script.Command = "exit 1"
	} else {
		script.Command = "false"
	}
	code, err := lifecycle.RunScript(context.Background(), process.NewExecSupervisor(), lifecycle.RunSpec{
		Script: script,
		Env:    os.Environ(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code == 0 {
		t.Fatalf("want non-zero exit, got %d", code)
	}
}

func TestRestrictedEnvStripsToken(t *testing.T) {
	env := process.RestrictedEnv([]string{"NPM_TOKEN=secret", "HOME=/tmp", "PATH=/bin"}, "/nm/.bin")
	for _, kv := range env {
		if kv == "NPM_TOKEN=secret" {
			t.Fatal("token not stripped")
		}
	}
}
