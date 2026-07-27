package transaction

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fsx"
)

const (
	currentName = "current"
	backupsDir  = "backups"
	stageDir    = "stage"
)

// Runner manages a project-local install transaction under .mew/txn/<id>/.
type Runner struct {
	ProjectRoot string
	ID          string
	Root        string
	doc         *Document
	journalGen  int
}

// NewRunner prepares a runner for projectRoot (does not create dirs until Begin).
func NewRunner(projectRoot string) *Runner {
	return &Runner{ProjectRoot: projectRoot}
}

// TxnRoot returns <project>/.mew/txn.
func TxnRoot(projectRoot string) string {
	return filepath.Join(projectRoot, ".mew", "txn")
}

// CurrentPath returns the active transaction pointer file.
func CurrentPath(projectRoot string) string {
	return filepath.Join(TxnRoot(projectRoot), currentName)
}

// StagePath returns the staging directory for this transaction.
func (r *Runner) StagePath() string {
	return filepath.Join(r.Root, stageDir)
}

// Begin creates .mew/txn/<id>/, writes journal state=staging, and sets current.
func (r *Runner) Begin(ctx context.Context) error {
	run, err := BeginMutation(ctx, r.ProjectRoot)
	if err != nil {
		return err
	}
	r.ID = run.ID
	r.Root = run.Root
	r.doc = run.doc
	r.journalGen = run.journalGen
	return nil
}

// SetState updates journal state and persists.
func (r *Runner) SetState(state string) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.state", "", "transaction not begun")
	}
	r.doc.State = state
	return r.saveJournal()
}

// SetPlan records ordered forward ops before the first live mutation.
func (r *Runner) SetPlan(plan []Op) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.plan", "", "transaction not begun")
	}
	cp := make([]Op, len(plan))
	for i, op := range plan {
		cp[i] = op
		if cp[i].Progress == "" {
			cp[i].Progress = ProgressPending
		}
		if cp[i].Phase == "" {
			cp[i].Phase = PhasePending
		}
	}
	r.doc.Plan = cp
	return r.saveJournal()
}

// RecordBackup copies a live project-relative path into backups/ and journals the op.
func (r *Runner) RecordBackup(rel string) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.backup", rel, "transaction not begun")
	}
	live, err := GuardPath(r.ProjectRoot, rel)
	if err != nil {
		return err
	}
	kind, target, hadPrior, err := pathKind(live)
	if err != nil {
		return err
	}
	op := Op{
		Kind:      OpBackup,
		Path:      rel,
		HadPrior:  hadPrior,
		PriorKind: kind,
		Progress:  ProgressApplied,
	}
	if !hadPrior {
		r.doc.Ops = append(r.doc.Ops, op)
		return r.saveJournal()
	}
	backupRel := filepath.Join(backupsDir, sanitizeBackupName(rel))
	backupAbs := filepath.Join(r.Root, backupRel)
	op.Backup = backupRel
	op.DestKind = kind
	op.SymlinkTarget = target
	if kind == DestKindSymlink || kind == DestKindJunction {
		if err := os.WriteFile(backupAbs+".link", []byte(target), 0o644); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.backup", rel, err)
		}
		op.Backup = backupRel + ".link"
	} else if err := backupTree(live, backupAbs); err != nil {
		return err
	}
	r.doc.Ops = append(r.doc.Ops, op)
	if err := r.saveJournal(); err != nil {
		return err
	}
	r.markPlanPhase(rel, PhaseBackupReady)
	if err := r.saveJournal(); err != nil {
		return err
	}
	return invokeTestHook("backup", len(r.doc.Ops)-1)
}

