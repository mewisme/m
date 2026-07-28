package lifecycle_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mewisme/m/internal/lifecycle"
	"github.com/mewisme/m/internal/process"
)

func envHost() process.EnvSource {
	return process.EnvSource{Vars: os.Environ(), Explicit: true}
}

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
		Env:         envHost(),
		Timeout:     time.Minute,
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
		Script:  script,
		Env:     envHost(),
		Timeout: time.Minute,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code == 0 {
		t.Fatalf("want non-zero exit, got %d", code)
	}
}

func TestRestrictedEnvStripsToken(t *testing.T) {
	env := process.RestrictedEnv(process.EnvSource{
		Vars:     []string{"NPM_TOKEN=secret", "HOME=/tmp", "PATH=/bin"},
		Explicit: true,
	}, "/nm/.bin")
	for _, kv := range env {
		if kv == "NPM_TOKEN=secret" {
			t.Fatal("token not stripped")
		}
	}
}

func TestDefaultCapabilitiesHonest(t *testing.T) {
	caps := lifecycle.DefaultCapabilities()
	if !caps.PackageCWD || !caps.ControlledPATH || !caps.StrippedEnv || !caps.Timeout {
		t.Fatal("expected enforced capabilities")
	}
	if caps.FilesystemIsolation || caps.NetworkIsolation {
		t.Fatal("must not claim filesystem/network isolation")
	}
}

func TestPrepareMarkerDoesNotSkipExecution(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	script := lifecycle.Script{
		PackageName: "counter",
		PackageKey:  "counter@1.0.0",
		PackageDir:  dir,
		Name:        "prepare",
		Integrity:   "sha256-deadbeef",
	}
	if runtime.GOOS == "windows" {
		script.Command = "echo ran>marker.txt"
	} else {
		script.Command = "touch marker.txt"
	}
	if err := lifecycle.MarkCacheForTest(cacheDir, script); err != nil {
		t.Fatal(err)
	}
	code, err := lifecycle.RunScript(context.Background(), process.NewExecSupervisor(), lifecycle.RunSpec{
		Script:  script,
		Env:     envHost(),
		Timeout: time.Minute,
	})
	if err != nil {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "marker.txt")); err != nil {
		t.Fatal("marker file must not skip prepare execution")
	}
}
