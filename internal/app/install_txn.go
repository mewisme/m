package app

import (
	"context"
	"encoding/json"
	"errors"
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
func runInstallTxn(ctx context.Context, ac *Context, opts InstallOptions, edit manifestEditFn, prepare mutationPrepareFn) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.install", "", "missing app context")
	}
	if opts.DryRun {
		return runInstallDryRun(ctx, ac, opts, edit, prepare)
	}

	root, err := resolveProjectRoot(ac, "")
	if err != nil {
		return res, err
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		return res, err
	}
	res, err = runInstallInSession(ctx, sess, opts, edit, prepare)
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
		return res, errors.Join(finishErr, installCleanupIncompleteError(res))
	}
	for _, w := range finish.CleanupWarnings {
		if ac != nil && ac.Reporter != nil {
			ac.Reporter.Debug("transaction cleanup warning", diagnostics.Attr{Key: "error", Value: w.Error()})
		}
	}
	if res.StoreMaintenanceRequired && !res.TransactionCleanupIncomplete && !res.RecoveryRequired {
		return res, errors.Join(finishErr, storeMaintenanceIncompleteError(res))
	}
	return res, finishErr
}

func mergeInstallResults(dst, src InstallResult) InstallResult {
	if src.RolledBack {
		dst.RolledBack = true
	}
	if src.RecoveryRequired {
		dst.RecoveryRequired = true
	}
	if src.CleanupIncomplete {
		dst.CleanupIncomplete = true
	}
	if src.TransactionCleanupIncomplete {
		dst.TransactionCleanupIncomplete = true
	}
	if src.StoreCleanupIncomplete {
		dst.StoreCleanupIncomplete = true
	}
	if src.StoreMaintenanceRequired {
		dst.StoreMaintenanceRequired = true
	}
	dst.CleanupWarningCodes, dst.CleanupWarnings = dedupeCleanupPairs(
		append(dst.CleanupWarningCodes, src.CleanupWarningCodes...),
		append(dst.CleanupWarnings, src.CleanupWarnings...),
	)
	return dst
}

func dedupeCleanupPairs(codes, warnings []string) (outCodes, outWarns []string) {
	seen := map[string]bool{}
	for i, code := range codes {
		msg := ""
		if i < len(warnings) {
			msg = warnings[i]
		}
		key := code + "\x00" + msg
		if seen[key] {
			continue
		}
		seen[key] = true
		outCodes = append(outCodes, code)
		outWarns = append(outWarns, msg)
	}
	for i := len(codes); i < len(warnings); i++ {
		msg := warnings[i]
		if msg == "" || seen["\x00"+msg] {
			continue
		}
		seen["\x00"+msg] = true
		outWarns = append(outWarns, msg)
	}
	return outCodes, outWarns
}

// runInstallInSession performs resolve through commit while the session holds the project lock.
func runInstallInSession(ctx context.Context, sess *MutationSession, opts InstallOptions, edit manifestEditFn, prepare mutationPrepareFn) (InstallResult, error) {
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

	if opts.Frozen && !usesStagedSnapshotInputs(opts) {
		if err := validateFrozenLockForProject(ctx, ac, proj); err != nil {
			return res, err
		}
	}

	if prepare != nil {
		if err := prepare(ctx, ac, proj, &opts); err != nil {
			return res, err
		}
	}

	emitPhase(ac, "resolve", "")
	manifestChanged := opts.WriteManifest || len(opts.StagedManifest) > 0
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	var resolution *resolver.Resolution
	if opts.PreResolvedGraph != nil {
		resolution = &resolver.Resolution{Graph: opts.PreResolvedGraph}
	} else {
		resolution, err = resolveForInstall(ctx, ac, proj, opts, manifestChanged)
		if err != nil {
			return res, err
		}
	}
	if err := guardLocalInstall(resolution); err != nil {
		return res, err
	}
	if err := transaction.InvokeTestHook("post_resolve", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "resolve", err)
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))

	stage := txn.StagePath()

	if len(opts.StagedManifest) > 0 {
		if err := os.WriteFile(filepath.Join(stage, "package.json"), opts.StagedManifest, 0o644); err != nil {
			return res, apperr.Wrap(apperr.IO, "app.install", "package.json", err)
		}
		manifestChanged = true
	} else if manifestChanged {
		if err := writeStagedManifest(stage, proj); err != nil {
			return res, err
		}
	}

	emitPhase(ac, "fetch", "")
	extractDir := filepath.Join(stage, "extract")
	stageNM := filepath.Join(stage, "node_modules")
	linkerMode, err := resolveLinkerMode(ctx, ac, proj, opts)
	if err != nil {
		return res, err
	}
	useStore := config.UseGlobalStore(ac.Config)
	fetchOut, err := fetchPackages(ctx, ac, resolution.Graph, extractDir, useStore)
	applyFetchOutcome(&res, fetchOut)
	if err != nil {
		return res, err
	}
	extracts := fetchOut.Extracts
	if err := transaction.InvokeTestHook("post_fetch", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "fetch", err)
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
		return res, err
	}
	if err := lnk.Apply(ctx, linkPlan); err != nil {
		return res, err
	}
	if err := transaction.InvokeTestHook("post_link", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "link", err)
	}
	emitLinkSummary(ac, linkPlan.LinkSummary)
	if linkerMode == "isolated" {
		emitPhase(ac, "link", fmt.Sprintf("linker=isolated packages=%d", len(resolution.Graph.Packages)))
		if err := writeModulesMetadata(stageNM, resolution.Graph); err != nil {
			return res, err
		}
	}

	if len(opts.StagedLock) > 0 {
		if err := os.WriteFile(filepath.Join(stage, lockFileName), opts.StagedLock, 0o644); err != nil {
			return res, apperr.Wrap(apperr.IO, "app.install", lockFileName, err)
		}
	} else if err := writeStagedLock(stage, ac, proj, resolution); err != nil {
		return res, err
	}
	if err := transaction.InvokeTestHook("post_lockfile", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "lockfile", err)
	}
	if useStore {
		if err := writeStagedStoreManifest(stage, resolution.Graph); err != nil {
			return res, err
		}
	}

	snapID := ""
	if !opts.SkipSnapshot {
		snapID, err = stageSnapshot(ctx, stage, proj, resolution)
		if err != nil {
			return res, err
		}
	}

	emitPhase(ac, "validate", "")
	if err := validateStaged(stage, linkPlan, resolution.Graph, linkerMode); err != nil {
		return res, err
	}
	if err := txn.SetState(transaction.StateValidated); err != nil {
		return res, err
	}
	if err := transaction.InvokeTestHook("post_validate", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "validate", err)
	}

	plan := buildCommitPlan(commitPlanInput{manifestChanged: manifestChanged, useStore: useStore, snapshotID: snapID})
	if err := txn.SetPlan(plan); err != nil {
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
			return res, err
		}
	}

	emitPhase(ac, "commit", "")
	if err := txn.Commit(ctx, nil); err != nil {
		emitPhase(ac, "rollback", "")
		return res, err
	}

	if err := pruneSnapshots(ac, proj); err != nil {
		if ac != nil && ac.Reporter != nil {
			ac.Reporter.Debug("snapshot prune failed", diagnostics.Attr{Key: "error", Value: err.Error()})
		}
	}

	return res, nil
}

