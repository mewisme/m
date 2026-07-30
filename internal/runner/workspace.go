package runner

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/workspace"
)

// WorkspaceOutputMode controls child output multiplexing.
type WorkspaceOutputMode string

const (
	OutputStream    WorkspaceOutputMode = "stream"
	OutputAggregate WorkspaceOutputMode = "aggregate"
)

// WorkspaceOrder controls task scheduling order.
type WorkspaceOrder string

const (
	OrderTopological WorkspaceOrder = "topological"
	OrderReverseTopo WorkspaceOrder = "reverse-topological"
	OrderParallel    WorkspaceOrder = "parallel"
	OrderSequential  WorkspaceOrder = "sequential"
)

// TaskStatus is a terminal or in-flight workspace task state.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusDone      TaskStatus = "done"
	StatusFailed    TaskStatus = "fail"
	StatusSkipped   TaskStatus = "skip"
	StatusCancelled TaskStatus = "cancel"
	StatusNotRun    TaskStatus = "not-run"
)

// WorkspaceRunOptions configures workspace script orchestration.
type WorkspaceRunOptions struct {
	ProjectRoot   string
	Selector      string
	IfPresent     bool
	ForwardedArgs []string
	Recursive     bool
	Filters       []string
	Concurrency   int
	Order         WorkspaceOrder
	Output        WorkspaceOutputMode
	Bail          bool
	HostEnv       []string
	TempDir       string // workspace-run spill dir; empty = derive from project
	// Suspend / Resume pause presentation around child launch when tasks inherit TTYs.
	Suspend func(context.Context) error
	Resume  func(context.Context) error
}

// WorkspaceTask is one scheduled package script execution.
type WorkspaceTask struct {
	Index       int
	Path        string // workspace-relative POSIX path
	Name        string // package name
	Selector    string
	ProjectRoot string
}

// WorkspaceTaskResult is the outcome of one workspace task.
type WorkspaceTaskResult struct {
	Task     WorkspaceTask
	Status   TaskStatus
	ExitCode int
	Err      error
}

// WorkspaceResult summarizes a workspace run.
type WorkspaceResult struct {
	Tasks               []WorkspaceTaskResult
	EffectiveConcurrent int
	Completed           int
	Failed              int
	Cancelled           int
	Skipped             int
	NotRun              int
	ExitCode            int
}

// SelectMembers resolves workspace packages for -r / --filter.
func SelectMembers(g *workspace.WorkspaceGraph, recursive bool, filters []string) ([]string, error) {
	if g == nil {
		return nil, apperr.New(apperr.Manifest, "runner.workspace", "", "not a workspace project")
	}
	var paths []string
	switch {
	case len(filters) > 0:
		ids, err := workspace.ExpandFilter(g, filters)
		if err != nil {
			return nil, err
		}
		paths = make([]string, 0, len(ids))
		for _, id := range ids {
			p := string(id)
			if p != "." {
				paths = append(paths, p)
			}
		}
	case recursive:
		for _, p := range g.MemberPaths() {
			if p != "." {
				paths = append(paths, p)
			}
		}
	default:
		return nil, apperr.New(apperr.Usage, "runner.workspace", "", "workspace mode requires -r or --filter")
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, apperr.New(apperr.NotFound, "runner.workspace", "", "no workspace packages selected")
	}
	return paths, nil
}

// BuildInducedSubgraph validates and returns edges and in-degrees for selected paths.
func BuildInducedSubgraph(g *workspace.WorkspaceGraph, paths []string) (map[string][]string, map[string]int, error) {
	if err := workspace.ValidateSelectedCycle(g, paths); err != nil {
		return nil, nil, err
	}
	return workspace.InducedSubgraph(g, paths)
}

// ParseWorkspaceOrder validates an order flag value.
func ParseWorkspaceOrder(s string) (WorkspaceOrder, error) {
	switch WorkspaceOrder(s) {
	case OrderTopological, OrderReverseTopo, OrderParallel, OrderSequential:
		return WorkspaceOrder(s), nil
	default:
		return "", apperr.New(apperr.Usage, "runner.workspace", s, "invalid --workspace-order")
	}
}

// ParseWorkspaceOutput validates an output flag value.
func ParseWorkspaceOutput(s string) (WorkspaceOutputMode, error) {
	switch WorkspaceOutputMode(s) {
	case OutputStream, OutputAggregate:
		return WorkspaceOutputMode(s), nil
	default:
		return "", apperr.New(apperr.Usage, "runner.workspace", s, "invalid --workspace-output")
	}
}

// memberByPath returns the workspace member for path.
func memberByPath(g *workspace.WorkspaceGraph, path string) (workspace.Member, bool) {
	if g == nil {
		return workspace.Member{}, false
	}
	m, ok := g.ByPath[path]
	return m, ok
}

// packageDir returns the absolute package directory for a workspace member path.
func packageDir(projectRoot, relPath string) string {
	return filepath.Join(projectRoot, filepath.FromSlash(relPath))
}
