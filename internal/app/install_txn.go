package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/diagnostics"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/linker/isolated"
	"github.com/mewisme/m/internal/linker/planner"
	"github.com/mewisme/m/internal/lockfile/mlock"
	"github.com/mewisme/m/internal/project"
	"github.com/mewisme/m/internal/resolver"
	"github.com/mewisme/m/internal/snapshot"
	"github.com/mewisme/m/internal/transaction"
)

// manifestEditFn applies in-memory manifest changes before staging (no live write).
type manifestEditFn func(*project.Project) error

type commitPlanInput struct {
	manifestChanged bool
	useStore        bool
	snapshotID      string
}

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
	manifestChanged := opts.WriteManifest
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	resolution, err := resolveForInstall(ctx, ac, proj, opts, manifestChanged)
	if err != nil {
		return res, err
	}
	if err := guardLocalInstall(resolution); err != nil {
		return res, err
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))

	if opts.DryRun {
		if p, err := BuildMutationPlan(resolution.Graph); err == nil {
			res.Plan = p
		}
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
	linkerMode, err := resolveLinkerMode(ctx, ac, proj, opts)
	if err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
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
	lnk := newLinker(linkerMode, linkerOpts{
		NodeModules: stageNM, ExtractDirs: extracts, Capabilities: caps, UseSmartLink: useStore,
	})
	linkPlan, err := lnk.Plan(ctx, resolution.Graph)
	if err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if err := lnk.Apply(ctx, linkPlan); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	emitLinkSummary(ac, linkPlan.LinkSummary)
	if linkerMode == "isolated" {
		emitPhase(ac, "link", fmt.Sprintf("linker=isolated packages=%d", len(resolution.Graph.Packages)))
		if err := writeModulesMetadata(stageNM, resolution.Graph); err != nil {
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	if err := writeStagedLock(stage, ac, proj, resolution); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if useStore {
		if err := writeStagedStoreManifest(stage, resolution.Graph); err != nil {
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	snapID, err := stageSnapshot(ctx, stage, proj, resolution)
	if err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}

	emitPhase(ac, "validate", "")
	if err := validateStaged(stage, linkPlan, resolution.Graph, linkerMode); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if err := txn.SetState(transaction.StateValidated); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}
	if err := transaction.InvokeTestHook("post_validate", 0); err != nil {
		_ = txn.Rollback(ctx)
		return res, apperr.Wrap(apperr.Transaction, "app.install", "validate", err)
	}

	plan := buildCommitPlan(commitPlanInput{manifestChanged: manifestChanged, useStore: useStore, snapshotID: snapID})
	if err := txn.SetPlan(plan); err != nil {
		_ = txn.Rollback(ctx)
		return res, err
	}

	backupPaths := []string{lockFileName, "node_modules"}
	if manifestChanged {
		backupPaths = append([]string{"package.json"}, backupPaths...)
	}
	if useStore {
		backupPaths = append(backupPaths, filepath.Join(".mew", "store-manifest.json"))
	}
	backupPaths = append(backupPaths, filepath.Join(".mew", "snapshots", "index.json"))

	for _, rel := range backupPaths {
		if err := txn.RecordBackup(rel); err != nil {
			emitPhase(ac, "rollback", rel)
			_ = txn.Rollback(ctx)
			return res, err
		}
	}

	emitPhase(ac, "commit", "")
	if err := txn.Commit(ctx, nil); err != nil {
		emitPhase(ac, "rollback", "")
		_ = txn.Rollback(ctx)
		return res, err
	}

	if err := pruneSnapshots(ac, proj); err != nil {
		return res, err
	}

	if err := txn.Finish(opts.KeepJournal); err != nil {
		return res, err
	}
	return res, nil
}

func buildCommitPlan(in commitPlanInput) []transaction.Op {
	var ops []transaction.Op
	if in.manifestChanged {
		ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: "package.json", Backup: "stage/package.json"})
	}
	ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: lockFileName, Backup: "stage/m.lock"})
	ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules"})
	if in.useStore {
		ops = append(ops, transaction.Op{
			Kind:   transaction.OpRename,
			Path:   filepath.Join(".mew", "store-manifest.json"),
			Backup: filepath.Join("stage", ".mew", "store-manifest.json"),
		})
	}
	if in.snapshotID != "" {
		snapRel := filepath.Join(".mew", "snapshots")
		ops = append(ops,
			transaction.Op{Kind: transaction.OpMkdir, Path: snapRel},
			transaction.Op{Kind: transaction.OpRename, Path: filepath.Join(snapRel, in.snapshotID), Backup: filepath.Join("stage", "snapshots", in.snapshotID)},
			transaction.Op{Kind: transaction.OpRename, Path: filepath.Join(snapRel, "index.json"), Backup: filepath.Join("stage", "snapshots", "index.json")},
		)
	}
	return ops
}

