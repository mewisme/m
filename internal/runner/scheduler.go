package runner

import (
	"context"
	"sort"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/workspace"
)

// RunScheduler executes workspace tasks with readiness queue and bounded concurrency.
func RunScheduler(
	ctx context.Context,
	sched *Schedule,
	exec WorkspaceTaskExecutor,
	opts WorkspaceRunOptions,
	rep diagnostics.Reporter,
	newTaskIO func(ScheduledTask) (TaskIO, func(), error),
) (WorkspaceResult, error) {
	return runScheduler(ctx, nil, sched, exec, opts, rep, newTaskIO)
}

func runScheduler(
	ctx context.Context,
	_ *workspace.WorkspaceGraph,
	sched *Schedule,
	exec WorkspaceTaskExecutor,
	opts WorkspaceRunOptions,
	rep diagnostics.Reporter,
	newTaskIO func(ScheduledTask) (TaskIO, func(), error),
) (WorkspaceResult, error) {
	if sched == nil || len(sched.Tasks) == 0 {
		return WorkspaceResult{}, apperr.New(apperr.NotFound, "runner.scheduler", "", "no tasks to run")
	}

	results := make([]WorkspaceTaskResult, len(sched.Tasks))
	remaining := make(map[string]int, len(sched.InDegree))
	for k, v := range sched.InDegree {
		remaining[k] = v
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		mu          sync.Mutex
		bailed      bool
		triggerErr  error
		triggerExit int
		terminal    int
		readyCh     = make(chan ScheduledTask, len(sched.Tasks))
		doneCh      = make(chan struct{}, len(sched.Tasks))
	)

	releaseDependents := func(path string, success bool) {
		if !success {
			return
		}
		for _, dep := range sched.Edges[path] {
			remaining[dep]--
		}
	}

	enqueueReady := func() {
		mu.Lock()
		defer mu.Unlock()
		if bailed {
			return
		}
		var ready []ScheduledTask
		for _, t := range sched.Tasks {
			if remaining[t.Path] != 0 {
				continue
			}
			if results[t.Index].Status != "" {
				continue
			}
			ready = append(ready, t)
		}
		sort.Slice(ready, func(i, j int) bool { return ready[i].Path < ready[j].Path })
		for _, t := range ready {
			results[t.Index].Status = StatusPending
			readyCh <- t
		}
	}

	handleResult := func(res WorkspaceTaskResult) {
		mu.Lock()
		defer mu.Unlock()
		results[res.Task.Index] = res
		terminal++
		success := res.Status == StatusDone || res.Status == StatusSkipped
		releaseDependents(res.Task.Path, success)
		emitWorkspaceTask(rep, res.Task.Name, opts.Selector, string(res.Status), res.Task.Index, res.ExitCode)

		if res.Status == StatusFailed && opts.Bail && !bailed {
			bailed = true
			triggerErr = res.Err
			triggerExit = res.ExitCode
			if triggerExit == 0 && res.Err != nil {
				triggerExit = apperr.ExitCode(res.Err)
			}
			cancel()
		}
		doneCh <- struct{}{}
	}

	workers := sched.EffectiveConcurrent
	if sched.Order == OrderSequential {
		workers = 1
	}
	if workers < 1 {
		workers = 1
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				case t, ok := <-readyCh:
					if !ok {
						return
					}
					mu.Lock()
					if bailed {
						results[t.Index] = WorkspaceTaskResult{
							Task: WorkspaceTask{
								Index: t.Index, Path: t.Path, Name: t.Name,
								Selector: opts.Selector, ProjectRoot: opts.ProjectRoot,
							},
							Status: StatusNotRun,
						}
						terminal++
						emitWorkspaceTask(rep, t.Name, opts.Selector, string(StatusNotRun), t.Index, 0)
						mu.Unlock()
						doneCh <- struct{}{}
						continue
					}
					mu.Unlock()

					task := WorkspaceTask{
						Index: t.Index, Path: t.Path, Name: t.Name,
						Selector: opts.Selector, ProjectRoot: opts.ProjectRoot,
					}
					emitWorkspaceTask(rep, t.Name, opts.Selector, "start", t.Index, 0)

					ioBundle, cleanup, err := newTaskIO(t)
					if err != nil {
						handleResult(WorkspaceTaskResult{Task: task, Status: StatusFailed, Err: err})
						continue
					}
					res := exec.Run(runCtx, task, ioBundle)
					if cleanup != nil {
						cleanup()
					}
					if pw, ok := ioBundle.Stdout.(*PrefixWriter); ok {
						_ = pw.Flush()
					}
					if pw, ok := ioBundle.Stderr.(*PrefixWriter); ok {
						_ = pw.Flush()
					}
					handleResult(res)
					enqueueReady()
				}
			}
		}()
	}

	enqueueReady()

	for terminal < len(sched.Tasks) {
		select {
		case <-runCtx.Done():
			goto drained
		case <-doneCh:
		}
	}

drained:
	close(readyCh)
	wg.Wait()

	mu.Lock()
	for i, r := range results {
		if r.Status == "" || r.Status == StatusPending {
			t := sched.Tasks[i]
			results[i] = WorkspaceTaskResult{
				Task: WorkspaceTask{
					Index: t.Index, Path: t.Path, Name: t.Name,
					Selector: opts.Selector, ProjectRoot: opts.ProjectRoot,
				},
				Status: StatusNotRun,
			}
			emitWorkspaceTask(rep, t.Name, opts.Selector, string(StatusNotRun), t.Index, 0)
		}
	}
	mu.Unlock()

	wsRes := summarizeResults(results, sched.EffectiveConcurrent)

	if opts.Bail && triggerErr != nil {
		wsRes.ExitCode = triggerExit
		return wsRes, triggerErr
	}
	if !opts.Bail {
		code, err := continueExit(results)
		wsRes.ExitCode = code
		if err != nil {
			return wsRes, err
		}
	}
	return wsRes, nil
}

func summarizeResults(results []WorkspaceTaskResult, effConc int) WorkspaceResult {
	ws := WorkspaceResult{Tasks: results, EffectiveConcurrent: effConc}
	for _, r := range results {
		switch r.Status {
		case StatusDone:
			ws.Completed++
		case StatusFailed:
			ws.Failed++
		case StatusCancelled:
			ws.Cancelled++
		case StatusSkipped:
			ws.Skipped++
		case StatusNotRun:
			ws.NotRun++
		}
	}
	return ws
}

func continueExit(results []WorkspaceTaskResult) (int, error) {
	sorted := append([]WorkspaceTaskResult(nil), results...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Task.Index < sorted[j].Task.Index
	})
	for _, r := range sorted {
		if r.Status == StatusFailed {
			code := r.ExitCode
			if code == 0 && r.Err != nil {
				code = apperr.ExitCode(r.Err)
			}
			return code, r.Err
		}
	}
	return 0, nil
}