// Commit applies the forward plan; marks committed only after the last op succeeds.
func (r *Runner) Commit(ctx context.Context, extra []Op) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.commit", "", "transaction not begun")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	plan := append(append([]Op{}, r.doc.Plan...), extra...)
	if len(plan) == 0 && r.doc.SchemaVersion < SchemaVersion {
		// v1 callers passed all forward ops via extra.
		plan = extra
	}
	if len(plan) > 0 && len(r.doc.Plan) == 0 {
		r.doc.Plan = plan
	}
	if err := r.SetState(StateCommitting); err != nil {
		return err
	}
	for i := range r.doc.Plan {
		if err := invokeTestHook("publish", i); err != nil {
			_, _ = r.Rollback(ctx)
			return apperr.Wrap(apperr.Transaction, "transaction.commit", r.doc.Plan[i].Path, err)
		}
		if err := invokeTestHook("commit", i); err != nil {
			_, _ = r.Rollback(ctx)
			return apperr.Wrap(apperr.Transaction, "transaction.commit", r.doc.Plan[i].Path, err)
		}
		if err := r.applyPlanOp(ctx, i, true); err != nil {
			_, _ = r.Rollback(ctx)
			return err
		}
	}
	if err := invokeTestHook("pre_committed", 0); err != nil {
		_, _ = r.Rollback(ctx)
		return apperr.Wrap(apperr.Transaction, "transaction.commit", "", err)
	}
	r.doc.State = StateCommitted
	if err := r.saveJournal(); err != nil {
		return err
	}
	return invokeTestHook("committed", 0)
}

// Rollback restores backups and inverses applied forward ops.
func (r *Runner) Rollback(ctx context.Context) (FinishResult, error) {
	var fr FinishResult
	if r.doc == nil {
		return fr, nil
	}
	if err := ctx.Err(); err != nil {
		return fr, err
	}
	if err := invokeTestHook("rollback", 0); err != nil {
		return fr, apperr.Wrap(apperr.Transaction, "transaction.rollback", "", err)
	}
	_ = repairPartialNodeModules(r.ProjectRoot)
	for i := len(r.doc.Plan) - 1; i >= 0; i-- {
		op := r.doc.Plan[i]
		if op.Progress != ProgressApplied && op.Progress != ProgressApplying {
			continue
		}
		if err := invokeTestHook("rollback", i+1); err != nil {
			return fr, apperr.Wrap(apperr.Transaction, "transaction.rollback", op.Path, err)
		}
		if err := r.rollbackPlanOp(ctx, i); err != nil {
			return fr, err
		}
	}
	for i := len(r.doc.Ops) - 1; i >= 0; i-- {
		op := r.doc.Ops[i]
		if op.Kind != OpBackup {
			continue
		}
		if r.planHasAppliedPath(op.Path) {
			continue
		}
		if err := r.restoreBackup(op); err != nil {
			return fr, err
		}
	}
	r.doc.State = StateAborted
	if err := r.saveJournal(); err != nil {
		return fr, err
	}
	fr = r.finishCriticalCleanup()
	return fr, nil
}

// Finish clears the current pointer and optionally removes the txn dir.
func (r *Runner) Finish(keepJournal bool) FinishResult {
	if r.doc == nil {
		return FinishResult{}
	}
	fr := r.finishCleanup(keepJournal)
	if r.doc.State == StateCommitted {
		fr.Committed = true
	}
	return fr
}

// Discard removes an incomplete transaction without restoring backups.
func (r *Runner) Discard() FinishResult {
	if r.doc == nil {
		return FinishResult{}
	}
	return r.finishCleanup(false)
}

func (r *Runner) finishCriticalCleanup() FinishResult {
	var fr FinishResult
	if err := clearCurrent(r.ProjectRoot); err != nil {
		fr.CleanupWarnings = append(fr.CleanupWarnings, err)
	} else {
		fr.CurrentCleared = true
	}
	if err := ReleaseProjectLock(r.ProjectRoot, r.ID); err != nil {
		fr.CleanupWarnings = append(fr.CleanupWarnings, err)
	} else {
		fr.LockReleased = true
	}
	return fr
}

func (r *Runner) finishCleanup(keepJournal bool) FinishResult {
	if err := invokeTestHook("finish", 0); err != nil {
		fr := r.finishCriticalCleanup()
		appendCleanupWarning(&fr, "finish_hook", apperr.Wrap(apperr.Transaction, "transaction.finish", "", err))
		return fr
	}
	fr := r.finishCriticalCleanup()
	if !keepJournal && r.Root != "" {
		if err := os.RemoveAll(r.Root); err != nil {
			appendCleanupWarning(&fr, "txn_dir_remove", apperr.Wrap(apperr.IO, "transaction.finish", r.Root, err))
		}
	}
	return fr
}