func stageSnapshot(ctx context.Context, stage string, proj *project.Project, res *resolver.Resolution) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	store := snapshot.NewStore(proj.Root)
	ids, id, nextSeq, err := store.PlannedIndex()
	if err != nil {
		return "", err
	}
	manifestPath := filepath.Join(proj.Root, "package.json")
	if _, statErr := os.Stat(filepath.Join(stage, "package.json")); statErr == nil {
		manifestPath = filepath.Join(stage, "package.json")
	}
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "app.snapshot", "package.json", err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(stage, lockFileName))
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "app.snapshot", lockFileName, err)
	}
	digest, err := snapshot.GraphDigest(res.Graph)
	if err != nil {
		return "", err
	}
	stageSnap := filepath.Join(stage, "snapshots")
	if err := store.StageCreate(stageSnap, id, manifest, lockBytes, digest); err != nil {
		return "", err
	}
	if err := store.StageIndex(stageSnap, ids, nextSeq); err != nil {
		return "", err
	}
	return id, nil
}

func pruneSnapshots(ac *Context, proj *project.Project) error {
	store := snapshot.NewStore(proj.Root)
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

func validateStaged(stage string, linkPlan *linker.Plan, g *graph.Graph, linkerMode string) error {
	if g == nil || linkPlan == nil {
		return nil
	}
	nm := filepath.Join(stage, "node_modules")
	if linkerMode == "isolated" {
		return validateStagedIsolated(nm, linkPlan, g)
	}
	return validateStagedHoisted(nm, linkPlan, g)
}

func validateStagedHoisted(nm string, linkPlan *linker.Plan, g *graph.Graph) error {
	children := childEdgesForValidate(g)
	byKey := packagesByKey(g)
	placed := placementsByKey(linkPlan.Placements)

	for _, pl := range linkPlan.Placements {
		if err := validatePlacementPackage(nm, pl, byKey); err != nil {
			return err
		}
	}
	for from, tos := range children {
		for _, toKey := range tos {
			if err := validateReachableDep(nm, from, toKey, placed, byKey); err != nil {
				return err
			}
		}
	}
	return validateBinTargets(linkPlan.Bins)
}

func validateStagedIsolated(nm string, linkPlan *linker.Plan, g *graph.Graph) error {
	children := childEdgesForValidate(g)
	byKey := packagesByKey(g)

	for _, pl := range linkPlan.Placements {
		if err := validatePlacementPackage(nm, pl, byKey); err != nil {
			return err
		}
	}
	for _, e := range g.Edges {
		if e.From != string(graph.RootImporter) {
			continue
		}
		name := packageNameFromKey(e.To)
		alias := filepath.Join(append([]string{nm}, installSegments(name)...)...)
		if _, err := os.Stat(filepath.Join(alias, "package.json")); err != nil {
			return apperr.Wrap(apperr.Integrity, "app.validate", e.To, err)
		}
	}
	for from, tos := range children {
		if from == string(graph.RootImporter) {
			continue
		}
		if err := validateIsolatedDeps(nm, from, tos, placedDestSet(linkPlan.Placements)); err != nil {
			return err
		}
	}
	return validateIsolatedBoundaries(nm, g, children)
}

func validatePlacementPackage(nm string, pl linker.Placement, byKey map[string]graph.Package) error {
	pkgPath := filepath.Join(pl.DestDir, "package.json")
	st, err := os.Stat(pkgPath)
	if err != nil {
		return apperr.Wrap(apperr.Integrity, "app.validate", pl.Key, err)
	}
	if st.IsDir() {
		return apperr.New(apperr.Integrity, "app.validate", pl.Key, "package.json is a directory")
	}
	pkg, ok := byKey[pl.Key]
	if !ok {
		return apperr.New(apperr.Integrity, "app.validate", pl.Key, "package missing from graph")
	}
	gotName, gotVersion, err := readPackageIdentity(pkgPath)
	if err != nil {
		return apperr.Wrap(apperr.Integrity, "app.validate", pl.Key, err)
	}
	if gotName != pkg.ID.Name || gotVersion != pkg.ID.Version {
		return apperr.New(apperr.Integrity, "app.validate", pl.Key,
			"package.json identity mismatch")
	}
	_ = nm
	return nil
}

func validateReachableDep(nm, from, toKey string, placed map[string][]linker.Placement, byKey map[string]graph.Package) error {
	_ = byKey
	targets, ok := placed[toKey]
	if !ok || len(targets) == 0 {
		return apperr.New(apperr.Integrity, "app.validate.reachable", toKey, "missing placement")
	}
	if from == string(graph.RootImporter) {
		return nil
	}
	parents, ok := placed[from]
	if !ok || len(parents) == 0 {
		return apperr.New(apperr.Integrity, "app.validate.reachable", from, "missing parent placement")
	}
	depName := packageNameFromKey(toKey)
	for _, parent := range parents {
		nestedNM := filepath.Join(parent.DestDir, "node_modules")
		candidate := filepath.Join(append([]string{nestedNM}, installSegments(depName)...)...)
		for _, target := range targets {
			if filepath.Clean(target.DestDir) == filepath.Clean(candidate) {
				return nil
			}
			if isHoistedReachable(parent.DestDir, nm, target.DestDir, depName) {
				return nil
			}
		}
	}
	return apperr.New(apperr.Integrity, "app.validate.reachable", toKey, "not reachable from "+from)
}

func isHoistedReachable(parentDest, nmRoot, targetDest, depName string) bool {
	scope := binNodeModulesForValidate(parentDest)
	if scope == "" {
		scope = nmRoot
	}
	candidate := filepath.Join(append([]string{scope}, installSegments(depName)...)...)
	return filepath.Clean(candidate) == filepath.Clean(targetDest)
}

func binNodeModulesForValidate(pkgInstallDir string) string {
	dir := filepath.Clean(pkgInstallDir)
	for {
		parent := filepath.Dir(dir)
		if filepath.Base(parent) == "node_modules" {
			return parent
		}
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func validateIsolatedDeps(nm, from string, tos []string, placed map[string]struct{}) error {
	privateNM := filepath.Join(nm, ".pnpm", isolated.StoreIDFromKey(from), "node_modules")
	for _, toKey := range tos {
		depName := packageNameFromKey(toKey)
		link := filepath.Join(append([]string{privateNM}, installSegments(depName)...)...)
		if _, ok := placed[toKey]; !ok {
			return apperr.New(apperr.Integrity, "app.validate.isolated", toKey, "missing placement")
		}
		if _, err := os.Stat(filepath.Join(link, "package.json")); err != nil {
			return apperr.Wrap(apperr.Integrity, "app.validate.isolated", toKey, err)
		}
	}
	return nil
}

func placementsByKey(ps []linker.Placement) map[string][]linker.Placement {
	out := map[string][]linker.Placement{}
	for _, p := range ps {
		out[p.Key] = append(out[p.Key], p)
	}
	return out
}

func placedDestSet(ps []linker.Placement) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range ps {
		out[p.Key] = struct{}{}
	}
	return out
}

func childEdgesForValidate(g *graph.Graph) map[string][]string {
	out := map[string][]string{}
	for _, e := range g.Edges {
		out[e.From] = append(out[e.From], e.To)
	}
	return out
}

func packagesByKey(g *graph.Graph) map[string]graph.Package {
	out := map[string]graph.Package{}
	for _, p := range g.Packages {
		out[p.ID.Key()] = p
	}
	return out
}

func validateBinTargets(bins []linker.BinSource) error {
	for _, b := range bins {
		if b.Cmd == "" || b.Target == "" || b.PackageDir == "" {
			return apperr.New(apperr.Integrity, "app.validate.bin", b.Cmd, "incomplete bin source")
		}
		script := filepath.Join(b.PackageDir, filepath.FromSlash(strings.TrimPrefix(b.Target, "./")))
		if _, err := os.Stat(script); err != nil {
			return apperr.Wrap(apperr.Integrity, "app.validate.bin", b.Cmd, err)
		}
	}
	return nil
}

func readPackageIdentity(pkgPath string) (name, version string, err error) {
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", "", err
	}
	var doc struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", "", err
	}
	return doc.Name, doc.Version, nil
}

