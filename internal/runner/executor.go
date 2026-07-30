package runner

import (
	"context"
	"io"
	"path/filepath"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/manifest"
)

// TaskIO carries stdin/stdout/stderr for one workspace task.
type TaskIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// WorkspaceTaskExecutor runs one workspace package script.
type WorkspaceTaskExecutor interface {
	Run(context.Context, WorkspaceTask, TaskIO) WorkspaceTaskResult
}

// FakeWorkspaceTaskExecutor supports deterministic scheduler tests.
type FakeWorkspaceTaskExecutor struct {
	mu sync.Mutex

	Active     int
	MaxActive  int
	StartOrder []string

	BlockUntil map[string]<-chan struct{}
	Results    map[string]WorkspaceTaskResult

	OnStart func(WorkspaceTask)
}

func (f *FakeWorkspaceTaskExecutor) Run(ctx context.Context, task WorkspaceTask, _ TaskIO) WorkspaceTaskResult {
	f.mu.Lock()
	f.Active++
	if f.Active > f.MaxActive {
		f.MaxActive = f.Active
	}
	f.StartOrder = append(f.StartOrder, task.Path)
	block := f.BlockUntil[task.Path]
	onStart := f.OnStart
	f.mu.Unlock()

	if onStart != nil {
		onStart(task)
	}

	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			f.mu.Lock()
			f.Active--
			f.mu.Unlock()
			return WorkspaceTaskResult{
				Task:     task,
				Status:   StatusCancelled,
				ExitCode: 130,
				Err:      ctx.Err(),
			}
		}
	}

	if ctx.Err() != nil {
		f.mu.Lock()
		f.Active--
		f.mu.Unlock()
		return WorkspaceTaskResult{
			Task:     task,
			Status:   StatusCancelled,
			ExitCode: 130,
			Err:      ctx.Err(),
		}
	}

	f.mu.Lock()
	res, ok := f.Results[task.Path]
	f.Active--
	f.mu.Unlock()

	if ok {
		res.Task = task
		return res
	}
	return WorkspaceTaskResult{
		Task:     task,
		Status:   StatusDone,
		ExitCode: 0,
	}
}

// RealWorkspaceExecutor wraps DefaultRunner for workspace tasks.
type RealWorkspaceExecutor struct {
	Selector  string
	IfPresent bool
	Forwarded []string
	HostEnv   []string
	Reporter  diagnostics.Reporter
	Runner    ScriptRunner
}

func (e *RealWorkspaceExecutor) Run(ctx context.Context, task WorkspaceTask, io TaskIO) WorkspaceTaskResult {
	pkgDir := packageDir(task.ProjectRoot, task.Path)
	doc, err := manifest.Load(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return WorkspaceTaskResult{
			Task:   task,
			Status: StatusFailed,
			Err:    apperr.Wrap(apperr.Manifest, "runner.workspace", task.Path, err),
		}
	}
	scripts := map[string]string{}
	if doc.Scripts != nil {
		scripts = doc.Scripts
	}
	selector := e.Selector
	if selector == "" {
		selector = task.Selector
	}

	runnerImpl := e.Runner
	if runnerImpl == nil {
		runnerImpl = NewDefaultRunner()
	}
	res, runErr := runnerImpl.Run(ctx, RunOptions{
		ProjectRoot:   task.ProjectRoot,
		PackageDir:    pkgDir,
		NodeModules:   filepath.Join(pkgDir, "node_modules"),
		PackageName:   doc.Name,
		PackageVer:    doc.Version,
		Scripts:       scripts,
		Selector:      selector,
		IfPresent:     e.IfPresent,
		ForwardedArgs: e.Forwarded,
		HostEnv:       e.HostEnv,
		Reporter:      e.Reporter,
		Stdin:         io.Stdin,
		Stdout:        io.Stdout,
		Stderr:        io.Stderr,
	})
	if runErr == nil {
		return WorkspaceTaskResult{Task: task, Status: StatusDone, ExitCode: res.ExitCode}
	}
	if apperr.CodeOf(runErr) == apperr.Cancelled {
		return WorkspaceTaskResult{Task: task, Status: StatusCancelled, ExitCode: res.ExitCode, Err: runErr}
	}
	if apperr.CodeOf(runErr) == apperr.NotFound && e.IfPresent {
		return WorkspaceTaskResult{Task: task, Status: StatusSkipped}
	}
	return WorkspaceTaskResult{
		Task:     task,
		Status:   StatusFailed,
		ExitCode: apperr.ExitCode(runErr),
		Err:      runErr,
	}
}
