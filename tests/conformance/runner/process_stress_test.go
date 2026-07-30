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

func TestProcessSoakShortCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	const minCycles = 100
	const maxGoroutineDelta = 5
	start := runtime.NumGoroutine()
	for i := 0; i < minCycles; i++ {
		sup := process.NewExecSupervisor()
		dir := t.TempDir()
		env := process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir)
		spec := process.Spec{
			Path:   "node",
			Args:   []string{"-e", "process.exit(0)"},
			Dir:    dir,
			Env:    env,
			Stdout: io.Discard,
			Stderr: io.Discard,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		h, err := sup.Start(ctx, spec)
		if err != nil {
			cancel()
			t.Skip("node required for soak")
		}
		_ = sup.Wait(ctx, h)
		cancel()
	}
	runtime.GC()
	runtime.GC()
	delta := runtime.NumGoroutine() - start
	if delta > maxGoroutineDelta {
		t.Fatalf("goroutine delta=%d exceeds %d", delta, maxGoroutineDelta)
	}
}

func TestProcessTreeGrandchildCleanupUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only suite")
	}
	skipWithoutNode(t)
	testCancelPropagates(t)
}

func TestProcessTreeJobObjectCleanupWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only suite")
	}
	skipWithoutNode(t)
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	env := process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	spec := process.Spec{
		Path:   "node",
		Args:   []string{"-e", "setInterval(()=>{}, 10000)"},
		Dir:    dir,
		Env:    env,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	h, err := sup.Start(ctx, spec)
	if err != nil {
		t.Skip("node required")
	}
	_ = sup.Wait(ctx, h)
}
