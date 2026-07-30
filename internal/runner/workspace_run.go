package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/workspace"
)

// WorkspaceRunner orchestrates workspace script execution.
type WorkspaceRunner struct {
	Executor WorkspaceTaskExecutor
}

// RunWorkspace executes scripts across selected workspace packages.
func RunWorkspace(
	ctx context.Context,
	g *workspace.WorkspaceGraph,
	paths []string,
	opts WorkspaceRunOptions,
	rep diagnostics.Reporter,
) (WorkspaceResult, error) {
	if err := ValidateConcurrency(opts.Concurrency); err != nil {
		return WorkspaceResult{}, err
	}
	sched, err := BuildSchedule(g, paths, opts.Order, opts.Concurrency)
	if err != nil {
		return WorkspaceResult{}, err
	}

	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = filepath.Join(opts.ProjectRoot, ".mew", "tmp", "workspace-run", "run")
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	aggregate := opts.Output == OutputAggregate
	var aggMu sync.Mutex
	aggStdout := map[int]*AggregateBuffer{}
	aggStderr := map[int]*AggregateBuffer{}

	newTaskIO := func(t ScheduledTask) (TaskIO, func(), error) {
		if aggregate {
			key := fmt.Sprintf("%d-%s", t.Index, t.Path)
			so := NewAggregateBuffer(tempDir, key, "stdout")
			se := NewAggregateBuffer(tempDir, key, "stderr")
			aggMu.Lock()
			aggStdout[t.Index] = so
			aggStderr[t.Index] = se
			aggMu.Unlock()
			return TaskIO{Stdout: so, Stderr: se, Stdin: os.Stdin}, func() {
				_ = so.Close()
				_ = se.Close()
			}, nil
		}
		structured := diagnostics.IsStructured(rep)
		return TaskIO{
			Stdin:  os.Stdin,
			Stdout: NewPrefixWriter(t.Name, opts.Selector, "stdout", os.Stdout, rep, structured),
			Stderr: NewPrefixWriter(t.Name, opts.Selector, "stderr", os.Stderr, rep, structured),
		}, nil, nil
	}

	var exec WorkspaceTaskExecutor = &RealWorkspaceExecutor{
		Selector:  opts.Selector,
		IfPresent: opts.IfPresent,
		Forwarded: opts.ForwardedArgs,
		HostEnv:   opts.HostEnv,
		Reporter:  rep,
	}
	if workspaceRunnerExecutor != nil {
		exec = workspaceRunnerExecutor
	}

	res, runErr := RunScheduler(ctx, sched, exec, opts, rep, newTaskIO)

	if aggregate {
		if emitErr := emitAggregateBlocks(rep, sched, aggStdout, aggStderr, opts.Selector); emitErr != nil {
			return res, emitErr
		}
	}
	emitWorkspaceSummary(rep, res.Completed, res.Failed, res.Cancelled, res.Skipped, res.NotRun, res.EffectiveConcurrent)
	return res, runErr
}

// workspaceRunnerExecutor is set by tests to inject a fake executor.
var workspaceRunnerExecutor WorkspaceTaskExecutor

// SetWorkspaceExecutorForTest injects a fake executor for unit tests.
func SetWorkspaceExecutorForTest(exec WorkspaceTaskExecutor) {
	workspaceRunnerExecutor = exec
}

func emitAggregateBlocks(rep diagnostics.Reporter, sched *Schedule, stdout, stderr map[int]*AggregateBuffer, script string) error {
	for _, t := range sched.Tasks {
		var soBytes, seBytes []byte
		var err error
		if so := stdout[t.Index]; so != nil {
			soBytes, err = so.Bytes()
			if err != nil {
				return err
			}
		}
		if se := stderr[t.Index]; se != nil {
			seBytes, err = se.Bytes()
			if err != nil {
				return err
			}
		}
		structured := diagnostics.IsStructured(rep)
		if structured {
			task := WorkspaceTask{Index: t.Index, Path: t.Path, Name: t.Name}
			if len(soBytes) > 0 {
				EmitChildOutput(rep, OutputAggregate, task, script, "stdout", string(soBytes), false, nil)
			}
			if len(seBytes) > 0 {
				EmitChildOutput(rep, OutputAggregate, task, script, "stderr", string(seBytes), false, nil)
			}
			continue
		}
		if len(soBytes) > 0 {
			fmt.Fprintf(os.Stdout, "[%s] %s\n", t.Name, soBytes)
		}
		if len(seBytes) > 0 {
			fmt.Fprintf(os.Stderr, "[%s] %s\n", t.Name, seBytes)
		}
	}
	return nil
}
