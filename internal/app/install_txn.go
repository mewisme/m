package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/hoisted"
	"github.com/mewisme/m/internal/linker/planner"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/snapshot"
	"github.com/mewisme/m/internal/transaction"
)

// manifestEditFn applies in-memory manifest changes before staging (no live write).
type manifestEditFn func(*project.Project) error

// runInstallTxn resolves, stages, validates, and commits install-family mutations.
func runInstallTxn(ctx context.Context, ac *Context, opts InstallOptions, edit manifestEditFn) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.install", "", "missing app context")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	if opts.Frozen {
		if err := ValidateFrozenLock(ctx, ac); err != nil {
			return res, err
		}
	}

	emitPhase(ac, "resolve", "")
	manifestChanged := false
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
		manifestChanged = true
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	resolution, err := resolveForInstall(ctx, ac, proj, opts, manifestChanged)
	if err != nil {
		return res, err
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))

	if opts.DryRun {
		return res, nil
	}

	txn := transaction.NewRunner(proj.Root)
	if err := txn.Begin(ctx); err != nil {
		return res, err
	}
	stage := txn.StagePath()

	if manifestChanged {
		if err := writeStagedManifest(stage, proj); err != nil {
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	emitPhase(ac, "fetch", "")
	extractDir := filepath.Join(stage, "extract")
	stageNM := filepath.Join(stage, "node_modules")
	useStore := config.UseGlobalStore(ac.Config)
	extracts, _, err := fetchPackages(ctx, ac, resolution.Graph, extractDir, useStore)
	if err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}

	emitPhase(ac, "link", "")
	var caps planner.Capabilities
	if useStore {
		storeRoot, storeErr := config.StoreRoot(ac.Config)
		if storeErr == nil {
			caps, _ = planner.ProbeCached(config.CacheRoot(ac.Config), storeRoot, stageNM)
		}
	}
	h := &hoisted.Linker{NodeModules: stageNM, ExtractDirs: extracts, Capabilities: caps, UseSmartLink: useStore}
	linkPlan, err := h.Plan(ctx, resolution.Graph)
	if err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if err := h.Apply(ctx, linkPlan); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	emitLinkSummary(ac, linkPlan.LinkSummary)

	if err := writeStagedLock(stage, ac, proj, resolution); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}

	emitPhase(ac, "validate", "")
	if err := validateStaged(stage, resolution.Graph); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if err := txn.SetState(transaction.StateValidated); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}

	for _, rel := range []string{"package.json", lockFileName, "node_modules", filepath.Join(".mew", "store-manifest.json")} {
		if rel == "package.json" && !manifestChanged {
			continue
		}
		if rel == filepath.Join(".mew", "store-manifest.json") && !useStore {
			continue
		}
		if err := txn.RecordBackup(rel); err != nil {
			emitPhase(ac, "rollback", rel)
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	emitPhase(ac, "commit", "")
	commitOps := buildCommitOps(manifestChanged)
	if err := txn.Commit(ctx, commitOps); err != nil {
		emitPhase(ac, "rollback", "")
		_ = txn.Rollback(ctx)
		return res, err
	}
	liveNM := filepath.Join(proj.Root, "node_modules")
	if err := publishNodeModules(stageNM, liveNM); err != nil {
		emitPhase(ac, "rollback", "")
		_ = txn.Rollback(ctx)
		return res, err
	}

	if useStore {
		if err := writeStoreManifest(proj.Root, resolution.Graph); err != nil {
			emitPhase(ac, "rollback", "")
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	if err := createSnapshot(ctx, ac, proj, resolution); err != nil {
		emitPhase(ac, "rollback", "")
		_ = txn.Rollback(ctx)
		return res, err
	}

	if err := txn.Finish(opts.KeepJournal); err != nil {
		return res, err
	}
	return res, nil
}

func buildCommitOps(manifestChanged bool) []transaction.Op {
	var ops []transaction.Op
	if manifestChanged {
		ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: "package.json", Backup: "stage/package.json"})
	}
	ops = append(ops,
		transaction.Op{Kind: transaction.OpRename, Path: lockFileName, Backup: "stage/m.lock"},
	)
	return ops
}

func writeStagedManifest(stage string, proj *project.Project) error {
	return proj.Doc.Write(filepath.Join(stage, "package.json"))
}

func writeStagedLock(stage string, ac *Context, proj *project.Project, res *resolver.Resolution) error {
	settings, err := mlock.SettingsFromEffective(ac.Config)
	if err != nil {
		return err
	}
	specs := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(proj.Normalized),
	}
	doc, err := mlock.FromResolution(res, specs, settings)
	if err != nil {
		return err
	}
	return mlock.WriteAtomic(filepath.Join(stage, lockFileName), doc)
}

func validateStaged(stage string, g *graph.Graph) error {
	if g == nil {
		return nil
	}
	nm := filepath.Join(stage, "node_modules")
	for _, pkg := range g.Packages {
		pkgPath := packageJSONPath(nm, pkg.ID.Name)
		if _, err := os.Stat(pkgPath); err != nil {
			return apperr.Wrap(apperr.Integrity, "app.validate", pkg.ID.Key(), err)
		}
	}
	return nil
}

func packageJSONPath(nm, name string) string {
	if strings.HasPrefix(name, "@") {
		if i := strings.Index(name, "/"); i > 0 {
			return filepath.Join(nm, name[:i], name[i+1:], "package.json")
		}
	}
	return filepath.Join(nm, name, "package.json")
}

func createSnapshot(ctx context.Context, ac *Context, proj *project.Project, res *resolver.Resolution) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store := snapshot.NewStore(proj.Root)
	id, err := store.NextID()
	if err != nil {
		return err
	}
	manifest, err := os.ReadFile(filepath.Join(proj.Root, "package.json"))
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.snapshot", "package.json", err)
	}
	lock, err := os.ReadFile(LockPath(proj.Root))
	if err != nil {
		return apperr.Wrap(apperr.IO, "app.snapshot", lockFileName, err)
	}
	digest, err := snapshot.GraphDigest(res.Graph)
	if err != nil {
		return err
	}
	if err := store.Create(id, manifest, lock, digest); err != nil {
		return err
	}
	return store.Prune(snapshotRetention(ac))
}

func snapshotRetention(ac *Context) int {
	const defaultRetain = 10
	if ac == nil || ac.Config == nil {
		return defaultRetain
	}
	v, err := config.Get(ac.Config, "transaction.snapshot_retention")
	if err != nil {
		return defaultRetain
	}
	switch n := v.Raw.(type) {
	case int:
		if n > 0 {
			return n
		}
	case float64:
		if int(n) > 0 {
			return int(n)
		}
	}
	return defaultRetain
}

func emitPhase(ac *Context, phase, subject string) {
	if ac == nil || ac.Reporter == nil {
		return
	}
	ac.Reporter.Progress(diagnostics.Event{V: 1, Type: "progress", Phase: phase, Package: subject})
}

func emitLinkSummary(ac *Context, summary linker.LinkSummary) {
	if ac == nil || ac.Reporter == nil {
		return
	}
	if summary.Mkdir == 0 && summary.Copy == 0 && summary.Hardlink == 0 && summary.Reflink == 0 && summary.Symlink == 0 && summary.Junction == 0 {
		return
	}
	subject := fmt.Sprintf("hardlink=%d reflink=%d copy=%d symlink=%d junction=%d mkdir=%d",
		summary.Hardlink, summary.Reflink, summary.Copy, summary.Symlink, summary.Junction, summary.Mkdir)
	ac.Reporter.Progress(diagnostics.Event{V: 1, Type: "progress", Phase: "link", Package: subject})
}