// Document returns the current journal (read-only copy).
func (r *Runner) Document() *Document {
	if r.doc == nil {
		return nil
	}
	cp := *r.doc
	cp.Ops = append([]Op(nil), r.doc.Ops...)
	cp.Plan = append([]Op(nil), r.doc.Plan...)
	return &cp
}

// LoadIncomplete returns the authoritative incomplete transaction from directory scan.
func LoadIncomplete(projectRoot string) (*Runner, error) {
	txns, err := ScanIncompleteTxns(projectRoot)
	if err != nil {
		return nil, err
	}
	auth, err := ResolveAuthoritativeIncomplete(txns)
	if err != nil {
		return nil, err
	}
	if auth == nil {
		id, readErr := readCurrent(projectRoot)
		if readErr != nil {
			return nil, readErr
		}
		if id != "" {
			_ = clearCurrent(projectRoot)
			_, _ = tryRemoveStaleLock(projectRoot)
		}
		return nil, nil
	}
	return scannedToRunner(projectRoot, auth), nil
}

// RecoverIncomplete rolls back or resumes an interrupted transaction (idempotent).
func RecoverIncomplete(ctx context.Context, projectRoot string) error {
	if err := invokeTestHook("recovery", 0); err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.recover", projectRoot, err)
	}
	txn, err := LoadIncomplete(projectRoot)
	if err != nil || txn == nil {
		return err
	}
	switch txn.doc.State {
	case StateStaging, StateValidated:
		if fr := txn.Discard(); fr.HasCriticalCleanupFailure() {
			return finishResultError(fr)
		}
		return nil
	case StateCommitting:
		_, err := txn.Rollback(ctx)
		return err
	default:
		return nil
	}
}

func (r *Runner) applyPlanOp(ctx context.Context, index int, forward bool) error {
	if index < 0 || index >= len(r.doc.Plan) {
		return nil
	}
	op := r.doc.Plan[index]
	if !forward {
		return r.rollbackPlanOp(ctx, index)
	}
	if op.Progress == ProgressApplied {
		return nil
	}
	r.doc.Plan[index].Progress = ProgressApplying
	if r.doc.Plan[index].Phase == "" || r.doc.Plan[index].Phase == PhasePending {
		r.doc.Plan[index].Phase = PhasePublishStarted
	}
	if err := r.saveJournal(); err != nil {
		return err
	}
	if err := r.applyForward(ctx, index, r.doc.Plan[index]); err != nil {
		return err
	}
	r.doc.Plan[index].Progress = ProgressApplied
	if r.doc.Plan[index].Phase != PhaseApplied {
		r.doc.Plan[index].Phase = PhaseApplied
	}
	return r.saveJournal()
}

