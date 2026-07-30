package runner_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runner"
)

func TestLaunchDirectArgv(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	_, err := r.Run(t.Context(), runner.RunOptions{
		ProjectRoot: "/proj",
		PackageDir:  "/proj",
		Scripts:     map[string]string{"dev": "node script.js"},
		Selector:    "dev",
		HostEnv:     []string{"PATH=/bin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 1 || rec.specs[0].Path != "node script.js" {
		t.Fatalf("specs=%+v", rec.specs)
	}
}

func TestLaunchEmptyArgs(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	_, err := r.Run(t.Context(), runner.RunOptions{
		ProjectRoot: "/proj",
		PackageDir:  "/proj",
		Scripts:     map[string]string{"dev": "node"},
		Selector:    "dev",
		HostEnv:     []string{"PATH=/bin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs[0].Args) != 0 {
		t.Fatalf("args=%v", rec.specs[0].Args)
	}
}

func TestLaunchQuotedArgs(t *testing.T) {
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	_, err := r.Run(t.Context(), runner.RunOptions{
		ProjectRoot:   "/proj",
		PackageDir:    "/proj",
		Scripts:       map[string]string{"dev": "node script.js"},
		Selector:      "dev",
		ForwardedArgs: []string{"a b", "c"},
		HostEnv:       []string{"PATH=/bin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.specs) != 1 {
		t.Fatalf("specs=%d", len(rec.specs))
	}
	path := rec.specs[0].Path
	if !strings.Contains(path, "a b") || !strings.Contains(path, "c") {
		t.Fatalf("path=%q", path)
	}
}

func TestLaunchUnicodeArgv(t *testing.T) {
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "shell-quoting")
	code, out := runMProject(t, proj, "run", "args", "--", "café")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	raw := readFile(t, filepath.Join(proj, "args.out"))
	if !strings.Contains(raw, "café") {
		t.Fatalf("out=%s", raw)
	}
	_ = out
}

func TestLaunchDoubleDashArgv(t *testing.T) {
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "shell-quoting")
	code, out := runMProject(t, proj, "run", "args", "--", "--help")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	raw := readFile(t, filepath.Join(proj, "args.out"))
	if !strings.Contains(raw, "--help") {
		t.Fatalf("out=%s", raw)
	}
	_ = out
}

type recordSupervisor struct {
	specs []process.Spec
}

func (r *recordSupervisor) Start(_ context.Context, spec process.Spec) (*process.Handle, error) {
	r.specs = append(r.specs, spec)
	return &process.Handle{}, nil
}

func (r *recordSupervisor) Wait(context.Context, *process.Handle) error { return nil }

var _ process.ProcessSupervisor = (*recordSupervisor)(nil)
