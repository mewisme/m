package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/linker/isolated"
	"github.com/mewisme/mew/internal/linker/planner"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/manifest"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/resolver"
	"github.com/mewisme/mew/internal/snapshot"
	"github.com/mewisme/mew/internal/transaction"
	"github.com/mewisme/mew/internal/workspace"
)

// manifestEditFn applies in-memory manifest changes before staging (no live write).
type manifestEditFn func(*project.Project) error

type commitPlanInput struct {
	manifestChanged bool
	useStore        bool
	snapshotID      string
	memberManifests []string
	lockBasename    string
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
	populateWarningCleanup(&res, finish)
	for _, w := range finish.WarningErrors() {
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
	if opts.CleanNodeModules {
		if err := cleanLiveNodeModules(txn, proj.Root); err != nil {
			return res, err
		}
	}

	if prepare != nil {
		if err := prepare(ctx, ac, proj, &opts); err != nil {
			return res, err
		}
	}
	if err := requireWorkspacesGate(ac, opts); err != nil {
		return res, err
	}
	if err := validatePnpmLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := validateNpmLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := validateBunLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := rejectBunLockbIfPresent(proj); err != nil {
		return res, err
	}
	if err := validateYarnLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := gateYarnPnPInstall(proj); err != nil {
		return res, err
	}

	emitPhase(ac, "resolve", "")
	phaseResolve := startInstallPhase(ac, "resolve")
	defer phaseResolve.done()
	manifestChanged := opts.WriteManifest || len(opts.StagedManifest) > 0 || len(opts.MemberEdits) > 0 || len(opts.StagedMemberManifests) > 0
	if !manifestChanged {
		drift, driftErr := manifestDriftsFromLock(ctx, proj)
		if driftErr != nil {
			return res, driftErr
		}
		if drift {
			manifestChanged = true
		}
	}
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	var resolution *resolver.Resolution
	if opts.PreResolvedGraph != nil {
		resolution = &resolver.Resolution{Graph: opts.PreResolvedGraph}
		if len(opts.StagedLock) > 0 {
			if lockDoc, decodeErr := mlock.Decode(opts.StagedLock); decodeErr == nil {
				resolution.Extensions = lockDoc.Extensions
			}
		}
	} else {
		resolution, err = resolveForInstall(ctx, ac, proj, opts, manifestChanged)
		if err != nil {
			return res, err
		}
	}
	if err := guardLocalInstall(ac, resolution); err != nil {
		return res, err
	}
	if workspace.Enabled(ac.Config) && len(opts.Filter) > 0 {
		prior, priorErr := readLockHints(ctx, ac, proj)
		if priorErr != nil {
			return res, priorErr
		}
		if prior != nil {
			var priorExt lockfile.Extensions
			if proj.Identity == project.IdentityMew {
				if priorDoc, readErr := readLockDocument(proj.Root, proj.Identity); readErr == nil && priorDoc != nil {
					priorExt = priorDoc.Extensions
				}
			}
			untouched := untouchedImporterIDs(prior, resolution.Graph)
			merged, mergeErr := mergeFilteredWorkspaceResolution(prior, priorExt, resolution, untouched)
			if mergeErr != nil {
				return res, mergeErr
			}
			resolution = merged
		}
	}
	if err := transaction.InvokeTestHook("post_resolve", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "resolve", err)
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))

	stage := txn.StagePath()
	lockName := IncumbentLockBasename(proj)

	if len(opts.StagedManifest) > 0 {
		if err := os.WriteFile(filepath.Join(stage, "package.json"), opts.StagedManifest, 0o644); err != nil {
			return res, apperr.Wrap(apperr.IO, "app.install", "package.json", err)
		}
		manifestChanged = true
	} else if manifestChanged {
		if err := writeStagedManifests(stage, proj, opts); err != nil {
			return res, err
		}
	}
	if len(opts.StagedMemberManifests) > 0 {
		if err := writeStagedMemberManifests(stage, opts.StagedMemberManifests); err != nil {
			return res, err
		}
		manifestChanged = true
	}

	emitPhase(ac, "fetch", "")
	phaseFetch := startInstallPhase(ac, "fetch")
	defer phaseFetch.done()
	extractDir := filepath.Join(stage, "extract")
	stageNM := filepath.Join(stage, "node_modules")
	linkerMode, err := resolveLinkerMode(ctx, ac, proj, opts)
	if err != nil {
		return res, err
	}
	useStore := config.UseGlobalStore(ac.Config)
	localRoot := proj.Root
	if len(opts.StagedManifest) > 0 || len(opts.StagedMemberManifests) > 0 {
		// ponytail: stage overlay supplies workspace locals until commit publishes manifests.
		localRoot = stage
	}
	localExtracts, err := buildLocalExtractDirs(localRoot, resolution, resolution.Graph)
	if err != nil {
		return res, err
	}
	if config.Bool(ac.Config, "offline", false) {
		report, preErr := PreflightOffline(ctx, ac, proj, resolution.Graph, resolution.Extensions, localExtracts)
		if preErr != nil {
			return res, preErr
		}
		if !report.OK() {
			return res, offlinePreflightError(report)
		}
	}
	fetchOut, err := fetchPackages(ctx, ac, proj, resolution.Graph, resolution.Extensions, extractDir, useStore, localExtracts)
	applyFetchOutcome(&res, fetchOut)
	if err != nil {
		return res, err
	}
	var storeRoot string
	if useStore {
		storeRoot, err = config.StoreRoot(ac.Config)
		if err != nil {
			return res, err
		}
	}
	if err := stagePatchDerivatives(ctx, stage, storeRoot, resolution.Extensions, fetchOut.Extracts); err != nil {
		return res, err
	}
	if err := applyPatchesToExtracts(ctx, resolution.Graph, resolution.Extensions, fetchOut.Extracts); err != nil {
		return res, err
	}
	extracts := fetchOut.Extracts
	if err := transaction.InvokeTestHook("post_fetch", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "fetch", err)
	}

	emitPhase(ac, "link", "")
	phaseLink := startInstallPhase(ac, "link")
	defer phaseLink.done()
	if err := os.MkdirAll(stageNM, 0o755); err != nil {
		return res, apperr.Wrap(apperr.IO, "app.install", stageNM, err)
	}
	var caps planner.Capabilities
	if linkerMode == "isolated" {
		caps, _ = planner.ProbeCached(config.CacheRoot(ac.Config), extractDir, stageNM)
	} else if useStore {
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
	graphDigest, err := snapshot.GraphDigest(resolution.Graph)
	if err != nil {
		return res, err
	}
	genBind, err := publishLinkBinMetadata(stage, linkPlan, linkerMode, graphDigest)
	if err != nil {
		return res, err
	}
	if err := WriteGenerationBinding(stage, genBind); err != nil {
		return res, err
	}

	if len(opts.StagedLock) > 0 {
		if err := os.WriteFile(filepath.Join(stage, lockName), opts.StagedLock, 0o644); err != nil {
			return res, apperr.Wrap(apperr.IO, "app.install", lockName, err)
		}
	} else if err := writeStagedLock(stage, ac, proj, resolution, opts); err != nil {
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
		snapID, err = stageSnapshot(ctx, stage, proj, resolution, opts)
		if err != nil {
			return res, err
		}
	}

	if err := runLifecyclePhase(ctx, ac, proj, opts, stageNM, resolution.Graph, linkPlan); err != nil {
		return res, err
	}

	emitPhase(ac, "validate", "")
	phaseValidate := startInstallPhase(ac, "validate")
	defer phaseValidate.done()
	if err := EnforceInstallPolicy(ctx, ac, resolution.Graph, linkPlan); err != nil {
		return res, err
	}
	if err := validateStaged(stage, linkPlan, resolution.Graph, linkerMode); err != nil {
		return res, err
	}
	if err := txn.SetState(transaction.StateValidated); err != nil {
		return res, err
	}
	if err := transaction.InvokeTestHook("post_validate", 0); err != nil {
		return res, apperr.Wrap(apperr.Transaction, "app.install", "validate", err)
	}

	plan := buildCommitPlan(commitPlanInput{
		manifestChanged: manifestChanged,
		useStore:        useStore,
		snapshotID:      snapID,
		memberManifests: allMemberManifestPaths(opts),
		lockBasename:    lockName,
	})
	if err := txn.SetPlan(plan); err != nil {
		return res, err
	}

	backupPaths := []string{lockName, "node_modules"}
	if manifestChanged {
		backupPaths = append([]string{"package.json"}, backupPaths...)
		backupPaths = append(backupPaths, allMemberManifestPaths(opts)...)
	}
	backupPaths = append(backupPaths, filepath.Join(".mew", "generation.json"))
	if useStore {
		backupPaths = append(backupPaths, filepath.Join(".mew", "store-manifest.json"))
	}
	backupPaths = append(backupPaths, filepath.Join(".mew", "snapshots", "index.json"))

	for _, rel := range backupPaths {
		if opts.CleanNodeModules && rel == "node_modules" && txn.HasBackup(rel) {
			continue
		}
		if err := txn.RecordBackup(rel); err != nil {
			emitPhase(ac, "rollback", rel)
			return res, err
		}
	}

	emitPhase(ac, "commit", "")
	phaseCommit := startInstallPhase(ac, "commit")
	defer phaseCommit.done()
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
	if err := requireWorkspacesGate(ac, opts); err != nil {
		return res, err
	}
	if err := validatePnpmLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := validateNpmLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := validateBunLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := rejectBunLockbIfPresent(proj); err != nil {
		return res, err
	}
	if err := validateYarnLockBeforeTxn(proj); err != nil {
		return res, err
	}
	if err := gateYarnPnPInstall(proj); err != nil {
		return res, err
	}
	emitPhase(ac, "resolve", "")
	phaseResolve := startInstallPhase(ac, "resolve")
	defer phaseResolve.done()
	manifestChanged := opts.WriteManifest || len(opts.StagedManifest) > 0 || len(opts.MemberEdits) > 0 || len(opts.StagedMemberManifests) > 0
	if !manifestChanged {
		drift, driftErr := manifestDriftsFromLock(ctx, proj)
		if driftErr != nil {
			return res, driftErr
		}
		if drift {
			manifestChanged = true
		}
	}
	if edit != nil {
		if err := edit(proj); err != nil {
			return res, err
		}
	}
	priorKeys, _ := priorPackageKeys(ctx, ac, proj)
	var resolution *resolver.Resolution
	if opts.PreResolvedGraph != nil {
		resolution = &resolver.Resolution{Graph: opts.PreResolvedGraph}
		if len(opts.StagedLock) > 0 {
			if lockDoc, decodeErr := mlock.Decode(opts.StagedLock); decodeErr == nil {
				resolution.Extensions = lockDoc.Extensions
			}
		}
	} else {
		resolution, err = resolveForInstall(ctx, ac, proj, opts, manifestChanged)
		if err != nil {
			return res, err
		}
	}
	if err := guardLocalInstall(ac, resolution); err != nil {
		return res, err
	}
	res = diffKeys(priorKeys, packageKeysFromGraph(resolution.Graph))
	if p, err := BuildMutationPlan(MutationPlanInput{
		PriorKeys:     priorKeys,
		Graph:         resolution.Graph,
		IgnoreScripts: opts.IgnoreScripts,
		AC:            ac,
	}); err == nil {
		res.Plan = p
	}
	return res, nil
}

