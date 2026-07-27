package transaction

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/m/internal/apperr"
)

// ScannedTxn is an incomplete transaction discovered by directory scan.
type ScannedTxn struct {
	ID         string
	Root       string
	State      string
	JournalGen int
	doc        *Document
}

// RecoverScannedOpts tunes automatic recovery during mutation preflight.
type RecoverScannedOpts struct {
	// SkipTakeover skips lock takeover when the caller already holds the project lock.
	SkipTakeover bool
}

// ScanIncompleteTxns walks .mew/txn/<id>/ and returns journals not committed or aborted.
func ScanIncompleteTxns(projectRoot string) ([]ScannedTxn, error) {
	dir := TxnRoot(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "transaction.scan", dir, err)
	}
	var out []ScannedTxn
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if name == lockDirName || strings.HasPrefix(name, "current") {
			continue
		}
		root := filepath.Join(dir, name)
		doc, loadErr := loadJournalGeneration(root)
		if loadErr != nil {
			return nil, loadErr
		}
		if doc == nil {
			continue
		}
		if doc.State == StateCommitted || doc.State == StateAborted {
			continue
		}
		out = append(out, ScannedTxn{
			ID:         name,
			Root:       root,
			State:      doc.State,
			JournalGen: currentJournalGen(root),
			doc:        doc,
		})
	}
	return out, nil
}

// ResolveAuthoritativeIncomplete picks one incomplete txn by state priority and journal generation.
func ResolveAuthoritativeIncomplete(txns []ScannedTxn) (*ScannedTxn, error) {
	if len(txns) == 0 {
		return nil, nil
	}
	var committing int
	var best *ScannedTxn
	for i := range txns {
		t := &txns[i]
		if t.State == StateCommitting {
			committing++
		}
		if best == nil || incompleteTxnRank(t) > incompleteTxnRank(best) {
			best = t
			continue
		}
		if incompleteTxnRank(t) == incompleteTxnRank(best) && t.JournalGen > best.JournalGen {
			best = t
		}
	}
	if committing > 1 {
		return nil, apperr.New(apperr.Integrity, "transaction.scan", "",
			"multiple incomplete transactions in committing state")
	}
	return best, nil
}

func incompleteTxnRank(t *ScannedTxn) int {
	switch t.State {
	case StateCommitting:
		return 3
	case StateValidated:
		return 2
	case StateStaging:
		return 1
	default:
		return 0
	}
}

// RecoverScanned idempotently rolls back or discards the authoritative incomplete transaction.
func RecoverScanned(ctx context.Context, projectRoot string, opts RecoverScannedOpts) error {
	if err := invokeTestHook("recovery", 0); err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.recover", projectRoot, err)
	}
	txns, err := ScanIncompleteTxns(projectRoot)
	if err != nil {
		return err
	}
	auth, err := ResolveAuthoritativeIncomplete(txns)
	if err != nil {
		return err
	}
	if auth == nil {
		return nil
	}
	if !opts.SkipTakeover {
		if err := TakeoverProjectLock(ctx, projectRoot, auth.ID); err != nil {
			return err
		}
	}
	run := scannedToRunner(projectRoot, auth)
	finishOpts := RecoveryFinishOpts(opts.SkipTakeover)
	switch auth.State {
	case StateStaging, StateValidated:
		if fr := run.Discard(finishOpts); fr.HasCriticalCleanupFailure() {
			return finishResultError(fr)
		}
		return nil
	case StateCommitting:
		_, err := run.Rollback(ctx, finishOpts)
		return err
	default:
		return nil
	}
}

// ScanCommittedStale finds committed transaction journals that still hold current or lock metadata.
func ScanCommittedStale(projectRoot string) ([]ScannedTxn, error) {
	dir := TxnRoot(projectRoot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.IO, "transaction.scan", dir, err)
	}
	currentID, _ := readCurrent(projectRoot)
	var out []ScannedTxn
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if name == lockDirName || strings.HasPrefix(name, "current") {
			continue
		}
		root := filepath.Join(dir, name)
		doc, loadErr := loadJournalGeneration(root)
		if loadErr != nil {
			return nil, loadErr
		}
		if doc == nil || doc.State != StateCommitted {
			continue
		}
		stale := currentID == name || lockHeldForTxn(projectRoot, name)
		if !stale {
			continue
		}
		out = append(out, ScannedTxn{
			ID:         name,
			Root:       root,
			State:      doc.State,
			JournalGen: currentJournalGen(root),
			doc:        doc,
		})
	}
	return out, nil
}

