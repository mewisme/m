package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/workspace"
)

func testWSGraph(t *testing.T, rel string) *workspace.WorkspaceGraph {
	t.Helper()
	modRoot := moduleRoot(t)
	root := filepath.Join(modRoot, "fixtures", "workspace-runner", rel)
	g, err := workspace.BuildGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestSelectMembersRecursive(t *testing.T) {
	g := testWSGraph(t, "dag-simple")
	paths, err := runner.SelectMembers(g, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("paths=%v", paths)
	}
}

func TestValidateSelectedCycle(t *testing.T) {
	g := testWSGraph(t, "cycle")
	paths := []string{"packages/alpha", "packages/beta"}
	err := workspace.ValidateSelectedCycle(g, paths)
	if err == nil {
		t.Fatal("expected cycle error")
	}
	if apperr.CodeOf(err) != apperr.Resolve {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestBuildScheduleStableOrder(t *testing.T) {
	g := testWSGraph(t, "dag-simple")
	paths, _ := runner.SelectMembers(g, true, nil)
	sched1, err := runner.BuildSchedule(g, paths, runner.OrderTopological, 0)
	if err != nil {
		t.Fatal(err)
	}
	sched2, err := runner.BuildSchedule(g, paths, runner.OrderTopological, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := range sched1.Tasks {
		if sched1.Tasks[i].Path != sched2.Tasks[i].Path {
			t.Fatalf("unstable order: %v vs %v", sched1.Tasks, sched2.Tasks)
		}
	}
}

func TestSchedulerConcurrencyCap(t *testing.T) {
	g := testWSGraph(t, "large")
	paths, _ := runner.SelectMembers(g, true, nil)
	sched, err := runner.BuildSchedule(g, paths, runner.OrderParallel, 2)
	if err != nil {
		t.Fatal(err)
	}
	if sched.EffectiveConcurrent != 2 {
		t.Fatalf("effective=%d want 2", sched.EffectiveConcurrent)
	}

	fake := &runner.FakeWorkspaceTaskExecutor{
		BlockUntil: map[string]<-chan struct{}{},
	}
	unblock := make(chan struct{})
	for _, p := range paths {
		fake.BlockUntil[p] = unblock
	}

	opts := runner.WorkspaceRunOptions{
		ProjectRoot: g.Root,
		Selector:    "build",
		Order:       runner.OrderParallel,
		Bail:        true,
		Output:      runner.OutputStream,
		Concurrency: 2,
	}
	done := make(chan struct{})
	go func() {
		_, _ = runner.RunScheduler(context.Background(), sched, fake, opts, nil, func(t runner.ScheduledTask) (runner.TaskIO, func(), error) {
			return runner.TaskIO{}, nil, nil
		})
		close(done)
	}()

	// ponytail: brief spin until two tasks are active or run completes.
	for i := 0; i < 100 && fake.MaxActive < 2; i++ {
		select {
		case <-done:
			goto finished
		default:
		}
	}
	if fake.MaxActive > 2 {
		t.Fatalf("maxActive=%d want <=2", fake.MaxActive)
	}
	close(unblock)
finished:
	<-done
	if fake.MaxActive > 2 {
		t.Fatalf("maxActive=%d want <=2", fake.MaxActive)
	}
}

func TestSchedulerBailPreservesTriggerExit(t *testing.T) {
	g := testWSGraph(t, "failure-bail")
	paths, _ := runner.SelectMembers(g, true, nil)
	sched, err := runner.BuildSchedule(g, paths, runner.OrderParallel, 4)
	if err != nil {
		t.Fatal(err)
	}
	failErr := apperr.New(apperr.Internal, "test", "fail", "boom")
	failErr.ExitHint = 7
	fake := &runner.FakeWorkspaceTaskExecutor{
		Results: map[string]runner.WorkspaceTaskResult{
			"packages/fail": {Status: runner.StatusFailed, ExitCode: 7, Err: failErr},
		},
	}
	opts := runner.WorkspaceRunOptions{
		ProjectRoot: g.Root,
		Selector:    "build",
		Order:       runner.OrderParallel,
		Bail:        true,
		Output:      runner.OutputStream,
	}
	res, err := runner.RunScheduler(context.Background(), sched, fake, opts, nil, func(t runner.ScheduledTask) (runner.TaskIO, func(), error) {
		return runner.TaskIO{}, nil, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.ExitCode(err) != 7 {
		t.Fatalf("exit=%d", apperr.ExitCode(err))
	}
	if res.NotRun == 0 {
		t.Fatalf("expected not-run tasks, got %+v", res)
	}
}

func TestContinueExitEarliestIndex(t *testing.T) {
	g := testWSGraph(t, "failure-bail")
	paths, _ := runner.SelectMembers(g, true, nil)
	sched, _ := runner.BuildSchedule(g, paths, runner.OrderSequential, 1)
	failErr := apperr.New(apperr.Internal, "test", "fail", "boom")
	failErr.ExitHint = 7
	fake := &runner.FakeWorkspaceTaskExecutor{
		Results: map[string]runner.WorkspaceTaskResult{
			"packages/fail": {Status: runner.StatusFailed, ExitCode: 7, Err: failErr},
		},
	}
	opts := runner.WorkspaceRunOptions{
		ProjectRoot: g.Root,
		Selector:    "build",
		Order:       runner.OrderSequential,
		Bail:        false,
		Output:      runner.OutputStream,
	}
	_, err := runner.RunScheduler(context.Background(), sched, fake, opts, nil, func(t runner.ScheduledTask) (runner.TaskIO, func(), error) {
		return runner.TaskIO{}, nil, nil
	})
	if err == nil || apperr.ExitCode(err) != 7 {
		t.Fatalf("exit err=%v", err)
	}
}

func TestIfPresentSkipReleasesDependent(t *testing.T) {
	g := testWSGraph(t, "dag-simple")
	paths := []string{"packages/base", "packages/lib", "packages/app"}
	sched, _ := runner.BuildSchedule(g, paths, runner.OrderTopological, 1)
	fake := &runner.FakeWorkspaceTaskExecutor{
		Results: map[string]runner.WorkspaceTaskResult{
			"packages/lib": {Status: runner.StatusSkipped},
		},
	}
	opts := runner.WorkspaceRunOptions{
		ProjectRoot: g.Root,
		Selector:    "build",
		IfPresent:   true,
		Order:       runner.OrderTopological,
		Bail:        true,
		Output:      runner.OutputStream,
	}
	res, err := runner.RunScheduler(context.Background(), sched, fake, opts, nil, func(t runner.ScheduledTask) (runner.TaskIO, func(), error) {
		return runner.TaskIO{}, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped != 1 {
		t.Fatalf("skipped=%d", res.Skipped)
	}
}

func TestValidateConcurrencyNegative(t *testing.T) {
	if err := runner.ValidateConcurrency(-1); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestParseWorkspaceOrderInvalid(t *testing.T) {
	if _, err := runner.ParseWorkspaceOrder("bogus"); err == nil {
		t.Fatal("expected error")
	}
}