func buildCommitPlan(in commitPlanInput) []transaction.Op {
	var ops []transaction.Op
	lockName := in.lockBasename
	if lockName == "" {
		lockName = "m.lock"
	}
	if in.manifestChanged {
		ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: "package.json", Backup: "stage/package.json"})
		for _, rel := range in.memberManifests {
			ops = append(ops, transaction.Op{
				Kind:   transaction.OpRename,
				Path:   rel,
				Backup: filepath.Join("stage", rel),
			})
		}
	}
	ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: lockName, Backup: filepath.Join("stage", lockName)})
	ops = append(ops, transaction.Op{Kind: transaction.OpRename, Path: "node_modules", Backup: "stage/node_modules"})
	ops = append(ops,
		transaction.Op{Kind: transaction.OpMkdir, Path: ".mew"},
		transaction.Op{Kind: transaction.OpRename, Path: filepath.Join(".mew", "generation.json"), Backup: filepath.Join("stage", ".mew", "generation.json")},
	)
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

func stageSnapshot(ctx context.Context, stage string, proj *project.Project, res *resolver.Resolution, opts InstallOptions) (string, error) {
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
	lockName := IncumbentLockBasename(proj)
	lockBytes, err := os.ReadFile(filepath.Join(stage, lockName))
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "app.snapshot", lockName, err)
	}
	members, err := collectSnapshotMemberManifests(stage, proj.Root, proj.Identity, opts, lockBytes)
	if err != nil {
		return "", err
	}
	digest, err := snapshot.GraphDigest(res.Graph)
	if err != nil {
		return "", err
	}
	stageSnap := filepath.Join(stage, "snapshots")
	if err := store.StageCreate(stageSnap, id, manifest, lockBytes, digest, members); err != nil {
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

func writeStagedManifests(stage string, proj *project.Project, opts InstallOptions) error {
	if err := writeStagedManifest(stage, proj); err != nil {
		return err
	}
	for memPath, doc := range opts.MemberEdits {
		dest := filepath.Join(stage, filepath.FromSlash(memPath), "package.json")
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "app.install", dest, err)
		}
		if err := doc.Write(dest); err != nil {
			return err
		}
	}
	return nil
}

