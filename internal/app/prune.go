package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker/hoisted"
	"github.com/mewisme/mew/internal/linker/isolated"
	"github.com/mewisme/mew/internal/plan"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/transaction"
)

// PruneOptions controls top-level m prune (extraneous node_modules removal).
type PruneOptions struct {
	Prod   bool
	DryRun bool
	InstallOptions
}

// PruneNodeModules removes node_modules entries not required by the lockfile graph.
func PruneNodeModules(ctx context.Context, ac *Context, opts PruneOptions) (InstallResult, error) {
	var res InstallResult
	if opts.DryRun {
		return runPruneDryRun(ctx, ac, opts)
	}
	root, err := resolveProjectRoot(ac, "")
	if err != nil {
		return res, err
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		return res, err
	}
	res, err = runPruneInSession(ctx, sess, opts)
	if err != nil {
		abortRes, abortErr := abortMutation(ctx, sess, sess.Runner(), err)
		res = mergeInstallResults(res, abortRes)
		return res, abortErr
	}
	finish, finishErr := sess.Finish(ctx, opts.KeepJournal)
	if finish.Committed {
		res.Committed = true
	}
	if finish.HasCriticalCleanupFailure() {
		populateCleanupResult(&res, finish)
		return res, finishErr
	}
	populateWarningCleanup(&res, finish)
	return res, finishErr
}

func runPruneDryRun(ctx context.Context, ac *Context, opts PruneOptions) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	extraneous, err := findExtraneousNodeModules(ctx, ac, proj, opts)
	if err != nil {
		return res, err
	}
	res.Removed = len(extraneous)
	res.Packages = len(extraneous)
	p := &plan.Plan{SchemaVersion: plan.SchemaVersion}
	for _, rel := range extraneous {
		p.Operations = append(p.Operations, plan.Operation{
			Op: "remove", Subject: rel, Detail: "extraneous node_modules entry",
		})
	}
	if err := p.Normalize(); err != nil {
		return res, err
	}
	res.Plan = p
	return res, nil
}

func runPruneInSession(ctx context.Context, sess *MutationSession, opts PruneOptions) (InstallResult, error) {
	var res InstallResult
	proj, err := sess.ReopenProject(ctx)
	if err != nil {
		return res, err
	}
	ac, err := sess.AppContext()
	if err != nil {
		return res, err
	}
	txn := sess.Runner()
	extraneous, err := findExtraneousNodeModules(ctx, ac, proj, opts)
	if err != nil {
		return res, err
	}
	res.Removed = len(extraneous)
	if len(extraneous) == 0 {
		return res, nil
	}
	nmRel := "node_modules"
	if _, err := os.Stat(filepath.Join(proj.Root, nmRel)); err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return res, apperr.Wrap(apperr.IO, "app.prune", nmRel, err)
	}
	if err := txn.RecordBackup(nmRel); err != nil {
		return res, err
	}
	ops := make([]transaction.Op, 0, len(extraneous))
	for _, rel := range extraneous {
		ops = append(ops, transaction.Op{Kind: transaction.OpRemove, Path: rel})
	}
	if err := txn.SetPlan(ops); err != nil {
		return res, err
	}
	if err := txn.Commit(ctx, nil); err != nil {
		return res, err
	}
	return res, nil
}

func findExtraneousNodeModules(ctx context.Context, ac *Context, proj *project.Project, opts PruneOptions) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "app.prune", LockPath(proj), "no lockfile")
	}
	if opts.Prod {
		g = prodPruneGraph(g)
	}
	linkerMode, err := resolveLinkerMode(ctx, ac, proj, opts.InstallOptions)
	if err != nil {
		return nil, err
	}
	nmRoot := filepath.Join(proj.Root, "node_modules")
	if _, err := os.Stat(nmRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "app.prune", nmRoot, err)
	}
	keep, err := expectedNodeModulePaths(g, linkerMode, nmRoot)
	if err != nil {
		return nil, err
	}
	return scanExtraneous(nmRoot, keep, proj.Root)
}

func prodPruneGraph(g *graph.Graph) *graph.Graph {
	if g == nil {
		return g
	}
	keep := map[string]struct{}{string(graph.RootImporter): {}}
	var queue []string
	queue = append(queue, string(graph.RootImporter))
	for len(queue) > 0 {
		from := queue[0]
		queue = queue[1:]
		for _, e := range g.Edges {
			if e.From != from {
				continue
			}
			if from == string(graph.RootImporter) && e.Kind == graph.DepDev {
				continue
			}
			if _, ok := keep[e.To]; ok {
				continue
			}
			keep[e.To] = struct{}{}
			queue = append(queue, e.To)
		}
	}
	b := graph.NewBuilder()
	for _, im := range g.Importers {
		b.Importer(im.ID, im.Name)
	}
	for _, p := range g.Packages {
		if _, ok := keep[p.ID.Key()]; ok {
			b.Package(p.ID, p.Integrity, p.TarballURL)
		}
	}
	for _, e := range g.Edges {
		if _, ok := keep[e.From]; !ok {
			continue
		}
		if _, ok := keep[e.To]; !ok {
			continue
		}
		b.EdgeEx(e.From, e.Name, e.To, e.Kind, e.Range, e.Optional)
	}
	out, err := b.Build()
	if err != nil {
		return g
	}
	return out
}

func expectedNodeModulePaths(g *graph.Graph, linkerMode, nmRoot string) (map[string]struct{}, error) {
	keep := map[string]struct{}{}
	addPath := func(p string) {
		p = filepath.Clean(p)
		keep[p] = struct{}{}
		for dir := p; strings.HasPrefix(dir, nmRoot) && dir != nmRoot; dir = filepath.Dir(dir) {
			keep[dir] = struct{}{}
		}
	}
	addPath(nmRoot)
	addPath(filepath.Join(nmRoot, ".bin"))
	switch linkerMode {
	case "isolated":
		layout, err := isolated.ComputeLayout(g, nmRoot)
		if err != nil {
			return nil, err
		}
		addPath(filepath.Join(nmRoot, ".pnpm"))
		for _, p := range layout.Packages {
			addPath(p.ContentDir)
			addPath(p.PrivateNM)
		}
		for _, link := range layout.Aliases {
			addPath(link.Src)
			addPath(link.Dest)
		}
		for _, link := range layout.DepLinks {
			addPath(link.Src)
			addPath(link.Dest)
		}
	default:
		placements, err := hoisted.Placements(g, nmRoot)
		if err != nil {
			return nil, err
		}
		for _, p := range placements {
			addPath(p.DestDir)
		}
	}
	return keep, nil
}

func scanExtraneous(nmRoot string, keep map[string]struct{}, projectRoot string) ([]string, error) {
	entries, err := os.ReadDir(nmRoot)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "app.prune", nmRoot, err)
	}
	var extraneous []string
	for _, e := range entries {
		name := e.Name()
		if name == ".bin" || name == ".pnpm" || name == ".modules.yaml" {
			continue
		}
		path := filepath.Clean(filepath.Join(nmRoot, name))
		if topLevelKept(name, path, keep) {
			continue
		}
		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil, err
		}
		extraneous = append(extraneous, filepath.ToSlash(rel))
	}
	sort.Strings(extraneous)
	return extraneous, nil
}

func topLevelKept(name, path string, keep map[string]struct{}) bool {
	if _, ok := keep[path]; ok {
		return true
	}
	if !strings.HasPrefix(name, "@") {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		child := filepath.Clean(filepath.Join(path, e.Name()))
		if _, ok := keep[child]; ok {
			return true
		}
	}
	return false
}
