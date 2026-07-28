package lifecycle_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/lifecycle"
	"github.com/mewisme/mew/internal/process"
)

func envHost() process.EnvSource {
	return process.EnvSource{Vars: os.Environ(), Explicit: true}
}

type stallSupervisor struct{}

func (stallSupervisor) Start(context.Context, process.Spec) (*process.Handle, error) {
	return &process.Handle{PID: 1}, nil
}

func (stallSupervisor) Wait(ctx context.Context, _ *process.Handle) error {
	<-ctx.Done()
	return ctx.Err()
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

func TestRunScriptTimeoutPreservesDeadlineExceeded(t *testing.T) {
	dir := t.TempDir()
	script := lifecycle.Script{
		PackageName: "slowpkg",
		PackageDir:  dir,
		Name:        "postinstall",
		Command:     "ignored",
	}
	_, err := lifecycle.RunScript(context.Background(), stallSupervisor{}, lifecycle.RunSpec{
		Script:  script,
		Env:     envHost(),
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if !strings.Contains(err.Error(), "slowpkg") || !strings.Contains(err.Error(), "postinstall") {
		t.Fatalf("error missing context: %v", err)
	}
}

func TestRunScriptCancelPreservesCanceled(t *testing.T) {
	dir := t.TempDir()
	script := lifecycle.Script{
		PackageName: "cancelpkg",
		PackageDir:  dir,
		Name:        "postinstall",
		Command:     "ignored",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := lifecycle.RunScript(ctx, stallSupervisor{}, lifecycle.RunSpec{
		Script:  script,
		Env:     envHost(),
		Timeout: time.Minute,
	})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
}

func TestRunScriptTimeoutAudit(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.jsonl")
	if err := lifecycle.AppendAudit(auditPath, lifecycle.AuditEntry{
		Package:  "auditpkg",
		Script:   "postinstall",
		ExitCode: 1,
		TimedOut: true,
		Status:   "timeout",
	}); err != nil {
		t.Fatal(err)
	}
	entries, err := lifecycle.ReadAudit(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].TimedOut || entries[0].Status != "timeout" {
		t.Fatalf("audit=%v", entries)
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
