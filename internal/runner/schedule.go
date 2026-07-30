package runner

import (
	"runtime"
	"sort"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/workspace"
)

// ScheduledTask carries stable scheduler metadata for one workspace package.
type ScheduledTask struct {
	Index int
	Path  string
	Name  string
}

// Schedule is the deterministic execution plan for selected workspace packages.
type Schedule struct {
	Tasks               []ScheduledTask
	Edges               map[string][]string
	InDegree            map[string]int
	EffectiveConcurrent int
	Order               WorkspaceOrder
}

// BuildSchedule assigns stable indices and readiness metadata for the selected set.
func BuildSchedule(g *workspace.WorkspaceGraph, paths []string, order WorkspaceOrder, requestedConcurrency int) (*Schedule, error) {
	if len(paths) == 0 {
		return nil, apperr.New(apperr.NotFound, "runner.schedule", "", "no workspace packages selected")
	}
	edges, inDegree, err := BuildInducedSubgraph(g, paths)
	if err != nil {
		return nil, err
	}
	ordered, err := orderPaths(g, paths, order)
	if err != nil {
		return nil, err
	}
	tasks := make([]ScheduledTask, 0, len(ordered))
	for i, p := range ordered {
		name := p
		if m, ok := memberByPath(g, p); ok && m.Name != "" {
			name = m.Name
		}
		tasks = append(tasks, ScheduledTask{Index: i, Path: p, Name: name})
	}
	eff := effectiveConcurrency(requestedConcurrency, len(tasks))
	schedInDegree := make(map[string]int, len(inDegree))
	for k, v := range inDegree {
		schedInDegree[k] = v
	}
	if order == OrderParallel {
		for p := range schedInDegree {
			schedInDegree[p] = 0
		}
	}
	return &Schedule{
		Tasks:               tasks,
		Edges:               edges,
		InDegree:            schedInDegree,
		EffectiveConcurrent: eff,
		Order:               order,
	}, nil
}

func orderPaths(g *workspace.WorkspaceGraph, paths []string, order WorkspaceOrder) ([]string, error) {
	switch order {
	case OrderTopological:
		return g.TopoOrderFor(paths)
	case OrderReverseTopo:
		return g.ReverseTopoOrder(paths)
	case OrderParallel, OrderSequential:
		out := append([]string(nil), paths...)
		sort.Strings(out)
		return out, nil
	default:
		return nil, apperr.New(apperr.Usage, "runner.schedule", string(order), "invalid --workspace-order")
	}
}

func effectiveConcurrency(requested, taskCount int) int {
	if taskCount == 0 {
		return 0
	}
	eff := requested
	if eff < 0 {
		return -1
	}
	if eff == 0 {
		eff = runtime.GOMAXPROCS(0)
	}
	if eff < 1 {
		eff = 1
	}
	if eff > taskCount {
		eff = taskCount
	}
	return eff
}

// EffectiveConcurrency computes the scheduler concurrency contract.
func EffectiveConcurrency(requested, taskCount int) int {
	return effectiveConcurrency(requested, taskCount)
}

// ValidateConcurrency returns ERR_M_USAGE for negative concurrency.
func ValidateConcurrency(n int) error {
	if n < 0 {
		return apperr.New(apperr.Usage, "runner.workspace", "", "negative --workspace-concurrency")
	}
	return nil
}