func (r *Runner) rollbackPlanOp(ctx context.Context, index int) error {
	if index < 0 || index >= len(r.doc.Plan) {
		return nil
	}
	op := r.doc.Plan[index]
	if op.Progress != ProgressApplied && op.Progress != ProgressApplying {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r.doc.Plan[index].Progress = ProgressRollingBack
	r.doc.Plan[index].Phase = PhaseRollbackStarted
	if err := r.saveJournal(); err != nil {
		return err
	}
	if backup := r.findBackupForPath(op.Path); backup != nil {
		if err := r.restoreBackup(*backup); err != nil {
			return err
		}
		r.doc.Plan[index].Phase = PhasePriorRestored
		if err := r.saveJournal(); err != nil {
			return err
		}
	}
	if err := r.cleanupPartialPublish(op); err != nil {
		return err
	}
	r.doc.Plan[index].Phase = PhaseRollbackComplete
	r.doc.Plan[index].Progress = ProgressRolledBack
	return r.saveJournal()
}

func (r *Runner) findBackupForPath(rel string) *Op {
	for i := range r.doc.Ops {
		if r.doc.Ops[i].Kind == OpBackup && r.doc.Ops[i].Path == rel {
			cp := r.doc.Ops[i]
			return &cp
		}
	}
	return nil
}

func (r *Runner) planHasAppliedPath(rel string) bool {
	for _, op := range r.doc.Plan {
		if op.Path != rel {
			continue
		}
		if op.Progress == ProgressApplied || op.Progress == ProgressApplying {
			return true
		}
	}
	return false
}

func (r *Runner) cleanupPartialPublish(op Op) error {
	switch op.Kind {
	case OpRename:
		if op.Backup != "" {
			staged := filepath.Join(r.Root, op.Backup)
			_ = os.RemoveAll(staged)
		}
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		_ = os.RemoveAll(live + ".mew-old")
	case OpMkdir:
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(live)
	case OpWrite, OpRemove:
		return nil
	default:
		return nil
	}
	return nil
}

func (r *Runner) markPlanPhase(path, phase string) {
	for i := range r.doc.Plan {
		if r.doc.Plan[i].Path == path {
			r.doc.Plan[i].Phase = phase
		}
	}
}

func (r *Runner) saveJournal() error {
	if r.doc != nil && r.doc.SchemaVersion < SchemaVersion {
		r.doc.SchemaVersion = SchemaVersion
	}
	data, err := Encode(r.doc)
	if err != nil {
		return err
	}
	gen, err := saveJournalGeneration(r.Root, r.journalGen, data)
	if err != nil {
		return err
	}
	r.journalGen = gen
	return nil
}

func (r *Runner) applyForward(ctx context.Context, planIndex int, op Op) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch op.Kind {
	case OpBackup:
		return nil
	case OpRename:
		return r.renameOp(planIndex, op, true)
	case OpWrite:
		return r.writeOp(planIndex, op)
	case OpRemove:
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(live)
	case OpMkdir:
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		return os.MkdirAll(live, 0o755)
	default:
		return apperr.New(apperr.Transaction, "transaction.commit", op.Kind, "unknown op kind")
	}
}

func (r *Runner) restoreBackup(op Op) error {
	live, err := GuardPath(r.ProjectRoot, op.Path)
	if err != nil {
		return err
	}
	hadPrior := op.HadPrior || op.Backup != ""
	if !hadPrior {
		return os.RemoveAll(live)
	}
	if op.Backup == "" {
		return nil
	}
	backup := filepath.Join(r.Root, op.Backup)
	if op.PriorKind == DestKindSymlink || op.PriorKind == DestKindJunction {
		target, err := os.ReadFile(backup)
		if err != nil {
			if os.IsNotExist(err) {
				return os.RemoveAll(live)
			}
			return apperr.Wrap(apperr.IO, "transaction.rollback", op.Path, err)
		}
		_ = os.RemoveAll(live)
		return os.Symlink(strings.TrimSpace(string(target)), live)
	}
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		return os.RemoveAll(live)
	}
	_ = os.RemoveAll(live)
	return restoreTree(backup, live)
}

func (r *Runner) renameOp(planIndex int, op Op, forward bool) error {
	live, err := GuardPath(r.ProjectRoot, op.Path)
	if err != nil {
		return err
	}
	src := filepath.Join(r.Root, op.Backup)
	if !forward {
		src, live = live, src
	}
	if isDir(src) || isDir(live) {
		return r.publishDirOp(planIndex, src, live)
	}
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.rename", live, err)
	}
	if err := fsx.PublishRename(src, live); err != nil {
		return err
	}
	r.doc.Plan[planIndex].Phase = PhaseNewFilePublished
	return r.saveJournal()
}

