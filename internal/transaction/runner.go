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
	r.ID = id
	r.Root = filepath.Join(TxnRoot(r.ProjectRoot), id)
	r.doc = &Document{
		SchemaVersion: SchemaVersion,
		ID:            id,
		ProjectRoot:   r.ProjectRoot,
		State:         StateStaging,
	}
	if err := os.MkdirAll(r.StagePath(), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := os.MkdirAll(filepath.Join(r.Root, backupsDir), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := r.saveJournal(); err != nil {
		return err
	}
	return writeCurrent(r.ProjectRoot, id)
}

// SetState updates journal state and persists.
func (r *Runner) SetState(state string) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.state", "", "transaction not begun")
	}
	r.doc.State = state
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
	if _, err := os.Stat(live); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.backup", rel, err)
	}
	backupRel := filepath.Join(backupsDir, filepath.Base(rel))
	backupAbs := filepath.Join(r.Root, backupRel)
	if err := copyPath(live, backupAbs); err != nil {
		return err
	}
	r.doc.Ops = append(r.doc.Ops, Op{Kind: OpBackup, Path: rel, Backup: backupRel})
	return r.saveJournal()
}

// Commit applies forward ops in order; on any error, rolls back.
func (r *Runner) Commit(ctx context.Context, extra []Op) error {
	if r.doc == nil {
		return apperr.New(apperr.Transaction, "transaction.commit", "", "transaction not begun")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ops := append(append([]Op{}, r.doc.Ops...), extra...)
	if err := r.SetState(StateCommitting); err != nil {
		return err
	}
	for i, op := range ops {
		if err := invokeTestHook("commit", i); err != nil {
			_ = r.Rollback(ctx)
			return apperr.Wrap(apperr.Transaction, "transaction.commit", op.Path, err)
		}
		if err := r.applyForward(ctx, op); err != nil {
			_ = r.Rollback(ctx)
			return err
		}
	}
	r.doc.Ops = ops
	r.doc.State = StateCommitted
	return r.saveJournal()
}

// Rollback restores backups in reverse order and sets state=aborted.
func (r *Runner) Rollback(ctx context.Context) error {
	if r.doc == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for i := len(r.doc.Ops) - 1; i >= 0; i-- {
		op := r.doc.Ops[i]
		if op.Kind != OpBackup {
			continue
		}
		if err := invokeTestHook("rollback", i); err != nil {
			return apperr.Wrap(apperr.Transaction, "transaction.rollback", op.Path, err)
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
	return nil
}

// Finish clears the current pointer and optionally removes the txn dir.
func (r *Runner) Finish(keepJournal bool) error {
	if r.doc == nil {
		return nil
	}
	_ = clearCurrent(r.ProjectRoot)
	if keepJournal {
		return nil
	}
	return os.RemoveAll(r.Root)
}

// Discard removes an incomplete transaction without restoring backups.
func (r *Runner) Discard() error {
	_ = clearCurrent(r.ProjectRoot)
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
	data, err := os.ReadFile(filepath.Join(root, JournalName))
	if err != nil {
		if os.IsNotExist(err) {
			_ = clearCurrent(projectRoot)
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "transaction.load", root, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	if doc.State == StateCommitted || doc.State == StateAborted {
		_ = clearCurrent(projectRoot)
		return nil, nil
	}
	return &Runner{ProjectRoot: projectRoot, ID: id, Root: root, doc: doc}, nil
}

func (r *Runner) saveJournal() error {
	data, err := Encode(r.doc)
	if err != nil {
		return err
	}
	path := filepath.Join(r.Root, JournalName)
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

func (r *Runner) applyForward(ctx context.Context, op Op) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch op.Kind {
	case OpBackup:
		return nil // backup already performed at RecordBackup
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
		if op.Backup == "" {
			return nil
		}
		live, err := GuardPath(r.ProjectRoot, op.Path)
		if err != nil {
			return err
		}
		backup := filepath.Join(r.Root, op.Backup)
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			_ = os.RemoveAll(live)
			return nil
		}
		_ = os.RemoveAll(live)
		return copyPath(backup, live)
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