func memberManifestPaths(opts InstallOptions) []string {
	if len(opts.MemberEdits) == 0 {
		return nil
	}
	out := make([]string, 0, len(opts.MemberEdits))
	for memPath := range opts.MemberEdits {
		out = append(out, filepath.Join(memPath, "package.json"))
	}
	sort.Strings(out)
	return out
}

func allMemberManifestPaths(opts InstallOptions) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, rel := range memberManifestPaths(opts) {
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}
	for rel := range opts.StagedMemberManifests {
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

func writeStagedMemberManifests(stage string, members map[string][]byte) error {
	for rel, data := range members {
		if _, err := snapshot.ParseMemberManifestPath(rel); err != nil {
			return err
		}
		dest := filepath.Join(stage, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "app.install", dest, err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return apperr.Wrap(apperr.IO, "app.install", dest, err)
		}
	}
	return nil
}

func collectSnapshotMemberManifests(stage, projRoot string, id project.Identity, opts InstallOptions, lockBytes []byte) (map[string][]byte, error) {
	out := map[string][]byte{}
	add := func(rel string) error {
		if _, err := snapshot.ParseMemberManifestPath(rel); err != nil {
			return err
		}
		if _, ok := out[rel]; ok {
			return nil
		}
		staged := filepath.Join(stage, filepath.FromSlash(rel))
		if data, err := os.ReadFile(staged); err == nil {
			out[rel] = data
			return nil
		}
		live := filepath.Join(projRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(live)
		if err != nil {
			return apperr.Wrap(apperr.IO, "app.snapshot", rel, err)
		}
		out[filepath.ToSlash(rel)] = data
		return nil
	}
	for _, rel := range memberManifestPaths(opts) {
		if err := add(rel); err != nil {
			return nil, err
		}
	}
	if id == project.IdentityMew {
		doc, err := mlock.Decode(lockBytes)
		if err != nil {
			return nil, err
		}
		for _, im := range doc.Importers {
			if im.ID == graph.RootImporter {
				continue
			}
			rel := filepath.Join(string(im.ID), "package.json")
			if err := add(rel); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func writeStagedLock(stage string, ac *Context, proj *project.Project, res *resolver.Resolution, opts InstallOptions) error {
	lockName := IncumbentLockBasename(proj)
	stagePath := filepath.Join(stage, lockName)
	switch proj.Identity {
	case project.IdentityMew:
		return writeStagedMlock(stagePath, ac, proj, res, opts)
	case project.IdentityNub, project.IdentityPNPM, project.IdentityNPM, project.IdentityBun, project.IdentityYarn:
		return writeStagedExtLock(stagePath, proj, res, opts)
	default:
		return lockfile.NewUnsupported("app.install", lockName, "lock adapter not implemented for identity")
	}
}

// writeStagedExtLockTestHook simulates encode/stage failures in tests.
var writeStagedExtLockTestHook func() error

// SetWriteStagedExtLockTestHook installs a test-only failure hook.
func SetWriteStagedExtLockTestHook(fn func() error) {
	writeStagedExtLockTestHook = fn
}

func writeStagedExtLock(stagePath string, proj *project.Project, res *resolver.Resolution, opts InstallOptions) error {
	if writeStagedExtLockTestHook != nil {
		if err := writeStagedExtLockTestHook(); err != nil {
			return err
		}
	}
	prior, err := readExtLockPrior(proj)
	if err != nil {
		return err
	}
	ext, ok := lockfile.ExtAdapterFor(proj.Identity)
	if !ok {
		return lockfile.NewUnsupported("app.install", project.IncumbentLockBasename(proj.Root, proj.Identity), "adapter not registered")
	}
	livePath := LockPath(proj)
	var extensions lockfile.Extensions
	if _, extData, readErr := ext.ReadWithExtensions(context.Background(), livePath); readErr == nil {
		extensions = extData
	}
	if res != nil && len(res.Extensions) > 0 {
		if extensions == nil {
			extensions = lockfile.Extensions{}
		}
		for k, v := range res.Extensions {
			extensions[k] = v
		}
	}
	det := lockfile.Detection{}
	if proj.Identity == project.IdentityPNPM {
		det, err = detectPnpmLock(prior, proj, opts.PnpmMajor)
		if err != nil {
			return err
		}
	}
	if proj.Identity == project.IdentityNPM {
		det, err = detectNpmLock(prior)
		if err != nil {
			return err
		}
	}
	if proj.Identity == project.IdentityBun {
		det, err = detectBunLock(prior)
		if err != nil {
			return err
		}
	}
	if proj.Identity == project.IdentityYarn {
		det, err = detectYarnLock(prior, proj.Root)
		if err != nil {
			return err
		}
	}
	out, err := lockfile.EncodePreserving(context.Background(), ext, livePath, res.Graph, prior, extensions, det)
	if err != nil {
		return err
	}
	data := out.Bytes
	if out.Unchanged {
		data = prior
	}
	return os.WriteFile(stagePath, data, 0o644)
}

func writeStagedMlock(stagePath string, ac *Context, proj *project.Project, res *resolver.Resolution, opts InstallOptions) error {
	settings, err := mlock.SettingsWithFingerprints(ac.Config, proj.Normalized.Overrides)
	if err != nil {
		return err
	}
	specs, err := buildImporterSpecifiers(proj, res, opts.MemberEdits)
	if err != nil {
		return err
	}
	if workspace.Enabled(ac.Config) && len(opts.Filter) > 0 {
		if priorDoc, readErr := readLockDocument(proj.Root, proj.Identity); readErr == nil && priorDoc != nil {
			mergePriorImporterSpecs(specs, priorDoc, res.Graph)
		}
	}
	doc, err := mlock.FromResolution(res, specs, settings)
	if err != nil {
		return err
	}
	if workspace.Enabled(ac.Config) && len(opts.Filter) > 0 {
		if priorDoc, readErr := readLockDocument(proj.Root, proj.Identity); readErr == nil && priorDoc != nil {
			mergePriorImporterSections(doc, priorDoc, importerIDsInGraph(res.Graph))
		}
	}
	return mlock.WriteAtomic(stagePath, doc)
}

func mergePriorImporterSections(doc *mlock.Document, prior *mlock.Document, active map[graph.ImporterID]bool) {
	if doc == nil || prior == nil {
		return
	}
	have := map[graph.ImporterID]bool{}
	for _, im := range doc.Importers {
		have[im.ID] = true
	}
	for _, sec := range prior.Importers {
		if active[sec.ID] || have[sec.ID] {
			continue
		}
		doc.Importers = append(doc.Importers, sec)
		have[sec.ID] = true
	}
	sort.Slice(doc.Importers, func(i, j int) bool { return doc.Importers[i].ID < doc.Importers[j].ID })
}

func buildImporterSpecifiers(proj *project.Project, res *resolver.Resolution, memberEdits map[string]*manifest.Document) (map[graph.ImporterID][]mlock.Specifier, error) {
	specs := map[graph.ImporterID][]mlock.Specifier{
		graph.RootImporter: mlock.SpecifiersFromManifest(proj.Normalized),
	}
	if res == nil || res.Graph == nil {
		return specs, nil
	}
	for _, im := range res.Graph.Importers {
		if im.ID == graph.RootImporter {
			continue
		}
		memPath := string(im.ID)
		if doc, ok := memberEdits[memPath]; ok {
			norm, err := manifest.ToNormalized(doc)
			if err != nil {
				return nil, err
			}
			specs[im.ID] = mlock.SpecifiersFromManifest(norm)
			continue
		}
		doc, err := manifest.Load(filepath.Join(proj.Root, filepath.FromSlash(memPath), "package.json"))
		if err != nil {
			return nil, apperr.Wrap(apperr.Manifest, "app.install", memPath, err)
		}
		norm, err := manifest.ToNormalized(doc)
		if err != nil {
			return nil, err
		}
		specs[im.ID] = mlock.SpecifiersFromManifest(norm)
	}
	return specs, nil
}

func mergePriorImporterSpecs(specs map[graph.ImporterID][]mlock.Specifier, prior *mlock.Document, g *graph.Graph) {
	if prior == nil || g == nil {
		return
	}
	active := importerIDsInGraph(g)
	for _, sec := range prior.Importers {
		if active[sec.ID] {
			continue
		}
		specs[sec.ID] = append([]mlock.Specifier(nil), sec.Specifiers...)
	}
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
		if err := validateIsolatedDep(nm, g, linkPlan, e); err != nil {
			return err
		}
	}
	return validateIsolatedBoundaries(nm, g)
}

func installedVersionMatches(installed, graphVersion string) bool {
	return registryBaseVersion(installed) == registryBaseVersion(graphVersion)
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
	if gotName != pkg.ID.Name || !installedVersionMatches(gotVersion, pkg.ID.Version) {
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

func validateIsolatedDep(nm string, g *graph.Graph, plan *linker.Plan, e graph.Edge) error {
	if _, ok := placedDestSet(plan.Placements)[e.To]; !ok {
		return apperr.New(apperr.Integrity, "app.validate.isolated", e.To, "missing placement")
	}
	layout, err := isolated.ComputeLayout(g, nm)
	if err != nil {
		return apperr.Wrap(apperr.Integrity, "app.validate.isolated", e.To, err)
	}
	link := isolated.DepLinkPath(nm, g, layout.Packages, e)
	if link == "" {
		return apperr.New(apperr.Integrity, "app.validate.isolated", e.From, "missing parent private node_modules")
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

type installPhaseTimer struct {
	ac    *Context
	phase string
	start time.Time
}

func startInstallPhase(ac *Context, phase string) installPhaseTimer {
	return installPhaseTimer{ac: ac, phase: phase, start: time.Now()}
}

func (t installPhaseTimer) done() {
	if t.ac == nil || t.ac.Reporter == nil {
		return
	}
	elapsed := time.Since(t.start).Round(time.Millisecond)
	t.ac.Reporter.Debug("install phase",
		diagnostics.Attr{Key: "phase", Value: t.phase},
		diagnostics.Attr{Key: "elapsed", Value: elapsed.String()},
	)
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

func guardLocalInstall(ac *Context, res *resolver.Resolution) error {
	if res == nil {
		return nil
	}
	if resolver.HasGitSources(res.Extensions) {
		return nil
	}
	if !resolver.HasLocalSources(res.Extensions) {
		return nil
	}
	locals, err := resolver.DecodeLocalSources(res.Extensions)
	if err != nil {
		return err
	}
	workspacesOn := ac != nil && ac.Config != nil && workspace.Enabled(ac.Config)
	for key, src := range locals {
		switch src.Protocol {
		case "workspace":
			if workspacesOn {
				continue
			}
		case "file", "portal", "tarball", "link":
			continue
		}
		return apperr.New(apperr.Install, "app.install", key,
			"local source install not implemented; resolved in lock only")
	}
	return nil
}

// cleanLiveNodeModules journals and removes the live node_modules tree (m ci).
func cleanLiveNodeModules(txn *transaction.Runner, projectRoot string) error {
	nm := filepath.Join(projectRoot, "node_modules")
	if _, err := os.Stat(nm); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "app.ci", "node_modules", err)
	}
	if err := txn.RecordBackup("node_modules"); err != nil {
		return err
	}
	if err := os.RemoveAll(nm); err != nil {
		return apperr.Wrap(apperr.IO, "app.ci", "node_modules", err)
	}
	return nil
}
