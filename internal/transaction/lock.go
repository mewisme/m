package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

const (
	lockFileName       = "lock"
	lockSchemaVersion  = 2
	lockRetryInterval  = 25 * time.Millisecond
	lockMaxWaitDefault = 30 * time.Second
)

// LockDocument is persisted at .mew/txn/lock (schema v2).
type LockDocument struct {
	SchemaVersion int       `json:"schemaVersion,omitempty"`
	PID           int       `json:"pid"`
	ProcessStart  int64     `json:"processStart,omitempty"`
	TxnID         string    `json:"txnId"`
	CreatedAt     time.Time `json:"createdAt"`
	ProjectRoot   string    `json:"projectRoot,omitempty"`
	Hostname      string    `json:"hostname,omitempty"`
	// StartedAt is the v1 field; kept for stale-lock reads only.
	StartedAt time.Time `json:"startedAt,omitempty"`
}

// LockPath returns <project>/.mew/txn/lock.
func LockPath(projectRoot string) string {
	return filepath.Join(TxnRoot(projectRoot), lockFileName)
}

// AcquireProjectLock creates the project lock exclusively, waiting until ctx is done.
func AcquireProjectLock(ctx context.Context, projectRoot, txnID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", projectRoot, err)
	}
	path := LockPath(absRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	deadline := time.Now().Add(lockMaxWaitDefault)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	for {
		if err := ctx.Err(); err != nil {
			return apperr.Wrap(apperr.Cancelled, "transaction.lock", path, err)
		}
		if stale, staleErr := tryRemoveStaleLock(path); staleErr != nil {
			return staleErr
		} else if stale {
			if err := createExclusiveLock(path, absRoot, txnID); err == nil {
				return nil
			} else if !errors.Is(err, os.ErrExist) {
				return err
			}
		} else if err := createExclusiveLock(path, absRoot, txnID); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrExist) {
			return err
		}
		if time.Now().After(deadline) {
			return apperr.New(apperr.Transaction, "transaction.lock", path,
				"another install transaction is in progress")
		}
		timer := time.NewTimer(lockRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return apperr.Wrap(apperr.Cancelled, "transaction.lock", path, ctx.Err())
		case <-timer.C:
		}
	}
}

// ReleaseProjectLock removes the project lock when owned by this process and txnID.
// An empty txnID releases only when the on-disk lock matches this process identity.
func ReleaseProjectLock(projectRoot, txnID string) error {
	path := LockPath(projectRoot)
	doc, err := readLockDocument(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	if txnID != "" && doc.TxnID != txnID {
		return nil
	}
	if !processOwnsLock(doc) {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return nil
}

func createExclusiveLock(path, projectRoot, txnID string) error {
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.lock", path, err)
	}
	host, _ := os.Hostname()
	doc := LockDocument{
		SchemaVersion: lockSchemaVersion,
		PID:           pid,
		ProcessStart:  start,
		TxnID:         txnID,
		CreatedAt:     time.Now().UTC(),
		ProjectRoot:   projectRoot,
		Hostname:      host,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Transaction, "transaction.lock", path, err)
	}
	raw = append(raw, '\n')
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return nil
}

func readLockDocument(path string) (LockDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LockDocument{}, err
	}
	var doc LockDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return LockDocument{}, apperr.Wrap(apperr.Transaction, "transaction.lock", path, err)
	}
	if doc.SchemaVersion == 0 && !doc.StartedAt.IsZero() {
		doc.SchemaVersion = 1
	}
	return doc, nil
}

func tryRemoveStaleLock(path string) (bool, error) {
	doc, err := readLockDocument(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		if apperr.CodeOf(err) == apperr.Transaction {
			_ = os.Remove(path)
			return true, nil
		}
		return false, apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	if lockHolderAlive(doc) {
		return false, nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, apperr.Wrap(apperr.IO, "transaction.lock", path, err)
	}
	return true, nil
}

func lockHolderAlive(doc LockDocument) bool {
	if doc.PID <= 0 {
		return false
	}
	if doc.SchemaVersion >= lockSchemaVersion && doc.ProcessStart != 0 {
		return processIdentityAlive(doc.PID, doc.ProcessStart)
	}
	return processAlive(doc.PID)
}

func processOwnsLock(doc LockDocument) bool {
	pid, start, err := currentProcessIdentity()
	if err != nil {
		return false
	}
	if doc.PID != pid {
		return false
	}
	if doc.SchemaVersion >= lockSchemaVersion && doc.ProcessStart != 0 {
		return doc.ProcessStart == start
	}
	return true
}
