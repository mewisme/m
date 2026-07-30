package runner_test

import (
	"context"
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/process"
)

func TestSignalsSIGINTUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only suite")
	}
	skipWithoutNode(t)
	testCancelPropagates(t)
}

func TestSignalsSIGTERMUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only suite")
	}
	skipWithoutNode(t)
	testCancelPropagates(t)
}

func testCancelPropagates(t *testing.T) {
	t.Helper()
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	env := process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir)
	spec := process.Spec{
		Path:   "node",
		Args:   []string{"-e", "setInterval(()=>{}, 10000)"},
		Dir:    dir,
		Env:    env,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	h, err := sup.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	err = sup.Wait(ctx, h)
	if err == nil {
		t.Fatal("expected cancellation")
	}
}
