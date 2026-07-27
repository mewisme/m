package transaction

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.ProjectRoot == "" {
		return apperr.New(apperr.Transaction, "transaction.begin", "", "empty project root")
	}
	id, err := newTxnID()
	if err != nil {
		return err
	}
	if err := AcquireProjectLock(r.ProjectRoot, id); err != nil {
		return err
	}
	r.ID = id
	r.Root = filepath.Join(TxnRoot(r.ProjectRoot), id)
	r.doc = &Document{
		SchemaVersion: SchemaVersion,
		ID:            id,
		ProjectRoot:   r.ProjectRoot,
		State:         StateStaging,
	}
	if err := os.MkdirAll(r.StagePath(), 0o755); err != nil {
		_ = ReleaseProjectLock(r.ProjectRoot)
		return apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := os.MkdirAll(filepath.Join(r.Root, backupsDir), 0o755); err != nil {
		_ = ReleaseProjectLock(r.ProjectRoot)
		return apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := r.saveJournal(); err != nil {
		_ = ReleaseProjectLock(r.ProjectRoot)
		return err
	}
	if err := invokeTestHook("journal_created", 0); err != nil {
		_ = ReleaseProjectLock(r.ProjectRoot)
		return apperr.Wrap(apperr.Transaction, "transaction.begin", r.ID, err)
	}
	if err := writeCurrent(r.ProjectRoot, id); err != nil {
		_ = ReleaseProjectLock(r.ProjectRoot)
		return err
	}
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
	} else if err := copyPath(live, backupAbs); err != nil {
		return err
	}
	r.doc.Ops = append(r.doc.Ops, op)
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
			_ = r.Rollback(ctx)
			return apperr.Wrap(apperr.Transaction, "transaction.commit", r.doc.Plan[i].Path, err)
		}
		if err := invokeTestHook("commit", i); err != nil {
			_ = r.Rollback(ctx)
			return apperr.Wrap(apperr.Transaction, "transaction.commit", r.doc.Plan[i].Path, err)
		}
		if err := r.applyPlanOp(ctx, i, true); err != nil {
			_ = r.Rollback(ctx)
			return err
		}
	}
	if err := invokeTestHook("pre_committed", 0); err != nil {
		_ = r.Rollback(ctx)
		return apperr.Wrap(apperr.Transaction, "transaction.commit", "", err)
	}
	r.doc.State = StateCommitted
	if err := r.saveJournal(); err != nil {
		return err
	}
	return invokeTestHook("committed", 0)
}

// Rollback restores backups and inverses applied forward ops.
func (r *Runner) Rollback(ctx context.Context) error {
	if r.doc == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := invokeTestHook("rollback", 0); err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.rollback", "", err)
	}
	_ = repairPartialNodeModules(r.ProjectRoot)
	for i := len(r.doc.Plan) - 1; i >= 0; i-- {
		op := r.doc.Plan[i]
		if op.Progress != ProgressApplied && op.Progress != ProgressApplying {
			continue
		}
		if err := invokeTestHook("rollback", i+1); err != nil {
			return apperr.Wrap(apperr.Transaction, "transaction.rollback", op.Path, err)
		}
		if err := r.applyPlanOp(ctx, i, false); err != nil {
			return err
		}
	}
	for i := len(r.doc.Ops) - 1; i >= 0; i-- {
		op := r.doc.Ops[i]
		if op.Kind != OpBackup {
			continue
		}
		if err := r.applyInverse(ctx, op); err != nil {
			return err
		}
	}
	r.doc.State = StateAborted
	if err := r.saveJournal(); err != nil {
		return err
	}
	_ = clearCurrent(r.ProjectRoot)
	_ = ReleaseProjectLock(r.ProjectRoot)
	return nil
}

// Finish clears the current pointer and optionally removes the txn dir.
func (r *Runner) Finish(keepJournal bool) error {
	if r.doc == nil {
		return nil
	}
	_ = clearCurrent(r.ProjectRoot)
	_ = ReleaseProjectLock(r.ProjectRoot)
	if keepJournal {
		return nil
	}
	return os.RemoveAll(r.Root)
}

// Discard removes an incomplete transaction without restoring backups.
func (r *Runner) Discard() error {
	_ = clearCurrent(r.ProjectRoot)
	_ = ReleaseProjectLock(r.ProjectRoot)
	if r.Root == "" {
		return nil
	}
	return os.RemoveAll(r.Root)
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

// LoadIncomplete reads current pointer and journal when state != committed.
func LoadIncomplete(projectRoot string) (*Runner, error) {
	id, err := readCurrent(projectRoot)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	root := filepath.Join(TxnRoot(projectRoot), id)
	doc, err := loadJournal(root)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		_ = clearCurrent(projectRoot)
		return nil, nil
	}
	if doc.State == StateCommitted || doc.State == StateAborted {
		_ = clearCurrent(projectRoot)
		_ = ReleaseProjectLock(projectRoot)
		return nil, nil
	}
	return &Runner{ProjectRoot: projectRoot, ID: id, Root: root, doc: doc}, nil
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
	case StateStaging:
		return txn.Discard()
	case StateValidated:
		return txn.Discard()
	case StateCommitting:
		return txn.Rollback(ctx)
	default:
		return nil
	}
}

