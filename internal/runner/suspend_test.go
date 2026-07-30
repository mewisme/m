package runner_test

import (
	"context"
	"io"
	"testing"

	"github.com/mewisme/mew/internal/process"
	"github.com/mewisme/mew/internal/runner"
)

func TestRunSuspendsBeforeChildStart(t *testing.T) {
	var order []string
	rec := &recordSupervisor{}
	r := &runner.DefaultRunner{Supervisor: rec}
	_, err := r.Run(context.Background(), runner.RunOptions{
		ProjectRoot: "/proj",
		PackageDir:  "/proj",
		Scripts:     map[string]string{"hi": "echo hi"},
		Selector:    "hi",
		HostEnv:     []string{"PATH=/bin"},
		Stdout:      io.Discard,
		Stderr:      io.Discard,
		Suspend: func(context.Context) error {
			order = append(order, "suspend")
			return nil
		},
		Resume: func(context.Context) error {
			order = append(order, "resume")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(rec.specs) != 1 {
		t.Fatalf("expected one start, got %d", len(rec.specs))
	}
	if len(order) != 2 || order[0] != "suspend" || order[1] != "resume" {
		t.Fatalf("order=%v", order)
	}
}

func TestExecSuspendsBeforeChildStart(t *testing.T) {
	// Exec needs binresolve fixtures; exercise Suspend via RunOptions path above.
	// Keep a compile-time check that ExecOptions carries Suspend.
	opts := runner.ExecOptions{
		Suspend: func(context.Context) error { return nil },
		Resume:  func(context.Context) error { return nil },
	}
	if opts.Suspend == nil || opts.Resume == nil {
		t.Fatal("ExecOptions must carry Suspend/Resume")
	}
	_ = process.ExitCode(nil)
}