func lockHeldForTxn(projectRoot, txnID string) bool {
	lockDir := LockPath(projectRoot)
	data, err := os.ReadFile(filepath.Join(lockDir, lockOwnerFile))
	if err != nil {
		return false
	}
	doc, err := parseLockDocument(data)
	if err != nil {
		return false
	}
	return doc.TxnID == txnID
}

// RecoverCommittedCleanup clears stale metadata left after a committed install when post-commit cleanup failed.
func RecoverCommittedCleanup(ctx context.Context, projectRoot string) (int, error) {
	stale, err := ScanCommittedStale(projectRoot)
	if err != nil {
		return 0, err
	}
	var cleaned int
	for _, t := range stale {
		if err := ctx.Err(); err != nil {
			return cleaned, err
		}
		if err := TakeoverProjectLock(ctx, projectRoot, t.ID); err != nil {
			return cleaned, err
		}
		run := scannedToRunner(projectRoot, &t)
		fr := run.Finish(false, StandaloneFinishOpts())
		if fr.HasCriticalCleanupFailure() {
			return cleaned, finishResultError(fr)
		}
		cleaned++
	}
	return cleaned, nil
}

func scannedToRunner(projectRoot string, t *ScannedTxn) *Runner {
	return &Runner{
		ProjectRoot: projectRoot,
		ID:          t.ID,
		Root:        t.Root,
		doc:         t.doc,
		journalGen:  t.JournalGen,
	}
}

// BeginMutation is the single entrypoint for install-family mutations.
func BeginMutation(ctx context.Context, projectRoot string) (*Runner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if projectRoot == "" {
		return nil, apperr.New(apperr.Transaction, "transaction.begin", "", "empty project root")
	}
	id, err := newTxnID()
	if err != nil {
		return nil, err
	}
	if err := AcquireProjectLock(ctx, projectRoot, id); err != nil {
		return nil, err
	}
	if err := RecoverScanned(ctx, projectRoot, RecoverScannedOpts{SkipTakeover: true}); err != nil {
		_ = ReleaseProjectLock(projectRoot, id)
		return nil, err
	}
	txns, err := ScanIncompleteTxns(projectRoot)
	if err != nil {
		_ = ReleaseProjectLock(projectRoot, id)
		return nil, err
	}
	if len(txns) > 0 {
		_ = ReleaseProjectLock(projectRoot, id)
		return nil, apperr.New(apperr.Integrity, "transaction.begin", projectRoot,
			"incomplete transaction remains after recovery")
	}
	run, err := beginWithID(ctx, projectRoot, id)
	if err != nil {
		_ = ReleaseProjectLock(projectRoot, id)
		return nil, err
	}
	return run, nil
}

func beginWithID(ctx context.Context, projectRoot, id string) (*Runner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r := &Runner{
		ProjectRoot: projectRoot,
		ID:          id,
		Root:        filepath.Join(TxnRoot(projectRoot), id),
		doc: &Document{
			SchemaVersion: SchemaVersion,
			ID:            id,
			ProjectRoot:   projectRoot,
			State:         StateStaging,
		},
	}
	if err := os.MkdirAll(r.StagePath(), 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := os.MkdirAll(filepath.Join(r.Root, backupsDir), 0o755); err != nil {
		return nil, apperr.Wrap(apperr.IO, "transaction.begin", r.Root, err)
	}
	if err := r.saveJournal(); err != nil {
		return nil, err
	}
	if err := invokeTestHook("journal_created", 0); err != nil {
		return nil, apperr.Wrap(apperr.Transaction, "transaction.begin", r.ID, err)
	}
	if err := writeCurrent(r.ProjectRoot, id); err != nil {
		return nil, err
	}
	return r, nil
}
