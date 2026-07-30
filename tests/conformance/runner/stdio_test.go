package runner_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/mewisme/mew/internal/process"
)

func TestStdioInherit(t *testing.T) {
	skipWithoutNode(t)
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	env := process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir)
	var stdout bytes.Buffer
	spec := process.Spec{
		Path:   "node",
		Args:   []string{"-e", "process.stdout.write('hello')"},
		Dir:    dir,
		Env:    env,
		Stdout: &stdout,
		Stderr: io.Discard,
	}
	h, err := sup.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := sup.Wait(context.Background(), h); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "hello" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestStdioPipeClosure(t *testing.T) {
	skipWithoutNode(t)
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	env := process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir)
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pr.Close() }()
	spec := process.Spec{
		Path:  "node",
		Args:  []string{"-e", "process.stdin.on('end',()=>process.exit(0)); process.stdin.resume()"},
		Dir:   dir,
		Env:   env,
		Stdin: pr,
	}
	h, err := sup.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	_ = pw.Close()
	if err := sup.Wait(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}

func TestStdioNoCorruption(t *testing.T) {
	skipWithoutNode(t)
	proj := setupRunnerFixture(t, "basic-scripts")
	code, out := runMProject(t, proj, "run", "dev")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
}