func runInstallDryRun(ctx context.Context, ac *Context, opts InstallOptions, edit manifestEditFn, prepare mutationPrepareFn) (InstallResult, error) {
	var res InstallResult
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return res, err
	}
	if opts.Frozen && !usesStagedSnapshotInputs(opts) {
		if err := validateFrozenLockForProject(ctx, ac, proj); err != nil {
			return res, err
		}
	}
	if prepare != nil {
		if err := prepare(ctx, ac, proj, &opts); err != nil {
			return res, err
		}
	}
	emitPhase(ac, "resolve", "")
	manifestChanged := opts.WriteManifest || len(opts.StagedManifest) > 0
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	var resolution *resolver.Resolution
	if opts.PreResolvedGraph != nil {
		resolution = &resolver.Resolution{Graph: opts.PreResolvedGraph}
	} else {
		resolution, err = resolveForInstall(ctx, ac, proj, opts, manifestChanged)
		if err != nil {
			return res, err
		}
	}
	if err := guardLocalInstall(resolution); err != nil {
		return res, err
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))
	if p, err := BuildMutationPlan(resolution.Graph); err == nil {
		res.Plan = p
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
	settings, err := mlock.SettingsWithFingerprints(ac.Config, proj.Normalized.Overrides)
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
			depName := edgeNameFor(g, from, toKey)
			if err := validateReachableDep(nm, from, toKey, depName, placed, byKey); err != nil {
				return err
			}
		}
	}
	return validateBinTargets(linkPlan.Bins)
}

func validateStagedIsolated(nm string, linkPlan *linker.Plan, g *graph.Graph) error {
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
		name := e.Name
		if name == "" {
			name = packageNameFromKey(e.To)
		}
		alias := filepath.Join(append([]string{nm}, installSegments(name)...)...)
		if _, err := os.Stat(filepath.Join(alias, "package.json")); err != nil {
			return apperr.Wrap(apperr.Integrity, "app.validate", e.To, err)
		}
	}
	for _, e := range g.Edges {
		if e.From == string(graph.RootImporter) {
			continue
		}
		if err := validateIsolatedDep(nm, e, placedDestSet(linkPlan.Placements)); err != nil {
			return err
		}
	}
	return validateIsolatedBoundaries(nm, g)
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

func validateReachableDep(nm, from, toKey, depName string, placed map[string][]linker.Placement, byKey map[string]graph.Package) error {
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

func validateIsolatedDep(nm string, e graph.Edge, placed map[string]struct{}) error {
	privateNM := filepath.Join(nm, ".pnpm", isolated.StoreIDFromKey(e.From), "node_modules")
	depName := e.Name
	if depName == "" {
		depName = packageNameFromKey(e.To)
	}
	link := filepath.Join(append([]string{privateNM}, installSegments(depName)...)...)
	if _, ok := placed[e.To]; !ok {
		return apperr.New(apperr.Integrity, "app.validate.isolated", e.To, "missing placement")
	}
	if _, err := os.Stat(filepath.Join(link, "package.json")); err != nil {
		return apperr.Wrap(apperr.Integrity, "app.validate.isolated", e.To, err)
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

func validateIsolatedBoundaries(nm string, g *graph.Graph) error {
	for from := range childEdgesForValidate(g) {
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
		for _, e := range g.Edges {
			if e.From != from {
				continue
			}
			name := e.Name
			if name == "" {
				name = packageNameFromKey(e.To)
			}
			allowed[name] = true
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

func edgeNameFor(g *graph.Graph, from, toKey string) string {
	if g != nil {
		for _, e := range g.Edges {
			if e.From == from && e.To == toKey {
				if e.Name != "" {
					return e.Name
				}
				break
			}
		}
	}
	return packageNameFromKey(toKey)
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