func (r *Runner) applyPlanOp(ctx context.Context, index int, forward bool) error {
	if index < 0 || index >= len(r.doc.Plan) {
		return nil
	}
	op := r.doc.Plan[index]
	if forward {
		if op.Progress == ProgressApplied {
			return nil
		}
		r.doc.Plan[index].Progress = ProgressApplying
		if err := r.saveJournal(); err != nil {
			return err
		}
		if err := r.applyForward(ctx, op); err != nil {
			return err
		}
		r.doc.Plan[index].Progress = ProgressApplied
		return r.saveJournal()
	}
	if op.Progress != ProgressApplied && op.Progress != ProgressApplying {
		return nil
	}
	r.doc.Plan[index].Progress = ProgressRollingBack
	if err := r.saveJournal(); err != nil {
		return err
	}
	if err := r.applyInverse(ctx, op); err != nil {
		return err
	}
	r.doc.Plan[index].Progress = ProgressRolledBack
	return r.saveJournal()
}

func (r *Runner) saveJournal() error {
	data, err := Encode(r.doc)
	if err != nil {
		return err
	}
	name := JournalName
	if r.doc != nil && r.doc.SchemaVersion == 1 {
		name = JournalNameV1
	}
	path := filepath.Join(r.Root, name)
	tmp, err := os.CreateTemp(r.Root, ".journal.*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.journal", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "transaction.journal", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "transaction.journal", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.journal", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmpName, path); err2 != nil {
			return apperr.Wrap(apperr.IO, "transaction.journal", path, err2)
		}
	}
	return nil
}

func loadJournal(root string) (*Document, error) {
	for _, name := range []string{JournalName, JournalNameV1} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.IO, "transaction.load", root, err)
		}
		return Decode(data)
	}
	return nil, nil
}

func (r *Runner) applyForward(ctx context.Context, op Op) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch op.Kind {
	case OpBackup:
		return nil
	case OpRename:
		return r.renameOp(op, true)
	case OpWrite:
		return r.writeOp(op)
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

func (r *Runner) applyInverse(ctx context.Context, op Op) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch op.Kind {
	case OpBackup:
		return r.restoreBackup(op)
	case OpRename:
		return r.renameOp(op, false)
	case OpWrite:
		if op.Backup == "" {
			live, err := GuardPath(r.ProjectRoot, op.Path)
			if err != nil {
				return err
			}
			return os.Remove(live)
		}
		return r.writeOp(Op{Kind: OpWrite, Path: op.Path, Backup: op.Backup})
	case OpRemove:
		if op.Backup == "" {
			return nil
		}
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		backup := filepath.Join(r.Root, op.Backup)
		return copyPath(backup, live)
	case OpMkdir:
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		return os.RemoveAll(live)
	default:
		return apperr.New(apperr.Transaction, "transaction.rollback", op.Kind, "unknown op kind")
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
	return copyPath(backup, live)
}

func (r *Runner) renameOp(op Op, forward bool) error {
	live, err := GuardPath(r.ProjectRoot, op.Path)
	if err != nil {
		return err
	}
	src := filepath.Join(r.Root, op.Backup)
	if !forward {
		src, live = live, src
	}
	if isDir(src) || isDir(live) {
		return publishDir(src, live)
	}
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.rename", live, err)
	}
	if err := os.Rename(src, live); err != nil {
		_ = os.Remove(live)
		if err2 := os.Rename(src, live); err2 != nil {
			return apperr.Wrap(apperr.IO, "transaction.rename", live, err2)
		}
	}
	return nil
}

func (r *Runner) writeOp(op Op) error {
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
	if err := os.Rename(tmpName, live); err != nil {
		_ = os.Remove(live)
		if err2 := os.Rename(tmpName, live); err2 != nil {
			return apperr.Wrap(apperr.IO, "transaction.write", live, err2)
		}
	}
	return nil
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

// publishDir swaps stageDir into liveDir using rename choreography (Windows-safe).
func publishDir(stageDir, liveDir string) error {
	backup := liveDir + ".mew-old"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(liveDir); err == nil {
		if err := os.Rename(liveDir, backup); err != nil {
			return apperr.Wrap(apperr.IO, "transaction.publish", liveDir, err)
		}
	}
	if err := os.Rename(stageDir, liveDir); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			_ = os.Rename(backup, liveDir)
		}
		return apperr.Wrap(apperr.IO, "transaction.publish", stageDir, err)
	}
	_ = os.RemoveAll(backup)
	return nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", src, err)
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", src, err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()&0o777)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.copy", dst, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyPath(path, target)
	})
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
	dir := TxnRoot(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.current", dir, err)
	}
	path := CurrentPath(projectRoot)
	tmp, err := os.CreateTemp(dir, ".current.*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.current", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.WriteString(id + "\n"); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "transaction.current", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.current", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmpName, path); err2 != nil {
			return apperr.Wrap(apperr.IO, "transaction.current", path, err2)
		}
	}
	return nil
}

func readCurrent(projectRoot string) (string, error) {
	data, err := os.ReadFile(CurrentPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", apperr.Wrap(apperr.IO, "transaction.current", projectRoot, err)
	}
	id := strings.TrimSpace(string(data))
	return id, nil
}

func clearCurrent(projectRoot string) error {
	err := os.Remove(CurrentPath(projectRoot))
	if err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "transaction.current", projectRoot, err)
	}
	return nil
}