func validateIsolatedBoundaries(nm string, g *graph.Graph, children map[string][]string) error {
	for from, tos := range children {
		if from == string(graph.RootImporter) {
			continue
		}
		privateNM := filepath.Join(nm, ".pnpm", isolated.StoreIDFromKey(from), "node_modules")
		allowed := map[string]bool{}
		selfName := packageNameFromKey(from)
		allowed[selfName] = true
		if strings.HasPrefix(selfName, "@") {
			if i := strings.Index(selfName, "/"); i > 0 {
				allowed[selfName[:i]] = true
			}
		}
		for _, to := range tos {
			allowed[packageNameFromKey(to)] = true
		}
		entries, err := os.ReadDir(privateNM)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return apperr.Wrap(apperr.IO, "app.validate.boundary", from, err)
		}
		for _, ent := range entries {
			if ent.Name() == ".bin" {
				continue
			}
			if !allowed[ent.Name()] && !strings.HasPrefix(ent.Name(), "@") {
				return apperr.New(apperr.Integrity, "app.validate.boundary", from,
					"unexpected dependency link "+ent.Name())
			}
		}
	}
	return nil
}

func packageNameFromKey(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[:i]
	}
	return s
}

func installSegments(name string) []string {
	if strings.HasPrefix(name, "@") {
		rest := name[1:]
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return []string{"@" + rest[:i], rest[i+1:]}
		}
		return []string{"@" + rest}
	}
	return []string{name}
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

func guardLocalInstall(res *resolver.Resolution) error {
	if res == nil || !resolver.HasLocalSources(res.Extensions) {
		return nil
	}
	return apperr.New(apperr.Install, "app.install", "",
		"local source install not implemented; resolved in lock only")
}