func (r *Runner) publishDirOp(planIndex int, stageDir, liveDir string) error {
	backup := liveDir + ".mew-old"
	if err := os.RemoveAll(backup); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.publish", backup, err)
	}
	if _, err := os.Stat(liveDir); err == nil {
		if err := os.Rename(liveDir, backup); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.publish", liveDir, err)
		}
		r.doc.Plan[planIndex].Phase = PhaseOldTreeMoved
		if err := r.saveJournal(); err != nil {
			return err
		}
	}
	if err := os.Rename(stageDir, liveDir); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			if restoreErr := os.Rename(backup, liveDir); restoreErr != nil {
				return apperr.Wrap(apperr.IO, "transaction.publish", liveDir, restoreErr)
			}
		}
		return apperr.Wrap(apperr.IO, "transaction.publish", stageDir, err)
	}
	r.doc.Plan[planIndex].Phase = PhaseNewTreePublished
	if err := r.saveJournal(); err != nil {
		return err
	}
	parent := filepath.Dir(liveDir)
	if err := fsx.SyncDir(parent); err != nil {
		return err
	}
	r.doc.Plan[planIndex].Phase = PhaseParentSynced
	if err := r.saveJournal(); err != nil {
		return err
	}
	if err := os.RemoveAll(backup); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.publish", backup, err)
	}
	return r.saveJournal()
}

func (r *Runner) writeOp(planIndex int, op Op) error {
	live, err := GuardPath(r.ProjectRoot, op.Path)
	if err != nil {
		return err
	}
	src := filepath.Join(r.Root, op.Backup)
	data, err := os.ReadFile(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.write", op.Path, err)
	}
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.write", live, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(live), ".tmp-*")
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.write", live, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "transaction.write", live, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "transaction.write", live, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.write", live, err)
	}
	r.doc.Plan[planIndex].Phase = PhaseNewFileWritten
	if err := r.saveJournal(); err != nil {
		return err
	}
	if err := fsx.PublishRename(tmpName, live); err != nil {
		return err
	}
	r.doc.Plan[planIndex].Phase = PhaseNewFilePublished
	return r.saveJournal()
}

// repairPartialNodeModules fixes interrupted node_modules rename choreography.
func repairPartialNodeModules(projectRoot string) error {
	live := filepath.Join(projectRoot, "node_modules")
	backup := live + ".mew-old"
	liveStat, liveErr := os.Stat(live)
	backupStat, backupErr := os.Stat(backup)
	if liveErr != nil && os.IsNotExist(liveErr) {
		if backupErr == nil && backupStat.IsDir() {
			return os.Rename(backup, live)
		}
		return nil
	}
	if backupErr == nil && backupStat.IsDir() && liveErr == nil && liveStat.IsDir() {
		_ = os.RemoveAll(backup)
	}
	return nil
}

func pathKind(path string) (kind, target string, hadPrior bool, err error) {
	info, statErr := os.Lstat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return DestKindNone, "", false, nil
		}
		return "", "", false, apperr.Wrap(apperr.IO, "transaction.pathkind", path, statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		tgt, readErr := os.Readlink(path)
		if readErr != nil {
			return "", "", false, apperr.Wrap(apperr.IO, "transaction.pathkind", path, readErr)
		}
		if fsx.IsJunction(path) {
			return DestKindJunction, tgt, true, nil
		}
		return DestKindSymlink, tgt, true, nil
	}
	if info.IsDir() {
		return DestKindDir, "", true, nil
	}
	return DestKindFile, "", true, nil
}

func sanitizeBackupName(rel string) string {
	s := strings.NewReplacer(string(filepath.Separator), "_", "..", "_").Replace(rel)
	if s == "" {
		return "backup"
	}
	return s
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func newTxnID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", apperr.Wrap(apperr.Internal, "transaction.id", "", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func writeCurrent(projectRoot, id string) error {
	return writeCurrentGeneration(projectRoot, id)
}

func readCurrent(projectRoot string) (string, error) {
	return readCurrentGeneration(projectRoot)
}

func clearCurrent(projectRoot string) error {
	dir := TxnRoot(projectRoot)
	_ = os.Remove(filepath.Join(dir, currentHeadName))
	err := os.Remove(CurrentPath(projectRoot))
	if err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "transaction.current", projectRoot, err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "transaction.current", projectRoot, readErr)
	}
	for _, ent := range entries {
		name := ent.Name()
		if strings.HasPrefix(name, "current.") && name != currentHeadName {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}
